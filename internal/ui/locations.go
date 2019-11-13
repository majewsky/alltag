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
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/majewsky/alltag/internal/date"
	"github.com/majewsky/alltag/internal/db"
	"github.com/sapcc/go-bits/respondwith"
)

var tListLocations = tmpl("list-locations.html", `
	{{- if not . -}}
		<p class="flash flash-warning">You need to create at least one location first.</p>
	{{- end -}}
	<div class="table-container">
		<table class="table has-hover-highlight">
			<thead>
				<tr>
					<th>Name</th>
					<th class="actions"><a href="/locations/new">New location</button></th>
				</tr>
			</thead>
			<tbody>
				{{- if . -}}
					{{- range . -}}
						<tr>
							<td><a href="/locations/{{ .ID }}">{{ .Label }}</a></td>
							<td class="actions"><a href="/locations/{{ .ID }}/edit">Edit</a> · <a href="/locations/{{ .ID }}/delete">Delete</a></td>
						</tr>
					{{- end -}}
				{{- else -}}
					<tr>
						<td colspan="2" class="text-muted text-center">No entries</td>
					</tr>
				{{- end -}}
			</tbody>
		</table>
	</div>
`)

func (h *handler) ListLocations(w http.ResponseWriter, r *http.Request) {
	locations, err := h.AllLocations(r)
	if respondwith.ErrorText(w, err) {
		return
	}

	Page{
		Title: "Manage locations",
		Navigation: []BreadcrumbItem{
			{URL: "/locations", Label: "Locations", Current: true},
		},
		Template: tListLocations,
		Data:     locations,
	}.WriteTo(w)
}

func (h *handler) FindLocationFromRequest(w http.ResponseWriter, r *http.Request) *db.Location {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if respondwith.ErrorText(w, err) {
		return nil
	}
	var location db.Location
	err = h.dbi.SelectOne(&location,
		`SELECT * FROM locations WHERE id = $1 AND username = $2`,
		id, currentUser(r),
	)
	if err == sql.ErrNoRows {
		http.Error(w, "Not found", http.StatusNotFound)
		return nil
	}
	if respondwith.ErrorText(w, err) {
		return nil
	}
	return &location
}

var tShowLocation = tmpl("show-location.html", `
	<div class="button-row">
		<a class="button" href="/locations/{{.Location.ID}}/edit">Edit location</a>
		{{if .Tasks -}}
			<button disabled>Delete location</button>
		{{- else -}}
			<a class="button" href="/locations/{{.Location.ID}}/delete">Delete location</a>
		{{- end -}}
	</div>
	<div class="table-container">
		<table class="table responsive has-hover-highlight">
			<thead>
				<tr>
					<th class="grow-column">Task</th>
					<th>Class</th>
					<th>Priority</th>
					<th>Starts at</th>
					<th>Due at</th>
					<th>Actions</th>
				</tr>
			</thead>
			<tbody>
				{{- if .Tasks -}}
					{{- range .Tasks -}}
						<tr class="{{if dateGreaterThan .StartsAt $.DateNow}}text-muted{{end}}">
							<td class="grow-column" data-label="Label">{{.Label}}</td>
							<td data-label="Class">{{.Class}}</td>
							<td class="nobr-column" data-label="Priority">{{.InitialPriority}} -> {{.FinalPriority}}</td>
							<td class="nobr-column" data-label="Starts at">{{.StartsAt}}</td>
							<td class="nobr-column" data-label="Due at">{{.DueAt}}</td>
							<td class="actions"><a href="/tasks/{{ .ID }}/edit">Edit</a></td>
						</tr>
					{{- end -}}
				{{- else -}}
					<tr>
						<td colspan="5" class="text-muted text-center">No entries</td>
					</tr>
				{{- end -}}
			</tbody>
		</table>
	</div>
`)

