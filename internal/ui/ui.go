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
	"strconv"

	"github.com/gorilla/mux"
	"github.com/majewsky/alltag/internal/db"
	"github.com/sapcc/go-bits/respondwith"
	"gopkg.in/gorp.v2"
)

type handler struct {
	dbi *gorp.DbMap
}

//NewHandler returns a http.Handler serving Alltag's UI.
func NewHandler(dbi *gorp.DbMap) http.Handler {
	h := handler{dbi}
	r := mux.NewRouter()

	r.Methods("GET").Path("/").
		HandlerFunc(h.StartPage)

	r.Methods("GET").Path("/locations").
		HandlerFunc(h.ListLocations)
	r.Methods("GET").Path("/locations/new").
		HandlerFunc(h.NewOrEditLocation)
	r.Methods("POST").Path("/locations/new").
		HandlerFunc(h.CreateOrUpdateLocation)
	r.Methods("GET").Path("/locations/{id:[0-9]+}").
		HandlerFunc(h.ShowLocation)
	r.Methods("GET").Path("/locations/{id:[0-9]+}/edit").
		HandlerFunc(h.NewOrEditLocation)
	r.Methods("POST").Path("/locations/{id:[0-9]+}/edit").
		HandlerFunc(h.CreateOrUpdateLocation)
	r.Methods("GET").Path("/locations/{id:[0-9]+}/delete").
		HandlerFunc(h.AskDeleteLocation)
	r.Methods("POST").Path("/locations/{id:[0-9]+}/delete").
		HandlerFunc(h.DeleteLocation)

	r.Methods("GET").Path("/tasks/new").
		HandlerFunc(h.NewTask)
	r.Methods("POST").Path("/tasks/new").
		HandlerFunc(h.CreateTask)
	r.Methods("GET").Path("/tasks/{id:[0-9]+}").
		HandlerFunc(h.ShowTask)
	r.Methods("GET").Path("/tasks/{id:[0-9]+}/edit").
		HandlerFunc(h.EditTask)
	r.Methods("POST").Path("/tasks/{id:[0-9]+}/edit").
		HandlerFunc(h.UpdateTask)
	r.Methods("GET").Path("/tasks/{id:[0-9]+}/close").
		HandlerFunc(h.AskCloseTask)
	r.Methods("POST").Path("/tasks/{id:[0-9]+}/close").
		HandlerFunc(h.CloseTask)

	return r
}

func (h *handler) AllLocations(r *http.Request) ([]db.Location, error) {
	var locations []db.Location
	_, err := h.dbi.Select(&locations,
		`SELECT * FROM locations WHERE username = $1 ORDER BY label`,
		currentUser(r),
	)
	return locations, err
}

func (h *handler) FindTaskFromRequest(w http.ResponseWriter, r *http.Request) *db.Task {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if respondwith.ErrorText(w, err) {
		return nil
	}

	var task db.Task
	err = h.dbi.SelectOne(&task,
		`SELECT * FROM tasks WHERE id = $1 AND username = $2`,
		id, currentUser(r),
	)
	if err == sql.ErrNoRows {
		http.Error(w, "Not found", http.StatusNotFound)
		return nil
	}
	if respondwith.ErrorText(w, err) {
		return nil
	}

	editTaskURL := fmt.Sprintf("/tasks/%d/edit", task.ID)
	if !task.IsClassified() && r.URL.Path != editTaskURL {
		http.Redirect(w, r, editTaskURL, http.StatusSeeOther)
		return nil
	}

	return &task
}

func (h *handler) FindTaskLocations(task db.Task) (map[int64]bool, error) {
	rows, err := h.dbi.Query(`SELECT location_id FROM task_locations WHERE task_id = $1`, task.ID)
	if err != nil {
		return nil, err
	}

	result := make(map[int64]bool)
	for rows.Next() {
		var id int64
		err := rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		result[id] = true
	}

	return result, rows.Close()
}
