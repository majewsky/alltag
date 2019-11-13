/*******************************************************************************
*
* Copyright 2019 Stefan Majewsky <majewsky@gmx.net>
*
* This program is free software: you can redistribute it and/or modify it under
* the terms of the GNU General Public License as published by the Free Software
* Foundation, either version 3 of the License, or (at your option) any later
* version.
*
* This program is distributed in the hope that it will be useful, but WITHOUT ANY
* WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR
* A PARTICULAR PURPOSE. See the GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License along with
* this program. If not, see <http://www.gnu.org/licenses/>.
*
*******************************************************************************/

package auth

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"

	"github.com/sapcc/go-bits/logg"
	"gopkg.in/ldap.v3"
)

//LDAPConfig contains all the configuration options for the LDAP auth driver.
type LDAPConfig struct {
	ServerURL    url.URL
	BindDN       string
	BindPassword string
	SearchBaseDN string
	SearchFilter string
}

type ldapDriver struct {
	cfg   LDAPConfig
	conn  *ldap.Conn
	mutex *sync.Mutex
}

//NewLDAPDriver initializes the LDAP auth driver.
func NewLDAPDriver(cfg LDAPConfig) (Driver, error) {
	conn, err := ldap.DialURL(cfg.ServerURL.String())
	if err != nil {
		return nil, err
	}

	if cfg.ServerURL.Scheme == "ldap" {
		host, _, err := net.SplitHostPort(cfg.ServerURL.Host)
		if err != nil {
			host = cfg.ServerURL.Host
		}
		err = conn.StartTLS(&tls.Config{ServerName: host})
		if err != nil {
			return nil, err
		}
	}

	err = conn.Bind(cfg.BindDN, cfg.BindPassword)
	return &ldapDriver{cfg, conn, &sync.Mutex{}}, err
}

//this list generated with `perl -E 'print chr for 32..126' | tr -d 0-9A-Za-z`
const allASCIISymbols = " !\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"

func (d *ldapDriver) CheckLogin(userName, password string) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	//disallow ASCII symbols in username (i.e. all non-ASCII characters are
	//allowed, but for ASCII, only letters and numbers) to avoid collision with
	//special characters used in encodings (e.g. LDAP search filter)
	if strings.ContainsAny(userName, allASCIISymbols) {
		return false
	}

	sr, err := d.conn.Search(&ldap.SearchRequest{
		BaseDN:       d.cfg.SearchBaseDN,
		Scope:        ldap.ScopeWholeSubtree,
		DerefAliases: ldap.NeverDerefAliases,
		SizeLimit:    2, //so that we can detect (and fail) when multiple users match
		Filter:       fmt.Sprintf(d.cfg.SearchFilter, userName),
		Attributes:   []string{"dn"},
	})
	if err != nil {
		logg.Fatal("unexpected error while searching in LDAP: " + err.Error())
	}

	//If the user does not exist, attempt to bind anyway, but record that we're
	//going to fail regardless of the result of this bind operation. This avoids
	//a timing side-channel, where an attacker could infer whether a user exists
	//or not from how long a request to Alltag takes.
	var (
		userExists bool
		userDN     string
	)
	if len(sr.Entries) == 1 {
		userExists = true
		userDN = sr.Entries[0].DN
	} else {
		userExists = false
		//make up a somewhat plausible DN to do the bogus bind request with
		userDN = fmt.Sprintf("uid=%s,%s", userName, d.cfg.SearchBaseDN)
	}

	//validate user password
	err = d.conn.Bind(userDN, password)
	authOK := err == nil && userExists

	//re-bind as service user to execute next search request
	err = d.conn.Bind(d.cfg.BindDN, d.cfg.BindPassword)
	if err != nil {
		logg.Fatal("unexpected error while re-binding LDAP service user: " + err.Error())
	}

	return authOK
}
