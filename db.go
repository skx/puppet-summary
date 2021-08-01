//
// This package contains our SQLite DB interface.  It is a little ropy.
//

package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//
// The global DB handle.
//
var db *sql.DB

//
// PuppetRuns is the structure which is used to list a summary of puppet
// runs on the front-page.
//
type PuppetRuns struct {
	Fqdn        string
	Environment string
	State       string
	At          string
	Epoch       string
	Ago         string
	Runtime     string
}

//
// PuppetReportSummary is the structure used to represent a series
// of puppet-runs against a particular node.
//
type PuppetReportSummary struct {
	ID          string
	Fqdn        string
	Environment string
	State       string
	At          string
	Ago         string
	Runtime     string
	Failed      int
	Changed     int
	Total       int
	YamlFile    string
}

//
// PuppetHistory is a simple structure used solely for the stacked-graph
// on the front-page of our site.
//
type PuppetHistory struct {
	Date      string
	Failed    string
	Changed   string
	Unchanged string
}

//
// PuppetState is used to return the number of nodes in a given state,
// and is used for the submission of metrics.
//
type PuppetState struct {
	State      string
	Count      int
	Percentage float64
}

//
// SetupDB opens our SQLite database, creating it if necessary.
//
func SetupDB(path string) error {

	var err error

	//
	// Return if the database already exists.
	//
	db, err = sql.Open("sqlite3", "file:"+path+"?_journal_mode=WAL")
	if err != nil {
		return err
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
	  environment text,
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
		return err
	}

	//
	// Check if the table has environment column
	//
	var name string
	row := db.QueryRow("SELECT name FROM pragma_table_info('reports') WHERE name='environment'")
	err = row.Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Did not find environment column, adding")
			_, err = db.Exec("ALTER TABLE reports ADD environment text")
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

//
// Populate environment column after adding it
//
func populateEnvironment(prefix string) error {

	//
	// Ensure we have a DB-handle
	//
	if db == nil {
		return errors.New("SetupDB not called")
	}

	var ids map[int]string = make(map[int]string)
	rows, err := db.Query("SELECT id,yaml_file FROM reports WHERE environment IS NULL")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var yamlfile string
		err = rows.Scan(&id, &yamlfile)
		if err != nil {
			return err
		}
		ids[id] = yamlfile
	}

	for id, yamlfile := range ids {
		if len(yamlfile) > 0 {
			var content []byte
			path := filepath.Join(prefix, yamlfile)
			content, err = ioutil.ReadFile(path)
			if err == nil {
				var report PuppetReport
				report, err = ParsePuppetReport(content)
				if err == nil {
					fmt.Println("Updating id:", id, "with environment:", report.Environment)
					_, _ = db.Exec("UPDATE reports SET environment = ? WHERE id = ?", report.Environment, id)
				}
			}
		}
	}
	return err
}

//
// Add an entry to the database.
//
// The entry contains most of the interesting data from the parsed YAML.
//
// But note that it doesn't contain changed resources, etc.
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
	stmt, err := tx.Prepare("INSERT INTO reports(fqdn,environment,state,yaml_file,executed_at,runtime, failed, changed, total, skipped) values(?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	stmt.Exec(data.Fqdn,
		data.Environment,
		data.State,
		path,
		time.Now().Unix(),
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
// Count the number of reports we have reaped.
//
func countUnchangedAndReapedReports() (int, error) {

	//
	// Ensure we have a DB-handle
	//
	if db == nil {
		return 0, errors.New("SetupDB not called")
	}

	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM reports WHERE yaml_file='pruned'")
	err := row.Scan(&count)
	return count, err
}

//
// Get a list of all environments
//
func getEnvironments() ([]string, error) {
	//
	// Ensure we have a DB-handle
	//
	if db == nil {
		return nil, errors.New("SetupDB not called")
	}

	var environments []string
	rows, err := db.Query("SELECT DISTINCT environment FROM reports ORDER BY environment")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var env string
		err := rows.Scan(&env)
		if err != nil {
			return nil, err
		}
		environments = append(environments, env)
	}
	return environments, nil
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
		return nil, errors.New("report not found")
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
	}
	return nil, errors.New("failed to find report with specified ID")
}

