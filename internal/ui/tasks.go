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
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/majewsky/alltag/internal/date"
	"github.com/majewsky/alltag/internal/db"
	"github.com/sapcc/go-bits/respondwith"
)

var tShowTask = tmpl("show-task.html", `
	<div class="selected-task">
		<p>Do this:</p>
		<p class="task-description">{{.Label}}</p>
	</div>
	<div class="button-row">
		<a class="button" href="/tasks/{{.ID}}/close">Done!</a>
	</div>
`)

func (h *handler) ShowTask(w http.ResponseWriter, r *http.Request) {
	task := h.FindTaskFromRequest(w, r)
	if task == nil {
		return
	}

	Page{
		Title: "Show task",
		Navigation: []BreadcrumbItem{
			{URL: "/tasks", Label: "Tasks"},
			{URL: r.URL.Path, Label: fmt.Sprintf("#%d", task.ID), Current: true},
			{URL: fmt.Sprintf("/tasks/%d/edit", task.ID), Label: "Edit"},
		},
		Template: tShowTask,
		Data:     task,
	}.WriteTo(w)
}

var tNewTask = tmpl("new-task.html", `
	<form method="POST" action="/tasks/new">
		<div class="flash flash-primary">
			Remember to <strong>break down your work into small pieces:</strong> Enter the smallest-possible first step that can be taken towards your goal.
		</div>
		<div class="form-row">
			<label for="label">Label</label>
			<input required type="text" name="label" id="label" />
		</div>
		<div class="button-row">
			<button type="submit" name="next" value="home">Save</button>
			<button type="submit" name="next" value="edit">Classify now</button>
		</div>
	</form>
`)

func (h *handler) NewTask(w http.ResponseWriter, r *http.Request) {
	Page{
		Title: "Add task",
		Navigation: []BreadcrumbItem{
			{URL: "/tasks", Label: "Tasks"},
			{URL: "/tasks/new", Label: "New", Current: true},
		},
		Template: tNewTask,
		Data:     nil,
	}.WriteTo(w)
}

func (h *handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if respondwith.ErrorText(w, err) {
		return
	}

	task := db.Task{
		Label:    r.PostForm.Get("label"),
		UserName: currentUser(r),
		StartsAt: date.Epoch, //zero value
		DueAt:    date.Epoch, //zero value
	}
	if task.Label == "" {
		http.Error(w, "label may not be empty", http.StatusBadRequest)
		return
	}
	err = h.dbi.Insert(&task)
	if respondwith.ErrorText(w, err) {
		return
	}

	nextURL := "/"
	if r.PostForm.Get("next") == "edit" {
		nextURL = fmt.Sprintf("/tasks/%d/edit", task.ID)
	}
	http.Redirect(w, r, nextURL, http.StatusSeeOther)
}

var tEditTask = tmpl("edit-task.html", `
	<form method="POST" action="/tasks/{{.Task.ID}}/edit">
		<div class="form-row">
			<label for="label">Label</label>
			<input required type="text" name="label" id="label" value="{{.Task.Label}}" />
		</div>
		<div class="form-row">
			<label for="task_class">Class</label>
			<select name="task_class" id="task_class" required {{if .IsClassified}}data-initial-value="{{.Task.Class}}"{{end}}>
				<option value="">-- Select --</option>
				<option value="mental">Mental</option>
				<option value="physical">Physical</option>
			</select>
		</div>
		<div class="form-row">
			<label>Locations</label>
			<div class="item-list">
				{{- range .Locations -}}
					<input type="checkbox" name="location_ids" id="location-{{ .ID }}" value="{{ .ID }}" {{if index $.IsTaskLocation .ID}}checked{{end}} />
					<label for="location-{{ .ID }}">{{.Label}}</label>
				{{- end -}}
			</div>
		</div>
		<div class="side-by-side">
			<div class="form-row">
				<label for="initial_priority">Initial priority</label>
				<select name="initial_priority" id="initial_priority" required {{if .IsClassified}}data-initial-value="{{.Task.InitialPriority}}"{{end}}>
					<option value="">-- Select --</option>
					<option value="0">Low</option>
					<option value="1">Normal</option>
					<option value="2">High</option>
					<option value="3">Critical</option>
				</select>
			</div>
			<div class="form-row">
				<label for="final_priority">Final priority</label>
				<select name="final_priority" id="final_priority" required {{if .IsClassified}}data-initial-value="{{.Task.FinalPriority}}"{{end}}>
					<option value="">-- Select --</option>
					<option value="0">Low</option>
					<option value="1">Normal</option>
					<option value="2">High</option>
					<option value="3">Critical</option>
				</select>
			</div>
		</div>
		<div class="side-by-side">
			<div class="form-row">
				<label for="starts_at">Starts at</label>
				<input required type="date" readonly value="{{.Task.StartsAt}}" />
			</div>
			<div class="form-row">
				<label for="due_at">Due at</label>
				<input required type="date" name="due_at" id="due_at" value="{{.Task.DueAt}}" />
			</div>
		</div>
		{{if .Task.RecurrenceDays}}
			<input type="checkbox" id="has_recurrence" name="has_recurrence" value="true" class="for-fieldset" {{if .Task.RecurrenceDays}}checked{{end}} />
			<fieldset>
				<label for="has_recurrence">Configure recurrence</label>
				<div class="form-row">
					<label for="recurrence_days">Recurrence interval (in days)</label>
					<input type="number" name="recurrence_days" id="recurrence_days" value="{{.Task.RecurrenceDays}}" min="0" step="1" />
				</div>
			</fieldset>
		{{end}}
		<div class="button-row">
			<button type="submit">Save</button>
		</div>
	</form>
`)

