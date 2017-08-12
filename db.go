//
// This package contains our SQLite DB interface.  It is a little ropy.
//

package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

//
// The global DB handle.
//
var db *sql.DB

//
// Define a structure for the nodes that are shown in the index.
//
type PuppetRuns struct {
	Fqdn    string
	State   string
	At      string
	Runtime string
}

//
// Define a structure for our list of reports
//
type PuppetReportSummary struct {
	Id      string
	Fqdn    string
	State   string
	At      string
	Runtime string
	Failed  int
	Changed int
	Total   int
}

//
// This structure is used solely for the stacked-graph on the
// front-page.
//
type PuppetHistory struct {
	Date      string
	Failed    string
	Changed   string
	Unchanged string
}

//
// Open our SQLite database, creating it if necessary.
//
func SetupDB(path string) {

	var err error

	//
	// Return if the database already exists.
	//
	db, err = sql.Open("sqlite3", path)
	if err != nil {
		fmt.Printf("Failed to setup database at %s\n", path)
		os.Exit(1)
	}

	//
	// Create the table.
	//
	sqlStmt := `

        PRAGMA automatic_index = ON;
        PRAGMA cache_size = 32768;
        PRAGMA journal_size_limit = 67110000;
        PRAGMA locking_mode = NORMAL;
        PRAGMA synchronous = NORMAL;
        PRAGMA temp_store = MEMORY;
        PRAGMA journal_mode = WAL;
        PRAGMA wal_autocheckpoint = 16384;

        CREATE TABLE IF NOT EXISTS reports (
          id          INTEGER PRIMARY KEY AUTOINCREMENT,
          fqdn        text,
          state       text,
          yaml_file   text,
          runtime     integer,
          executed_at integer(4),
          total       integer,
          skipped     integer,
          failed      integer,
          changed     integer
        )
	`

	//
	// Create the table, if missing.
	//
	// Errors here are pretty unlikely.
	//
	_, err = db.Exec(sqlStmt)
	if err != nil {
		panic(err)
	}
}

