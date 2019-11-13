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

package ui

import (
	"bytes"
	"html/template"
	"net/http"
	"regexp"
	"strconv"

	"github.com/majewsky/alltag/build/bindata"
	"github.com/majewsky/alltag/internal/date"
	"github.com/sapcc/go-bits/respondwith"
)

func currentUser(r *http.Request) string {
	//This header was set by the authentication middleware in main.go.
	return r.Header.Get("X-Alltag-Username")
}

////////////////////////////////////////////////////////////////////////////////
// helpers for templates

func dateGreaterThan(lhs, rhs date.Date) bool {
	return lhs.After(rhs)
}

var tmplFuncMap = template.FuncMap{
	"dateGreaterThan": dateGreaterThan,
}

//ensure that goimports does not replace html/template with text/template
const _ template.HTML = ""

func tmpl(name, input string) *template.Template {
	return template.Must(template.New(name).Funcs(tmplFuncMap).Parse(input))
}

////////////////////////////////////////////////////////////////////////////////
// general page layout

var tPage = tmpl("page.html", `<!DOCTYPE html>
<html>
	<head>
		<meta charset="utf-8">
		<meta http-equiv="X-UA-Compatible" content="IE=edge" />
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<title>{{ .Title }} - Alltag</title>
		<link rel="stylesheet" type="text/css" href="/static/{{ index .Assets "alltag.css" }}" />
	</head>
	<body>
		<nav id="nav">
			<div id="nav-bar">
				<div id="nav-title"></div>
				<a id="nav-fold" href="#">
					<span>Close menu</span>
				</a>
				<a id="nav-unfold" href="#nav">
					<span>{{ .Title }}</span>
				</a>
				<div class="nav-area" id="nav-left">
					{{- range $idx, $item := .Navigation -}}
						{{- if gt $idx 0 }}<div class="breadcrumb-arrow">&gt;</div>{{ end -}}
						<a class="nav-item {{if gt $idx 0}}nav-level-{{$idx}}{{end}} {{if $item.Current}}nav-item-current{{end}}" href="{{ $item.URL }}">{{$item.Label}}</a>
					{{- end -}}
				</div>
				<div class="nav-area" id="nav-right"></div>
			</div>
		</nav>
		<main class="{{if .ContainsBodyText}}contains-body-text{{end}}">{{ .Content }}</main>
		<footer>
			<p>
				Admin:
				<a href="/locations">Manage locations</a>
			</p>
		</footer>
		<script src="/static/{{ index .Assets "alltag.js" }}"></script>
	</body>
</html>`)

var assetPaths = parseAssetPaths()

func parseAssetPaths() map[string]string {
	//create a map that adds hashes to the asset filenames, e.g.
	//"alltag.css" -> "alltag-0bc18...1a08f.css"
	result := make(map[string]string)
	rx := regexp.MustCompile(`^(.*)-[0-9a-f]{64}(\..*)$`)
	for _, name := range bindata.AssetNames() {
		nameWithoutHash := rx.ReplaceAllString(name, "$1$2")
		result[nameWithoutHash] = name
	}
	return result
}

//BreadcrumbItem appears in type Page.
type BreadcrumbItem struct {
	URL     string
	Label   string
	Current bool
}

//Page describes a single HTML page.
type Page struct {
	Status           int //defaults to http.StatusOK
	Title            string
	Navigation       []BreadcrumbItem
	ContainsBodyText bool
	Template         *template.Template
	Data             interface{}
}

//WriteTo answers the HTTP request with this page.
func (p Page) WriteTo(w http.ResponseWriter) {
	var buf bytes.Buffer
	err := p.Template.Execute(&buf, p.Data)
	if respondwith.ErrorText(w, err) {
		return
	}

	navWithInitial := append([]BreadcrumbItem{
		{URL: "/", Label: "Alltag", Current: len(p.Navigation) == 0},
	}, p.Navigation...)

	pageData := struct {
		Title            string
		Navigation       []BreadcrumbItem
		ContainsBodyText bool
		Content          template.HTML
		Assets           map[string]string
	}{p.Title, navWithInitial, p.ContainsBodyText, template.HTML(buf.String()), assetPaths}

	buf.Reset()
	err = tPage.Execute(&buf, pageData)
	if respondwith.ErrorText(w, err) {
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	w.Header().Set("Content-Length", strconv.FormatInt(int64(buf.Len()), 10))
	if p.Status != 0 {
		w.WriteHeader(p.Status)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	w.Write(buf.Bytes())
}