func (h *handler) EditTask(w http.ResponseWriter, r *http.Request) {
	task := h.FindTaskFromRequest(w, r)
	if task == nil {
		return
	}

	locations, err := h.AllLocations(r)
	if respondwith.ErrorText(w, err) {
		return
	}
	isTaskLocation, err := h.FindTaskLocations(*task)
	if respondwith.ErrorText(w, err) {
		return
	}

	//for unclassified tasks, show the StartsAt date that UpdateTask() will enter
	if !task.IsClassified() {
		task.StartsAt = date.Now()
	}

	data := struct {
		Task           db.Task
		Locations      []db.Location
		IsClassified   bool
		IsTaskLocation map[int64]bool
	}{*task, locations, task.IsClassified(), isTaskLocation}
	Page{
		Title: "Edit task",
		Navigation: []BreadcrumbItem{
			{URL: "/tasks", Label: "Tasks"},
			{URL: fmt.Sprintf("/tasks/%d", task.ID), Label: fmt.Sprintf("#%d", task.ID)},
			{URL: r.URL.Path, Label: "Edit", Current: true},
		},
		Template: tEditTask,
		Data:     data,
	}.WriteTo(w)
}

func (h *handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if respondwith.ErrorText(w, err) {
		return
	}
	task := h.FindTaskFromRequest(w, r)
	if task == nil {
		return
	}

	locations, err := h.AllLocations(r)
	if respondwith.ErrorText(w, err) {
		return
	}
	isValidLocationID := make(map[int64]bool)
	for _, loc := range locations {
		isValidLocationID[loc.ID] = true
	}
	isTaskLocation, err := h.FindTaskLocations(*task)
	if respondwith.ErrorText(w, err) {
		return
	}

	//do everything in a transaction to enable easy rollback
	tx, err := h.dbi.Begin()
	if respondwith.ErrorText(w, err) {
		return
	}

	//update task attributes
	task.Label = r.PostForm.Get("label")
	if task.Label == "" {
		http.Error(w, "label may not be empty", http.StatusBadRequest)
		return
	}

	class := db.TaskClass(r.PostForm.Get("task_class"))
	if !db.IsTaskClass[class] {
		msg := fmt.Sprintf("invalid task class: %q", class)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	task.Class = &class

	task.InitialPriority, err = parsePriority(r.PostForm.Get("initial_priority"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	task.FinalPriority, err = parsePriority(r.PostForm.Get("final_priority"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if task.InitialPriority >= task.FinalPriority {
		http.Error(w, "final priority must be higher than initial priority", http.StatusBadRequest)
		return
	}

	task.RecurrenceDays, err = parseRecurrenceDays(r.PostForm)
	if err != nil {
		msg := fmt.Sprintf("invalid recurrence_days value: %q", r.PostForm.Get("recurrence_days"))
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if task.StartsAt == date.Epoch {
		//StartsAt is set during initial classification
		task.StartsAt = date.Now()
	}
	task.DueAt, err = date.Parse(r.PostForm.Get("due_at"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if task.DueAt.Before(date.Now()) {
		http.Error(w, "due date cannot be in the past", http.StatusBadRequest)
		return
	}
	if !task.DueAt.After(task.StartsAt) {
		http.Error(w, "due date must occur after start date", http.StatusBadRequest)
		return
	}

	_, err = tx.Update(task)
	if respondwith.ErrorText(w, err) {
		return
	}

	//update task locations
	newLocationIDs := make(map[int64]bool)
	for _, idStr := range r.PostForm["location_ids"] {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || !isValidLocationID[id] {
			msg := fmt.Sprintf("invalid location ID: %q", idStr)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}
		newLocationIDs[id] = true
	}
	if len(newLocationIDs) == 0 {
		http.Error(w, "need to specify at least one location", http.StatusBadRequest)
		return
	}
	for locationID := range newLocationIDs {
		if !isTaskLocation[locationID] {
			err := tx.Insert(&db.TaskLocation{TaskID: task.ID, LocationID: locationID})
			if respondwith.ErrorText(w, err) {
				return
			}
		}
	}
	for locationID := range isTaskLocation {
		if !newLocationIDs[locationID] {
			_, err := tx.Delete(&db.TaskLocation{TaskID: task.ID, LocationID: locationID})
			if respondwith.ErrorText(w, err) {
				return
			}
		}
	}

	err = tx.Commit()
	if respondwith.ErrorText(w, err) {
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

var tCloseTask = tmpl("close-task.html", `
	<form method="POST" action="/tasks/{{.ID}}/close">
		<div class="form-row">
			<label>Label</label>
			<input required type="text" value="{{.Label}}" readonly />
		</div>
		<input type="checkbox" id="has_recurrence" name="has_recurrence" value="true" class="for-fieldset" {{if .RecurrenceDays}}checked{{end}} />
		<fieldset>
			<label for="has_recurrence">Configure recurrence</label>
			<div class="form-row">
				<label for="recurrence_days">Recurrence interval (in days)</label>
				<input type="number" name="recurrence_days" id="recurrence_days" value="{{if .RecurrenceDays}}{{.RecurrenceDays}}{{end}}" min="0" step="1" />
			</div>
			<div class="flash flash-primary">
				Upon closing, the task will respawn with the start date set to that many days from now.
			</div>
		</fieldset>
		<div class="button-row">
			<button type="submit">Close</button>
		</div>
	</form>
`)

func (h *handler) AskCloseTask(w http.ResponseWriter, r *http.Request) {
	task := h.FindTaskFromRequest(w, r)
	if task == nil {
		return
	}

	Page{
		Title: "Close task",
		Navigation: []BreadcrumbItem{
			{URL: "/tasks", Label: "Tasks"},
			{URL: fmt.Sprintf("/tasks/%d", task.ID), Label: fmt.Sprintf("#%d", task.ID)},
			{URL: r.URL.Path, Label: "Close", Current: true},
		},
		Template: tCloseTask,
		Data:     task,
	}.WriteTo(w)
}

func (h *handler) CloseTask(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if respondwith.ErrorText(w, err) {
		return
	}
	task := h.FindTaskFromRequest(w, r)
	if task == nil {
		return
	}

	task.RecurrenceDays, err = parseRecurrenceDays(r.PostForm)
	if err != nil {
		msg := fmt.Sprintf("invalid recurrence_days value: %q", r.PostForm.Get("recurrence_days"))
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if task.RecurrenceDays == 0 {
		//without recurrence, closing a task just deletes it
		_, err := h.dbi.Delete(task)
		if respondwith.ErrorText(w, err) {
			return
		}
	} else {
		//with recurrence, closing a task shifts its start and due date into the future
		durationInDays := task.DueAt.Sub(task.StartsAt)
		task.StartsAt = date.Now().AddDays(int(task.RecurrenceDays))
		task.DueAt = task.StartsAt.AddDays(durationInDays)
		_, err := h.dbi.Update(task)
		if respondwith.ErrorText(w, err) {
			return
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func parsePriority(input string) (uint16, error) {
	val, err := strconv.ParseUint(input, 10, 16)
	if err != nil || val > 3 {
		return 0, fmt.Errorf("invalid priority value: %q", input)
	}
	return uint16(val), nil
}

func parseRecurrenceDays(postForm url.Values) (int32, error) {
	if postForm.Get("has_recurrence") != "true" {
		return 0, nil
	}
	val, err := strconv.ParseUint(postForm.Get("recurrence_days"), 0, 31)
	return int32(val), err
}
