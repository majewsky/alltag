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

package main

import (
	"bytes"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/majewsky/alltag/build/bindata"
	"github.com/majewsky/alltag/internal/auth"
	"github.com/majewsky/alltag/internal/db"
	"github.com/majewsky/alltag/internal/ui"
	_ "github.com/majewsky/xyrillian.css"
	"github.com/sapcc/go-bits/logg"
)

type loggDebug struct{}

func (loggDebug) Printf(format string, values ...interface{}) {
	logg.Debug(format, values...)
}

func main() {
	dbi, err := db.Init(mustGetenv("ALLTAG_DB_URI"))
	must(err)

	logg.ShowDebug, _ = strconv.ParseBool(os.Getenv("ALLTAG_DEBUG"))
	if logg.ShowDebug {
		dbi.TraceOn("SQL: ", loggDebug{})
	}

	handler := ui.NewHandler(dbi)
	handler = authenticateUsers(handler)
	handler = addSecurityHeaders(handler)
	http.Handle("/", handler)

	//the static files are not protected by authentication - otherwise the
	//browser cannot load the JS source maps
	http.HandleFunc("/static/", serveStaticFiles)

	listenAddress := getenvOrDefault("ALLTAG_LISTEN_ADDRESS", "127.0.0.1:8080")
	logg.Info("listening on %s...", listenAddress)
	must(http.ListenAndServe(listenAddress, nil))
}

////////////////////////////////////////////////////////////////////////////////
// HTTP handler that generates index.html

const indexHTML = `<!DOCTYPE html>
<html>
	<head>
		<meta charset="utf-8">
		<meta http-equiv="X-UA-Compatible" content="IE=edge" />
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<title>Alltag</title>
		<link rel="stylesheet" type="text/css" href="/static/{alltag.css}" />
	</head>
	<body>
		<p id="noscript">Please enable JavaScript to use this application.</p>
		<script src="/static/{redom.min.js}"></script>
		<script src="/static/{alltag.js}"></script>
	</body>
</html>`

var staticFileRefRx = regexp.MustCompile(`\{.+?\}`)

func serveIndexHTML(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	//rewrite all /static/something references in indexHTML into the corresponding hashed artifacts
	content := []byte(staticFileRefRx.ReplaceAllStringFunc(indexHTML, func(input string) string {
		input = strings.TrimPrefix(input, "{")
		input = strings.TrimSuffix(input, "}")
		return resolveAssetName(input)
	}))

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(content)), 10))
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

var hashedFileNameRx = regexp.MustCompile(`^(.*)-[0-9a-f]{64}(\..*)$`)

func resolveAssetName(input string) string {
	for _, hashedName := range bindata.AssetNames() {
		nameWithoutHash := hashedFileNameRx.ReplaceAllString(hashedName, "$1$2")
		if nameWithoutHash == input {
			return hashedName
		}
	}
	return input
}

////////////////////////////////////////////////////////////////////////////////
// HTTP handler that serves /static/

func serveStaticFiles(w http.ResponseWriter, r *http.Request) {
	assetPath := strings.TrimPrefix(r.URL.Path, "/")
	assetPath = strings.TrimPrefix(assetPath, "static/")
	assetBytes, err := bindata.Asset(assetPath)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	assetInfo, err := bindata.AssetInfo(assetPath)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	//add Cache-Control header for hashed artifacts
	if hashedFileNameRx.MatchString(assetPath) {
		w.Header().Set("Cache-Control", "public, immutable")
	}

	//add SourceMap header for JS files that have a source map
	if strings.HasSuffix(assetPath, ".js") {
		sourceMapPath := assetPath + ".map"
		_, err := bindata.AssetInfo(sourceMapPath)
		if err == nil {
			w.Header().Set("SourceMap", "/static/"+sourceMapPath)
		}
	}

	http.ServeContent(w, r, path.Base(assetPath), assetInfo.ModTime(), bytes.NewReader(assetBytes))
}

////////////////////////////////////////////////////////////////////////////////
// HTTP middlewares

func authenticateUsers(h http.Handler) http.Handler {
	ldapServerURL, err := url.Parse(mustGetenv("ALLTAG_LDAP_URI"))
	if err != nil {
		logg.Fatal("cannot parse ALLTAG_LDAP_URI: %s", err.Error())
	}

	driver, err := auth.NewLDAPDriver(auth.LDAPConfig{
		ServerURL:    *ldapServerURL,
		BindDN:       mustGetenv("ALLTAG_LDAP_BIND_DN"),
		BindPassword: mustGetenv("ALLTAG_LDAP_BIND_PASSWORD"),
		SearchBaseDN: mustGetenv("ALLTAG_LDAP_SEARCH_BASE_DN"),
		SearchFilter: getenvOrDefault("ALLTAG_LDAP_SEARCH_FILTER", "(uid=%s)"),
	})
	must(err)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userName, password, ok := r.BasicAuth()
		if ok {
			ok = driver.CheckLogin(userName, password)
		}

		if ok {
			r.Header.Set("X-Alltag-Username", userName)
			h.ServeHTTP(w, r)
		} else {
			w.Header().Set("Www-Authenticate", `Basic realm="Alltag", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	})
}

func addSecurityHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr := w.Header()
		hdr.Set("X-Frame-Options", "SAMEORIGIN")
		hdr.Set("X-XSS-Protection", "1; mode=block")
		hdr.Set("X-Content-Type-Options", "nosniff")
		hdr.Set("Referrer-Policy", "no-referrer")
		hdr.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:;")
		h.ServeHTTP(w, r)
	})
}

////////////////////////////////////////////////////////////////////////////////
// utilities

func must(err error) {
	if err != nil {
		logg.Fatal(err.Error())
	}
}

func mustGetenv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		logg.Fatal("missing environment variable: %s", key)
	}
	return val
}

func getenvOrDefault(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}