//
// Add an entry to the database.
//
// The entry contains most of the interesting data from the parsed YAML.
//
// But note that it odesn't contain changed resources, etc.
//
//
func addDB(data PuppetReport, path string) error {

	//
	// Ensure we have a DB-handle
	//
	if db == nil {
		return errors.New("SetupDB not called")
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT INTO reports(fqdn,state,yaml_file,executed_at,runtime, failed, changed, total, skipped) values(?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	stmt.Exec(data.Fqdn,
		data.State,
		path,
		data.At_Unix,
		data.Runtime,
		data.Failed,
		data.Changed,
		data.Total,
		data.Skipped)
	tx.Commit()

	return nil
}

//
// Count the number of reports we have.
//
func countReports() (int, error) {

	//
	// Ensure we have a DB-handle
	//
	if db == nil {
		return 0, errors.New("SetupDB not called")
	}

	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM reports")
	err := row.Scan(&count)
	return count, err
}

//
// Return the contents of the YAML file which was associated
// with the given report-ID.
//
func getYAML(prefix string, id string) ([]byte, error) {

	//
	// Ensure we have a DB-handle
	//
	if db == nil {
		return nil, errors.New("SetupDB not called")
	}

	var path string
	row := db.QueryRow("SELECT yaml_file FROM reports WHERE id=?", id)
	err := row.Scan(&path)

	switch {
	case err == sql.ErrNoRows:
	case err != nil:
		return nil, errors.New("Report not found.")
	default:
	}

	//
	// Read the file content, first of all adding in the
	// prefix.
	//
	// (Because our reports are stored as relative paths
	// such as "$host/$time", rather than absolute paths
	// such as "reports/$host/$time".)
	//
	if len(path) > 0 {
		path = filepath.Join(prefix, path)
		content, err := ioutil.ReadFile(path)
		return content, err
	} else {
		return nil, errors.New("Failed to find report with specified ID")
	}
}

//
// Get the data which is shown on our index page
//
//  * The node-name.
//  * The status.
//  * The last-seen time.
//
func getIndexNodes() ([]PuppetRuns, error) {

	//
	// Ensure we have a DB-handle
	//
	if db == nil {
		return nil, errors.New("SetupDB not called")
	}

	//
	// Select the status.
	//
	rows, err := db.Query("SELECT fqdn, state, runtime, max(executed_at) FROM reports GROUP by fqdn;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	//
	// We'll have a list of them.
	//
	var NodeList []PuppetRuns

	//
	// For each row in the result-set
	//
	// Parse into a structure and add to the list.
	//
	for rows.Next() {
		var tmp PuppetRuns
		err := rows.Scan(&tmp.Fqdn, &tmp.State, &tmp.Runtime, &tmp.At)
		if err == nil {

			//
			// At this point tmp.At is a string containing
			// seconds-past-the-epoch.
			//
			// We want that to contain a human-readable
			// string so we first convert to an integer, then
			// parse as a Unix-timestamp
			//
			i, _ := strconv.ParseInt(tmp.At, 10, 64)
			tmp.At = time.Unix(i, 0).Format("2006-01-02 15:04:05")

			NodeList = append(NodeList, tmp)
		}
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return NodeList, nil
}

//
// Get the summary-details of the runs against a given host
//
func getReports(fqdn string) ([]PuppetReportSummary, error) {

	//
	// Ensure we have a DB-handle
	//
	if db == nil {
		return nil, errors.New("SetupDB not called")
	}

	//
	// Select the status.
	//
	stmt, err := db.Prepare("SELECT id, fqdn, state, executed_at, runtime, failed, changed, total FROM reports WHERE fqdn=? ORDER by executed_at DESC LIMIT 50")
	rows, err := stmt.Query(fqdn)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	defer rows.Close()

	//
	// We'll return a list of these hosts.
	//
	var NodeList []PuppetReportSummary

	//
	// For each row in the result-set
	//
	// Parse into a structure and add to the list.
	//
	for rows.Next() {
		var tmp PuppetReportSummary
		err := rows.Scan(&tmp.Id, &tmp.Fqdn, &tmp.State, &tmp.At, &tmp.Runtime, &tmp.Failed, &tmp.Changed, &tmp.Total)
		if err == nil {
			//
			// At this point tmp.At is a string containing
			// seconds-past-the-epoch.
			//
			// We want that to contain a human-readable
			// string so we first convert to an integer, then
			// parse as a Unix-timestamp
			//
			i, _ := strconv.ParseInt(tmp.At, 10, 64)
			tmp.At = time.Unix(i, 0).Format("2006-01-02 15:04:05")

			// Add the result of this fetch to our list.
			NodeList = append(NodeList, tmp)
		}
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	if len(NodeList) < 1 {
		return nil, errors.New("Failed to find reports for " + fqdn)

	}
	return NodeList, nil
}

//
// Get data for our stacked bar-graph
//
func getHistory() ([]PuppetHistory, error) {

	//
	// Ensure we have a DB-handle
	//
	if db == nil {
		return nil, errors.New("SetupDB not called")
	}

	//
	// Our result.
	//
	var res []PuppetHistory

	//
	// An array to hold the unique dates we've seen.
	//
	var dates []string

	//
	// Get all the distinct dates we have data for.
	//
	stmt, err := db.Prepare("SELECT DISTINCT(strftime('%d/%m/%Y', DATE(executed_at, 'unixepoch'))) FROM reports")
	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	defer rows.Close()

	//
	// For each row in the result-set
	//
	for rows.Next() {
		var d string
		err := rows.Scan(&d)
		if err != nil {
			return nil, errors.New("Failed to scan SQL")
		}

		dates = append(dates, d)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	//
	// Now we have all the unique dates in `dates`.
	//
	for _, known := range dates {

		//
		// The result for this date.
		//
		var x PuppetHistory
		x.Changed = "0"
		x.Unchanged = "0"
		x.Failed = "0"
		x.Date = known

		stmt, err = db.Prepare("SELECT distinct state, COUNT(state) AS CountOf FROM reports WHERE strftime('%d/%m/%Y', DATE(executed_at, 'unixepoch'))=? GROUP by state")
		rows, err = stmt.Query(known)
		if err != nil {
			return nil, err
		}
		defer stmt.Close()
		defer rows.Close()

		//
		// For each row in the result-set
		//
		for rows.Next() {
			var name string
			var count string

			err := rows.Scan(&name, &count)
			if err != nil {
				return nil, errors.New("Failed to scan SQL")
			}
			if name == "changed" {
				x.Changed = count
			}
			if name == "unchanged" {
				x.Unchanged = count
			}
			if name == "failed" {
				x.Failed = count
			}
		}
		err = rows.Err()
		if err != nil {
			return nil, err
		}

		//
		// Add this days result.
		//
		res = append(res, x)

	}

	return res, err

}

//
// Prune old reports
//
// We have to find the old reports, individually, so we can unlink the
// copy of the on-disk YAML, but once we've done that we can delete them
// as a group.
//
func pruneReports(days int, verbose bool) error {

	//
	// Ensure we have a DB-handle
	//
	if db == nil {
		return errors.New("SetupDB not called")
	}

	//
	// Convert our query into something useful.
	//
	time := days * (24 * 60 * 60)

	//
	// Find things that are old.
	//
	find, err := db.Prepare("SELECT id,yaml_file FROM reports WHERE ( ( strftime('%s','now') - executed_at ) > ? )")
	if err != nil {
		return err
	}

	//
	// Remove old reports, en mass.
	//
	clean, err := db.Prepare("DELETE FROM reports WHERE ( ( strftime('%s','now') - executed_at ) > ? )")
	if err != nil {
		return err
	}

	//
	// Find the old reports.
	//
	rows, err := find.Query(time)
	if err != nil {
		return err
	}
	defer find.Close()
	defer clean.Close()
	defer rows.Close()

	//
	// For each row in the result-set
	//
	// Parse into "id" + "path", then remove the path from disk.
	//
	for rows.Next() {
		var id string
		var path string

		err := rows.Scan(&id, &path)
		if err == nil {

			if verbose {
				fmt.Printf("Removing ID:%s - %s\n", id, path)
			}

			//
			//  Remove the file from-disk
			//
			//  We won't care if this fails, it might have
			// been removed behind our back or failed to
			// be uploaded in the first place.
			//
			os.Remove(path)
		}
	}
	err = rows.Err()
	if err != nil {
		return err
	}

	//
	//  Now cleanup the old records
	//
	_, err = clean.Exec(time)
	if err != nil {
		return err
	}

	return nil
}
