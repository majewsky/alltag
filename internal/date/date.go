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

//Package date contains the Date datatype.
package date

import (
	"database/sql/driver"
	"fmt"
	"regexp"
	"time"
)

//Date represents a calendar date. It's similar to time.Time, but it carries no
//information neither about the time of day nor about the timezone.
type Date struct {
	year  int
	month time.Month
	day   int
}

//Epoch is the Date value for the Unix epoch. It is used as a stand-in for
//missing dates.
var Epoch = Date{1970, time.January, 1}

//FromTime converts a time.Time into its Date.
func FromTime(t time.Time) Date {
	y, m, d := t.Date()
	return Date{y, m, d}
}

//Now returns the current Date.
func Now() Date {
	return FromTime(time.Now())
}

var dateRx = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)

//Parse parses a date string in the format "yyyy-mm-dd".
func Parse(input string) (Date, error) {
	if !dateRx.MatchString(input) {
		return Epoch, fmt.Errorf("malformed date value: %q", input)
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", input+" 00:00:00", time.Local)
	if err != nil {
		return Epoch, fmt.Errorf("invalid date value: %q", input)
	}
	return FromTime(t), nil
}

//FirstSecondIn returns a timestamp representing the second when the day with
//this Date starts in the given timezone.
func (d Date) FirstSecondIn(loc *time.Location) time.Time {
	return time.Date(d.year, d.month, d.day, 0, 0, 0, 0, loc)
}

//String returns this date in the format "yyyy-mm-dd".
func (d Date) String() string {
	return d.FirstSecondIn(time.UTC).Format("2006-01-02")
}

//After returns whether this date is after the other one.
func (d Date) After(other Date) bool {
	return cmp(d, other) > 0
}

//Before returns whether this date is before the other one.
func (d Date) Before(other Date) bool {
	return cmp(d, other) < 0
}

func cmp(left Date, right Date) int {
	diff := left.year - right.year
	if diff != 0 {
		return diff
	}
	diff = int(left.month - right.month)
	if diff != 0 {
		return diff
	}
	return left.day - right.day
}

const day = 24 * time.Hour

//Sub returns the number of days between this date and the other one. If
//`other.After(d)`, the return value is negative.
func (d Date) Sub(other Date) int {
	//we do this subtraction in UTC, so we do not have to deal with
	//discontinuities like DST
	t1 := d.FirstSecondIn(time.UTC)
	t2 := other.FirstSecondIn(time.UTC)
	return int(t1.Sub(t2).Round(day) / day)
}

//AddDays shifts this date by that many days into the future (or into the past,
//if the number of days is negative).
func (d Date) AddDays(days int) Date {
	//all calculations done in UTC, same reason as above
	return FromTime(d.FirstSecondIn(time.UTC).AddDate(0, 0, days))
}

//Scan implements the database/sql.Scanner interface.
func (d *Date) Scan(src interface{}) error {
	tst, ok := src.(time.Time)
	if !ok {
		return fmt.Errorf("cannot scan %T into date.Date", src)
	}
	d.year, d.month, d.day = tst.Date()
	return nil
}

//Value implements the database/sql/driver.Valuer interface.
func (d Date) Value() (driver.Value, error) {
	return d.FirstSecondIn(time.UTC), nil
}
