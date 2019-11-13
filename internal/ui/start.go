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
	"net/http"
	"sort"
	"time"

	"github.com/lib/pq"
	"github.com/majewsky/alltag/internal/db"
	"github.com/sapcc/go-bits/respondwith"
)

var tStartPage = tmpl("startpage.html", `
	<div class="table-container">
		<table class="table responsive has-hover-highlight">
			<thead>
				<tr>
					<th>Location</th>
					<th class="actions">
						<a href="/tasks/new" class="button">Add new task</a>
						or
						{{ if .UnclassifiedTaskID }}
							<a href="/tasks/{{.UnclassifiedTaskID}}/edit" class="button">Classify a task</a>
						{{ else }}
							<button disabled>Classify a task</button>
						{{ end }}
					</th>
				</tr>
			</thead>
			<tbody>
				{{range .Locations}}
					<tr>
						<td data-label="Location">{{.Label}}</td>
						<td class="actions">
							Do a
							{{ $mentalTaskID := index $.NextMentalTaskIDs .ID }}
							{{ if $mentalTaskID }}
								<a href="/tasks/{{$mentalTaskID}}" class="button">mental</a>
							{{ else }}
								<button disabled>mental</button>
							{{ end }}
							or a
							{{ $physicalTaskID := index $.NextPhysicalTaskIDs .ID }}
							{{ if $physicalTaskID }}
								<a href="/tasks/{{$physicalTaskID}}" class="button">physical</a>
							{{ else }}
								<button disabled>physical</button>
							{{ end }}
							task
						</td>
					</tr>
				{{end}}
			</tbody>
		</table>
	</div>
`)

const sqlGetOpenTasks = `
	SELECT t.*
	FROM tasks t
	WHERE class IS NOT NULL AND username = $1
`

const sqlGetOpenTasksByLocation = `
	SELECT t.id, ARRAY_AGG(l.location_id)
	FROM tasks t JOIN task_locations l ON l.task_id = t.id
	WHERE class IS NOT NULL AND username = $1 AND starts_at <= NOW()
	GROUP BY t.id
`

func (h *handler) StartPage(w http.ResponseWriter, r *http.Request) {
	//can only show normal start page once locations are configured
	locations, err := h.AllLocations(r)
	if respondwith.ErrorText(w, err) {
		return
	}
	if len(locations) == 0 {
		http.Redirect(w, r, "/locations", http.StatusSeeOther)
	}

	//check for unclassified tasks
	unclassifiedTaskID, err := h.dbi.SelectInt(
		`SELECT id FROM tasks WHERE class IS NULL AND username = $1 ORDER BY id ASC LIMIT 1`,
		currentUser(r),
	)
	if err == sql.ErrNoRows {
		unclassifiedTaskID = 0
	} else if respondwith.ErrorText(w, err) {
		return
	}

	//retrieve all open tasks incl. their location mappings
	var openTasks []db.Task
	_, err = h.dbi.Select(&openTasks, sqlGetOpenTasks, currentUser(r))
	if respondwith.ErrorText(w, err) {
		return
	}
	now := time.Now()
	sort.Slice(openTasks, func(i, j int) bool {
		return openTasks[i].CurrentPriority(now) < openTasks[j].CurrentPriority(now)
	})

	//retrieve location associations for those tasks
	rows, err := h.dbi.Query(sqlGetOpenTasksByLocation, currentUser(r))
	if respondwith.ErrorText(w, err) {
		return
	}
	locationIDsForTask := make(map[int64][]int64)
	for rows.Next() {
		var (
			taskID      int64
			locationIDs []int64
		)
		err := rows.Scan(&taskID, pq.Array(&locationIDs))
		if respondwith.ErrorText(w, err) {
			return
		}
		locationIDsForTask[taskID] = locationIDs
	}
	err = rows.Close()
	if respondwith.ErrorText(w, err) {
		return
	}

	//select next task for all pairs of (location, taskClass)
	nextMentalTaskIDs := make(map[int64]int64)
	nextPhysicalTaskIDs := make(map[int64]int64)
	for _, task := range openTasks {
		for _, locationID := range locationIDsForTask[task.ID] {
			//because `openTasks` is sorted by current priority, the last task to
			//write into this particular `nextTaskIDs[][]` slot is the one with the
			//highest current priority
			switch *task.Class {
			case db.TaskClassMental:
				nextMentalTaskIDs[locationID] = task.ID
			case db.TaskClassPhysical:
				nextPhysicalTaskIDs[locationID] = task.ID
			}
		}
	}

	Page{
		Title:    "Alltag",
		Template: tStartPage,
		Data: struct {
			Locations           []db.Location
			NextMentalTaskIDs   map[int64]int64
			NextPhysicalTaskIDs map[int64]int64
			UnclassifiedTaskID  int64
		}{locations, nextMentalTaskIDs, nextPhysicalTaskIDs, unclassifiedTaskID},
	}.WriteTo(w)
}