//
// Get the data which is shown on our index page
//
//  * The node-name.
//  * The status.
//  * The last-seen time.
//
func getIndexNodes(environment string) ([]PuppetRuns, error) {

	//
	// Our return-result.
	//
	var NodeList []PuppetRuns

	//
	// The threshold which marks the difference between
	// "current" and "orphaned"
	//
	// Here we set it to 4.5 days, which should be long
	// enough to cover any hosts that were powered-off over
	// a weekend.  (Friday + Saturday + Sunday + slack).
	//
	threshold := 3.5 * (24 * 60 * 60)

	//
	// Ensure we have a DB-handle
	//
	if db == nil {
		return nil, errors.New("SetupDB not called")
	}

	//
	// Shared query piece
	//
	queryStart := "SELECT fqdn, state, runtime, max(executed_at) FROM reports WHERE "

	//
	// If environment is specified add a filter
	//
	if len(environment) > 0 {
		queryStart += " environment = '" + environment + "' AND "
	}

	//
	// Select the status - for nodes seen in the past 24 hours.
	//
	rows, err := db.Query(queryStart+" ( ( strftime('%s','now') - executed_at ) < ? ) GROUP by fqdn;", threshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	//
	// We'll keep track of which nodes we've seen recently.
	//
	seen := make(map[string]int)

	//
	// For each row in the result-set
	//
	// Parse into a structure and add to the list.
	//
	for rows.Next() {
		var tmp PuppetRuns
		var at string
		err = rows.Scan(&tmp.Fqdn, &tmp.State, &tmp.Runtime, &at)
		if err != nil {
			return nil, err
		}

		//
		// At this point `at` is a string containing seconds past
		// the epoch.
		//
		// We want to parse that into a string `At` which will
		// contain the literal time, and also the relative
		// time "Ago"
		//
		tmp.Epoch = at
		tmp.Ago = timeRelative(at)

		//
		i, _ := strconv.ParseInt(at, 10, 64)
		tmp.At = time.Unix(i, 0).Format("2006-01-02 15:04:05")

		//
		// Mark this node as non-orphaned.
		//
		seen[tmp.Fqdn] = 1

		//
		// Add the new record.
		//
		NodeList = append(NodeList, tmp)

	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	//
	// Now look for orphaned nodes.
	//
	rows2, err2 := db.Query(queryStart+" ( ( strftime('%s','now') - executed_at ) > ? ) GROUP by fqdn;", threshold)
	if err2 != nil {
		return nil, err
	}
	defer rows2.Close()

	//
	// For each row in the result-set
	//
	// Parse into a structure and add to the list.
	//
	for rows2.Next() {
		var tmp PuppetRuns
		var at string
		err = rows2.Scan(&tmp.Fqdn, &tmp.State, &tmp.Runtime, &at)
		if err != nil {
			return nil, err
		}

		//
		// At this point `at` is a string containing
		// seconds-past-the-epoch.
		//
		// We want that to contain a human-readable
		// string so we first convert to an integer, then
		// parse as a Unix-timestamp
		//
		tmp.Epoch = at
		tmp.Ago = timeRelative(at)

		//
		i, _ := strconv.ParseInt(at, 10, 64)
		tmp.At = time.Unix(i, 0).Format("2006-01-02 15:04:05")

		//
		// Force the state to be `orphaned`.
		//
		tmp.State = "orphaned"

		//
		// If we've NOT already seen this node then
		// we can add it to our result set.
		//
		if seen[tmp.Fqdn] != 1 {
			NodeList = append(NodeList, tmp)
		}
	}
	err = rows2.Err()
	if err != nil {
		return nil, err
	}

	return NodeList, nil
}

//
// Return the state of our nodes.
//
func getStates(environment string) ([]PuppetState, error) {

	//
	// Get the nodes.
	//
	NodeList, err := getIndexNodes(environment)
	if err != nil {
		return nil, err
	}

	//
	// Create a map to hold state.
	//
	states := make(map[string]int)

	//
	// Each known-state will default to being empty.
	//
	states["changed"] = 0
	states["unchanged"] = 0
	states["failed"] = 0
	states["orphaned"] = 0

	//
	// Count the nodes we encounter, such that we can
	// create a %-figure for each distinct-state.
	//
	var total int

	//
	// Count the states.
	//
	for _, o := range NodeList {
		states[o.State]++
		total++
	}

	//
	// Our return-result
	//
	var data []PuppetState

	//
	// Get the distinct keys/states in a sorted order.
	//
	var keys []string
	for name := range states {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	//
	// Now for each key ..
	//
	for _, name := range keys {

		var tmp PuppetState
		tmp.State = name
		tmp.Count = states[name]
		tmp.Percentage = 0

		// Percentage has to be capped :)
		if total != 0 {
			c := float64(states[name])
			tmp.Percentage = (c / float64(total)) * 100
		}
		data = append(data, tmp)
	}

	return data, nil
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
	stmt, err := db.Prepare("SELECT id, fqdn, environment, state, executed_at, runtime, failed, changed, total, yaml_file FROM reports WHERE fqdn=? ORDER by executed_at DESC")
	if err != nil {
		return nil, err
	}
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
		var at string
		err = rows.Scan(&tmp.ID, &tmp.Fqdn, &tmp.Environment, &tmp.State, &at, &tmp.Runtime, &tmp.Failed, &tmp.Changed, &tmp.Total, &tmp.YamlFile)
		if err != nil {
			return nil, err
		}

		//
		// At this point `at` is a string containing seconds past
		// the epoch.
		//
		// We want to parse that into a string `At` which will
		// contain the literal time, and also the relative
		// time "Ago"
		//
		tmp.Ago = timeRelative(at)

		i, _ := strconv.ParseInt(at, 10, 64)
		tmp.At = time.Unix(i, 0).Format("2006-01-02 15:04:05")

		// Add the result of this fetch to our list.
		NodeList = append(NodeList, tmp)
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
func getHistory(environment string) ([]PuppetHistory, error) {

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

	sel := "SELECT DISTINCT(strftime('%d/%m/%Y', DATE(executed_at, 'unixepoch'))) FROM reports"
	if len(environment) > 0 {
		sel = sel + " WHERE environment = '" + environment + "'"
	}
	//
	// Get all the distinct dates we have data for.
	//
	stmt, err := db.Prepare(sel)
	if err != nil {
		return nil, err
	}

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
		err = rows.Scan(&d)
		if err != nil {
			return nil, errors.New("failed to scan SQL")
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
	loc,_ := time.LoadLocation("Local")
	for _, known := range dates {

		//
		// The result for this date.
		//
		var x PuppetHistory
		x.Changed = "0"
		x.Unchanged = "0"
		x.Failed = "0"
		x.Date = known
		formatTime,_:=time.ParseInLocation("02/01/2006 15:04:05", known + " 00:00:00", loc)
		ts1 := formatTime.Unix()
		ts2 := ts1 + 3600*24 - 1

		query := "SELECT distinct state, COUNT(state) AS CountOf FROM reports WHERE executed_at between ? and ?"
		if len(environment) > 0 {
			query += " AND environment = '" + environment + "' "
		}
		query += " GROUP by state"
		stmt, err = db.Prepare(query)
		if err != nil {
			return nil, err
		}

		rows, err = stmt.Query(ts1, ts2)
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

			err = rows.Scan(&name, &count)
			if err != nil {
				return nil, errors.New("failed to scan SQL")
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
// Prune dangling reports
//
// Walk the reports directory and remove all files that are not referenced
// in the database.
//
func pruneDangling(prefix string, noop bool, verbose bool) error {

	//
	// Ensure we have a DB-handle
	//
	if db == nil {
		return errors.New("SetupDB not called")
	}

	//
	// Find all yaml files
	//
	find, err := db.Query("SELECT yaml_file FROM reports")
	if err != nil {
		return err
	}

	//
	// Copy them for easy access
	//
	reports := make(map[string]int)
	for find.Next() {
		var fname string
		find.Scan(&fname)
		reports[fname] = 1
	}

	//
	// We have to be real careful so we will match filenames to this regexp
	//
	r, _ := regexp.Compile("^[0-9a-f]{40}$")

	//
	// Walk reports directory
	//
	err = filepath.Walk(prefix, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error accessing path %q: %v\n", path, err)
			return err
		}
		if !info.IsDir() {
			rel, lerr := filepath.Rel(prefix, path)
			if r.MatchString(info.Name()) && lerr == nil {
				_, found := reports[rel]
				if found {
					// can be used to find db entries with no file reports
					reports[rel] = 2
				} else {
					if noop {
						fmt.Printf("Would remove file %q\n", path)
					} else {
						if verbose {
							fmt.Printf("Removing file %q\n", path)
						}
						os.Remove(path)
					}
				}
			} else {
				fmt.Printf("Warning - unexpected file or error parsing: %q\n", path)
			}
		}
		return nil
	})

	//
	// Check for database entries with missing yaml file reports
	//
	if verbose {
		for k, v := range reports {
			if v != 2 {
				fmt.Printf("Missing file: %q\n", k)
			}
		}
	}

	return err
}

//
// Prune old reports
//
// We have to find the old reports, individually, so we can unlink the
// copy of the on-disk YAML, but once we've done that we can delete them
// as a group.
//
func pruneReports(environment string, prefix string, days int, verbose bool) error {

	//
	// Ensure we have a DB-handle
	//
	if db == nil {
		return errors.New("SetupDB not called")
	}

	//
	// Select appropriate environment, if specified
	//
	envCondition := ""
	if len(environment) > 0 {
		envCondition = " AND environment = '" + environment + "'"
	}

	//
	// Convert our query into something useful.
	//
	time := days * (24 * 60 * 60)

	//
	// Find things that are old.
	//
	find, err := db.Prepare("SELECT id,yaml_file FROM reports WHERE ( ( strftime('%s','now') - executed_at ) > ? )" + envCondition)
	if err != nil {
		return err
	}

	//
	// Remove old reports, en mass.
	//
	clean, err := db.Prepare("DELETE FROM reports WHERE ( ( strftime('%s','now') - executed_at ) > ? )" + envCondition)
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

		err = rows.Scan(&id, &path)
		if err == nil {

			//
			// Convert the path to a qualified one,
			// rather than one relative to our report-dir.
			//
			path = filepath.Join(prefix, path)
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

//
// Prune reports from nodes which are unchanged.
//
// We have to find the old reports, individually, so we can unlink the
// copy of the on-disk YAML, but once we've done that we can delete them
// as a group.
//
func pruneUnchanged(environment string, prefix string, verbose bool) error {

	//
	// Ensure we have a DB-handle
	//
	if db == nil {
		return errors.New("SetupDB not called")
	}

	//
	// Select appropriate environment, if specified
	//
	envCondition := ""
	if len(environment) > 0 {
		envCondition = " AND environment = '" + environment + "'"
	}

	//
	// Find unchanged reports.
	//
	find, err := db.Prepare("SELECT id,yaml_file FROM reports WHERE state='unchanged'" + envCondition)
	if err != nil {
		return err
	}

	//
	// Prepare to update them all.
	//
	clean, err := db.Prepare("UPDATE reports SET yaml_file='pruned' WHERE state='unchanged'" + envCondition)
	if err != nil {
		return err
	}

	//
	// Find the reports.
	//
	rows, err := find.Query()
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

		err = rows.Scan(&id, &path)
		if err == nil {

			//
			// Convert the path to a qualified one,
			// rather than one relative to our report-dir.
			//
			path = filepath.Join(prefix, path)
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
	_, err = clean.Exec()
	if err != nil {
		return err
	}

	return nil
}

func pruneOrphaned(environment string, prefix string, verbose bool) error {

	NodeList, err := getIndexNodes(environment)
	if err != nil {
		return err
	}

	for _, entry := range NodeList {

		if entry.State == "orphaned" {
			if verbose {
				fmt.Printf("Orphaned host: %s\n", entry.Fqdn)
			}

			//
			// Find all reports that refer to this host.
			//
			rows, err := db.Query("SELECT yaml_file FROM reports WHERE fqdn=?", entry.Fqdn)
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				var tmp string
				err = rows.Scan(&tmp)
				if err != nil {
					return err
				}

				//
				// Convert the path to a qualified one,
				// rather than one relative to our report-dir.
				//
				path := filepath.Join(prefix, tmp)
				if verbose {
					fmt.Printf("\tRemoving: %s\n", path)
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

			//
			// Now remove the report-entries
			//
			clean, err := db.Prepare("DELETE FROM reports WHERE fqdn=?")
			if err != nil {
				return err
			}
			defer clean.Close()
			_, err = clean.Exec(entry.Fqdn)
			if err != nil {
				return err
			}

		}

	}

	return nil
}