func (h *handler) ShowLocation(w http.ResponseWriter, r *http.Request) {
	location := h.FindLocationFromRequest(w, r)
	if location == nil {
		return
	}

	var tasks []db.Task
	_, err := h.dbi.Select(&tasks,
		`SELECT t.* FROM tasks t JOIN task_locations l ON t.id = l.task_id WHERE l.location_id = $1 AND t.username = $2`,
		location.ID, currentUser(r),
	)
	if respondwith.ErrorText(w, err) {
		return
	}
	now := time.Now()
	sort.Slice(tasks, func(i, j int) bool {
		//note the sign: this sorts into reverse order (i.e. highest current priority on top)
		return tasks[i].SortOrder(now) > tasks[j].SortOrder(now)
	})

	Page{
		Title: "Show location",
		Navigation: []BreadcrumbItem{
			{URL: "/locations", Label: "Locations"},
			{URL: r.URL.Path, Label: location.Label, Current: true},
		},
		Template: tShowLocation,
		Data: struct {
			Location db.Location
			Tasks    []db.Task
			DateNow  date.Date
		}{*location, tasks, date.Now()},
	}.WriteTo(w)
}

var tNewOrEditLocation = tmpl("edit-location.html", `
	<form method="POST" action="/locations/{{if .}}{{.ID}}/edit{{else}}new{{end}}">
		<div class="form-row">
			<label for="label">Label</label>
			<input required type="text" name="label" id="label" value="{{if .}}{{.Label}}{{end}}" />
		</div>
		<div class="button-row">
			<button type="submit">{{if .}}Save{{else}}Create{{end}}</button>
		</div>
	</form>
`)

func (h *handler) NewOrEditLocation(w http.ResponseWriter, r *http.Request) {
	var location *db.Location
	if _, hasID := mux.Vars(r)["id"]; hasID {
		location = h.FindLocationFromRequest(w, r)
		if location == nil {
			return
		}
	}

	var nav []BreadcrumbItem
	if location == nil {
		nav = []BreadcrumbItem{
			{URL: "/locations", Label: "Locations"},
			{URL: r.URL.Path, Label: "New", Current: true},
		}
	} else {
		nav = []BreadcrumbItem{
			{URL: "/locations", Label: "Locations"},
			{URL: fmt.Sprintf("/locations/%d", location.ID), Label: location.Label},
			{URL: r.URL.Path, Label: "Edit", Current: true},
		}
	}

	p := Page{
		Title:      "Edit location",
		Navigation: nav,
		Template:   tNewOrEditLocation,
		Data:       location,
	}
	if location == nil {
		p.Title = "Add location"
	}
	p.WriteTo(w)
}

func (h *handler) CreateOrUpdateLocation(w http.ResponseWriter, r *http.Request) {
	_, isUpdate := mux.Vars(r)["id"]

	var location *db.Location
	if isUpdate {
		location = h.FindLocationFromRequest(w, r)
		if location == nil {
			return
		}
	} else {
		location = &db.Location{
			UserName: currentUser(r),
		}
	}

	err := r.ParseForm()
	if respondwith.ErrorText(w, err) {
		return
	}

	location.Label = r.PostForm.Get("label")
	if location.Label == "" {
		http.Error(w, "label may not be empty", http.StatusBadRequest)
		return
	}

	if isUpdate {
		_, err = h.dbi.Update(location)
	} else {
		err = h.dbi.Insert(location)
	}
	if respondwith.ErrorText(w, err) {
		return
	}
	http.Redirect(w, r, "/locations", http.StatusSeeOther)
}

var tDeleteLocation = tmpl("delete-location.html", `
	<form class="contains-body-text" method="POST" action="/locations/{{.ID}}/delete">
		<p>Really delete the location <strong>{{.Label}}</strong>? This cannot be undone.</p>
		<div class="button-row">
			<button type="submit">Delete permanently</button>
		</div>
	</form>
`)

func (h *handler) AskDeleteLocation(w http.ResponseWriter, r *http.Request) {
	location := h.FindLocationFromRequest(w, r)
	if location == nil {
		return
	}

	Page{
		Title: "Delete location",
		Navigation: []BreadcrumbItem{
			{URL: "/locations", Label: "Locations"},
			{URL: fmt.Sprintf("/locations/%d", location.ID), Label: location.Label},
			{URL: r.URL.Path, Label: "Delete", Current: true},
		},
		Template: tDeleteLocation,
		Data:     location,
	}.WriteTo(w)
}

func (h *handler) DeleteLocation(w http.ResponseWriter, r *http.Request) {
	location := h.FindLocationFromRequest(w, r)
	if location == nil {
		return
	}

	_, err := h.dbi.Delete(location)
	if respondwith.ErrorText(w, err) {
		return
	}
	http.Redirect(w, r, "/locations", http.StatusSeeOther)
}
