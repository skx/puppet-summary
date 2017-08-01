//
//  To start the server run:
//
//     go run main.go
//
//  To post a report do:
//
//     curl  --data-binary @./201707292317.yaml http://localhost:3001/upload
//
//  To import __ALL__ your reports:
//
//      find . -name '*.yaml' -exec curl --data-binary @\{\} http://localhost:3001/upload \;
//
//
//  TODO:
//
//    * Add sub-commands for different modes.  We want at least two:
//
//        ps serve  -> Run the httpd.
//        ps prune  -> Remove old reports
//
//    * Update the SQLite magic.
//       - Simplify how this works.
//       - Add command-line flags for storage-location and DB-path.
//
//
// Steve
// --
//

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"text/template"
	"time"
)

//
// Create the SQLite database.
//
// If this already exists then we'll do nothing.
//
func SetupDB() {

	//
	// Return if the database already exists.
	//
	_, err := os.Stat("foo.db")
	if err == nil {
		//
		// It does.  Return.
		//
		return
	}
	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	//
	// Create the table.
	//
	sqlStmt := `
        CREATE TABLE reports (
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
	// TODO: Changed, Failed, RunTime
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
func addDB(data PuppetReport, path string) {
	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	stmt, err := tx.Prepare("INSERT INTO reports(fqdn,state,yaml_file,executed_at,runtime, failed, changed, total, skipped) values(?,?,?,?,?,?,?,?,?)")
	if err != nil {
		panic(err)
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
}

/*
 * Handle the submission of Puppet report.
 *
 * The input is read, and parsed as Yaml, and assuming that succeeds
 * then the data is written beneath ./reports/$hostname/$timestamp
 * and a summary-record is inserted into our SQLite database.
 *
 */
func ReportSubmissionHandler(res http.ResponseWriter, req *http.Request) {
	var (
		status int
		err    error
	)
	defer func() {
		if nil != err {
			http.Error(res, err.Error(), status)
		}
	}()

	//
	// Read the body of the request.
	//
	content, err := ioutil.ReadAll(req.Body)
	if err != nil {
		status = http.StatusInternalServerError
		return
	}

	//
	// Parse the YAML into something we can work with.
	//
	report, err := ParsePuppetReport(content)
	if err != nil {
		status = http.StatusInternalServerError
		return
	}

	//
	// Create a report directory, unless it already exists.
	//
	dir := filepath.Join("./reports", report.Fqdn)
	_, err = os.Stat(dir)
	if err != nil {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			status = http.StatusInternalServerError
			return
		}
	}

	//
	// Now write out the file.
	//
	path := filepath.Join(dir, fmt.Sprintf("%d", report.At_Unix))
	err = ioutil.WriteFile(path, content, 0644)
	if err != nil {
		status = http.StatusInternalServerError
		return
	}

	//
	// Record the new entry.
	//
	addDB(report, path)

	//
	// Show something to the caller.
	//
	out := fmt.Sprintf("{\"host\":\"%s\"}", report.Fqdn)
	fmt.Fprintf(res, string(out))

}

//
// Called via GET /report/NN
//
func ReportHandler(res http.ResponseWriter, req *http.Request) {
	var (
		status int
		err    error
	)
	defer func() {
		if nil != err {
			http.Error(res, err.Error(), status)
		}
	}()

	//
	// Get the node name we're going to show.
	//
	vars := mux.Vars(req)
	id := vars["id"]

	//
	// Open the database
	//
	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	//
	// Get the path to the file for this file.
	//
	stmt, err := db.Prepare("SELECT yaml_file FROM reports WHERE id=?")
	rows, err := stmt.Query(id)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	defer rows.Close()

	//
	// The path to the file we expect to receive.
	//
	var path string

	//
	// For each row in the result-set
	//
	for rows.Next() {
		err := rows.Scan(&path)
		if err != nil {
			status = http.StatusInternalServerError
			return
		}
	}
	err = rows.Err()
	if err != nil {
		panic(err)
	}

	//
	// Read the file content
	//
	content, err := ioutil.ReadFile(path)
	if err != nil {
		status = http.StatusNotFound
		return
	}

	//
	// Parse it
	//
	report, err := ParsePuppetReport(content)
	if err != nil {
		status = http.StatusInternalServerError
		return
	}

	//
	// Define a template for the result we'll send to the browser.
	//
	tmpl, err := Asset("data/report_handler.template")
	if err != nil {
		fmt.Printf("Failed to find asset data/report_handler.template")
		os.Exit(2)
	}

	//
	// All done.
	//
	src := string(tmpl)
	t := template.Must(template.New("tmpl").Parse(src))
	t.Execute(res, report)
}

//
// Called via GET /node/$FQDN
//
func NodeHandler(res http.ResponseWriter, req *http.Request) {
	var (
		status int
		err    error
	)
	defer func() {
		if nil != err {
			http.Error(res, err.Error(), status)
		}
	}()

	//
	// Get the node name we're going to show.
	//
	vars := mux.Vars(req)
	fqdn := vars["fqdn"]

	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	//
	// Select the status.
	//
	stmt, err := db.Prepare("SELECT id, state, executed_at, runtime, failed, changed, total FROM reports WHERE fqdn=? ORDER by executed_at DESC LIMIT 50")
	rows, err := stmt.Query(fqdn)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	defer rows.Close()

	//
	// Define a structure for our results.
	//
	type PuppetNode struct {
		Id      string
		State   string
		At      string
		Runtime string
		Failed  int
		Changed int
		Total   int
	}

	//
	// We'll have a list of them.
	//
	var NodeList []PuppetNode

	//
	// For each row in the result-set
	//
	// Parse into a structure and add to the list.
	//
	for rows.Next() {
		var tmp PuppetNode
		err := rows.Scan(&tmp.Id, &tmp.State, &tmp.At, &tmp.Runtime, &tmp.Failed, &tmp.Changed, &tmp.Total)
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
		panic(err)
	}

	if len(NodeList) < 1 {
		status = http.StatusNotFound
		return
	}

	//
	// Annoying.
	//
	type Pagedata struct {
		Fqdn  string
		Nodes []PuppetNode
	}
	var x Pagedata
	x.Nodes = NodeList
	x.Fqdn = fqdn

	//
	// Define a template for the result we'll send to the browser.
	//
	tmpl, err := Asset("data/node_handler.template")
	if err != nil {
		fmt.Printf("Failed to find asset data/node_handler.template")
		os.Exit(2)
	}

	//
	//  Populate the template and return it.
	//
	src := string(tmpl)
	t := template.Must(template.New("tmpl").Parse(src))
	t.Execute(res, x)
}

//
// Show the hosts we know about - and their last known state.
//
func IndexHandler(res http.ResponseWriter, req *http.Request) {

	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	//
	// Select the status.
	//
	rows, err := db.Query("SELECT fqdn, state, max(executed_at) FROM reports GROUP by fqdn;")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	//
	// Define a structure for our results.
	//
	type PuppetNode struct {
		Fqdn  string
		State string
		At    string
	}

	//
	// We'll have a list of them.
	//
	var NodeList []PuppetNode

	//
	// For each row in the result-set
	//
	// Parse into a structure and add to the list.
	//
	for rows.Next() {
		var tmp PuppetNode
		err := rows.Scan(&tmp.Fqdn, &tmp.State, &tmp.At)
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
		panic(err)
	}

	//
	// Define a template for the result we'll send to the browser.
	//
	tmpl, err := Asset("data/index_handler.template")
	if err != nil {
		fmt.Printf("Failed to find asset data/index_handler.template")
		os.Exit(2)
	}

	//
	//  Populate the template and return it.
	//
	src := string(tmpl)
	t := template.Must(template.New("tmpl").Parse(src))
	t.Execute(res, NodeList)
}

//
//  Entry-point.
//
func main() {

	//
	// Parse the command-line arguments.
	//
	host := flag.String("host", "127.0.0.1", "The IP to listen upon")
	port := flag.Int("port", 3001, "The port to bind upon")
	flag.Parse()

	SetupDB()

	//
	// Create a new router and our route-mappings.
	//
	router := mux.NewRouter()

	//
	// Upload a new report.
	//
	router.HandleFunc("/upload", ReportSubmissionHandler).Methods("POST")

	//
	// Show the recent state of a node.
	//
	router.HandleFunc("/node/{fqdn}", NodeHandler).Methods("GET")

	//
	// Show "everything" about a given run.
	//
	router.HandleFunc("/report/{id}", ReportHandler).Methods("GET")

	//
	// Handle a display of all known nodes, and their last state.
	//
	router.HandleFunc("/", IndexHandler).Methods("GET")

	//
	// Bind the router.
	//
	http.Handle("/", router)

	//
	// Launch the server
	//
	fmt.Printf("Launching the server on http://%s:%d\n", *host, *port)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", *host, *port), nil)
	if err != nil {
		panic(err)
	}
}
