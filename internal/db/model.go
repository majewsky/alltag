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

package db

import (
	"database/sql"
	"fmt"
	"math"
	"net/url"
	"time"

	"github.com/majewsky/alltag/internal/date"
	"github.com/sapcc/go-bits/easypg"
	"github.com/sapcc/go-bits/logg"
	"gopkg.in/gorp.v2"
)

//TaskClass is an enum that appears in type TaskAttributes.
type TaskClass string

const (
	//TaskClassMental is an enum value for mental tasks.
	TaskClassMental TaskClass = "mental"
	//TaskClassPhysical is an enum value for physical tasks.
	TaskClassPhysical TaskClass = "physical"
)

//IsTaskClass contains all acceptable TaskClass values.
var IsTaskClass = map[TaskClass]bool{
	TaskClassMental:   true,
	TaskClassPhysical: true,
}

//Task describes a single task. It may occur once or have a recurrence rule
//configured. This struct contains only the ID and label, which is the minimum
//set of data required to create a task. All other attributes are in the
//TaskAttributes instance containing the same task ID.
//
//Each task is owned by a user. Only that user can see the task and interact
//with it.
type Task struct {
	//These basic attributes are filled on initial creation.
	ID       int64  `db:"id"`
	Label    string `db:"label"`
	UserName string `db:"username"`

	//The following attributes are entered during classification and are zero
	//before that. `task.Class == nil` is the canonical test for whether a task
	//has been classified or not, cf. IsClassified().
	Class           *TaskClass `db:"class"`
	InitialPriority uint16     `db:"init_priority"`
	FinalPriority   uint16     `db:"final_priority"`
	//If RecurrenceDays is not 0, marking the task as done will not delete it,
	//but reset its StartsAt to this many days after now (and shift DueAt
	//accordingly).
	RecurrenceDays int32 `db:"recurrence_days"`

	//The StartsAt and DueAt timestamps are set during classification.
	StartsAt date.Date `db:"starts_at"`
	DueAt    date.Date `db:"due_at"`
}

//IsClassified returns whether this has undergone classification.
func (t Task) IsClassified() bool {
	return t.Class != nil
}

//CurrentPriority interpolates the current priority of this task.
//For unclassified tasks, negative infinity is returned.
//
//Since this method is usually called multiple times in a row while sorting a
//list, calling time.Now() in this function could lead to inconsistent results.
//Therefore, the result of time.Now() (as called once at the start of the
//sorting) is expected as an argument.
func (t Task) CurrentPriority(now time.Time) float64 {
	if !t.IsClassified() {
		return math.Inf(-1)
	}

	startSecs := float64(t.StartsAt.FirstSecondIn(now.Location()).Unix())
	nowSecs := float64(now.Unix())
	endSecs := float64(t.DueAt.FirstSecondIn(now.Location()).Unix())

	startPrio := float64(t.InitialPriority)
	endPrio := float64(t.FinalPriority)

	return (endPrio - startPrio) / (endSecs - startSecs) * (nowSecs - startSecs)
}

//SortOrder is like CurrentPriority, except that for tasks that start in the
//future, it returns a negative value indicating the time until the task
//starts. This method is used to sort tasks in the UI.
//
//Since this method is usually called multiple times in a row while sorting a
//list, calling time.Now() in this function could lead to inconsistent results.
//Therefore, the result of time.Now() (as called once at the start of the
//sorting) is expected as an argument.
func (t Task) SortOrder(now time.Time) float64 {
	if !t.IsClassified() {
		return math.Inf(-1)
	}
	startsAt := t.StartsAt.FirstSecondIn(now.Location())
	if startsAt.After(now) {
		return -startsAt.Sub(now).Seconds()
	}
	return t.CurrentPriority(now)
}

//Location is a place where tasks can be carried out. There is an M:N
//relationship between tasks and locations.
//
//Each location is owned by a user. Only that user can see the location and
//interact with it.
type Location struct {
	ID       int64  `db:"id"`
	Label    string `db:"label"`
	UserName string `db:"username"`
}

//TaskLocation describes a single entry in the N:M mapping between type Task
//and type Location.
type TaskLocation struct {
	TaskID     int64 `db:"task_id"`
	LocationID int64 `db:"location_id"`
}

//Init connects to the database and initializes the schema and model types.
func Init(urlStr string) (*gorp.DbMap, error) {
	dbURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("malformed ALLTAG_DB_URI: %s", err.Error())
	}

	dbConn, err := easypg.Connect(easypg.Configuration{
		PostgresURL: dbURL,
		Migrations:  sqlMigrations,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot connect to database: %s", err.Error())
	}

	gorpDB := &gorp.DbMap{Db: dbConn, Dialect: gorp.PostgresDialect{}}
	gorpDB.AddTableWithName(Location{}, "locations").SetKeys(true, "id")
	gorpDB.AddTableWithName(Task{}, "tasks").SetKeys(true, "id")
	gorpDB.AddTableWithName(TaskLocation{}, "task_locations").SetKeys(false, "task_id", "location_id")
	return gorpDB, nil
}

//RollbackUnlessCommitted calls Rollback() on a transaction if it hasn't been
//committed or rolled back yet. Use this with the defer keyword to make sure
//that a transaction is automatically rolled back when a function fails.
func RollbackUnlessCommitted(tx *gorp.Transaction) {
	err := tx.Rollback()
	switch err {
	case nil:
		//rolled back successfully
		logg.Info("implicit rollback done")
		return
	case sql.ErrTxDone:
		//already committed or rolled back - nothing to do
		return
	default:
		logg.Error("implicit rollback failed: %s", err.Error())
	}
}
