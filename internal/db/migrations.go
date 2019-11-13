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

var sqlMigrations = map[string]string{
	"001_initial.down.sql": `
		DROP TABLE locations;
		DROP TABLE tasks;
		DROP TABLE task_locations;
		DROP TYPE task_class;
	`,
	"001_initial.up.sql": `
		CREATE TABLE locations (
			id       BIGSERIAL PRIMARY KEY,
			label    TEXT      NOT NULL,
			username TEXT      NOT NULL
		);

		CREATE TYPE task_class AS ENUM ('mental', 'physical');

		CREATE TABLE tasks (
			-- basic data (entered on initial creation)
			id       BIGSERIAL  PRIMARY KEY,
			label    TEXT       NOT NULL,
			username TEXT       NOT NULL,
			-- extended attributes (entered during classification)
			class           task_class DEFAULT NULL,
			init_priority   SMALLINT   NOT NULL DEFAULT 0,
			final_priority  SMALLINT   NOT NULL DEFAULT 0,
			recurrence_days INT        NOT NULL DEFAULT 0,
			-- timestamps
			starts_at DATE NOT NULL DEFAULT NOW(),
			due_at    DATE NOT NULL DEFAULT NOW()
		);

		CREATE TABLE task_locations (
			location_id BIGINT NOT NULL REFERENCES locations ON DELETE CASCADE,
			task_id     BIGINT NOT NULL REFERENCES tasks ON DELETE CASCADE,
			PRIMARY KEY (location_id, task_id)
		);
	`,
}
