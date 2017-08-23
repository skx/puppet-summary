//
// Launch our HTTP-server for both consuming reports, and viewing them.
//

package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"github.com/google/subcommands"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"text/template"
)

var ReportPrefix = "reports"

//
// Utility method to determine whether a file/directory exists.
//
func Exists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

//
// API /api/state/$state
//
func APIState(res http.ResponseWriter, req *http.Request) {

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
	// Get the state the user is interested in.
	//
	vars := mux.Vars(req)
	state := vars["state"]

	//
	// Test the state is valid
	//
	switch state {
	case "changed":
	case "unchanged":
	case "failed":
	case "orphaned":
	default:
		err = errors.New("Invalid state")
		status = http.StatusInternalServerError
		return
	}

	//
	// Get the nodes.
	//
	NodeList, err := getIndexNodes()
	if err != nil {
		status = http.StatusInternalServerError
		return
	}

	//
	// The result
	//
	var result []string

	//
	// See what state the user is interested in.
	//
	for _, o := range NodeList {
		if o.State == state {
			result = append(result, o.Fqdn)
		}
	}

	//
	// Convert the string-array to JSON, and return it.
	//
	res.Header().Set("Content-Type", "application/json")

	if len(result) > 0 {
		out, _ := json.Marshal(result)
		fmt.Fprintf(res, "%s", out)
	} else {
		fmt.Fprintf(res, "[]")
	}

}

//
// Show the radiator
//
func RadiatorView(res http.ResponseWriter, req *http.Request) {

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
	// Get the state of the nodes.
	//
	data, err := getStates()
	if err != nil {
		status = http.StatusInternalServerError
		return
	}

	//
	// Sum up our known-nodes.
	//
	total := 0
	for i := range data {
		total += data[i].Count
	}

	//
	// Add in the total count of nodes.
	//
	var tmp PuppetState
	tmp.State = "All"
	tmp.Count = total
	tmp.Percentage = 0
	data = append(data, tmp)

	//
	// Load our template to host the result we'll send to the browser.
	//
	tmpl, err := Asset("data/radiator.template")
	if err != nil {
		err = errors.New("Failed to find asset data/radiator.template")
		status = http.StatusInternalServerError
		return
	}

	//
	// What kind of reply should we send?
	//
	accept := req.Header.Get("Accept")

	switch accept {
	case "application/json":
		js, err := json.Marshal(data)

		if err != nil {
			status = http.StatusInternalServerError
			return
		}
		res.Header().Set("Content-Type", "application/json")
		res.Write(js)

	case "application/xml":
		x, err := xml.MarshalIndent(data, "", "  ")
		if err != nil {
			status = http.StatusInternalServerError
			return
		}

		res.Header().Set("Content-Type", "application/xml")
		res.Write(x)
	default:
		//
		// Populate & return the template.
		//
		src := string(tmpl)
		t := template.Must(template.New("tmpl").Parse(src))
		t.Execute(res, data)
	}
}

//
// Handle the submission of Puppet report.
//
// The input is read, and parsed as Yaml, and assuming that succeeds
// then the data is written beneath ./reports/$hostname/$timestamp
// and a summary-record is inserted into our SQLite database.
//
//
func ReportSubmissionHandler(res http.ResponseWriter, req *http.Request) {
	var (
		status int
		err    error
	)
	defer func() {
		if nil != err {
			http.Error(res, err.Error(), status)

			// Don't spam stdout when running test-cases.
			if flag.Lookup("test.v") == nil {
				fmt.Printf("Error: %s\n", err.Error())
			}
		}
	}()

	//
	// Ensure this was a POST-request
	//
	if req.Method != "POST" {
		err = errors.New("Must be called via HTTP-POST")
		status = http.StatusInternalServerError
		return
	}

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
	// Create a report directory for this host, unless it already exists.
	//
	dir := filepath.Join(ReportPrefix, report.Fqdn)
	if !Exists(dir) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			status = http.StatusInternalServerError
			return
		}
	}

	//
	// Does this report already exist?  This shouldn't happen
	// in a usual setup, but will happen if you're repeatedly
	// importing reports manually from a puppet-server.
	//
	// (Which is something you might do when testing this
	// dashboard.)
	//
	path := filepath.Join(dir, fmt.Sprintf("%d", report.At_Unix))

	if Exists(path) {
		fmt.Fprintf(res, "Ignoring duplicate submission")
		return
	}

	//
	// Create the new report-file, on-disk.
	//
	err = ioutil.WriteFile(path, content, 0644)
	if err != nil {
		status = http.StatusInternalServerError
		return
	}

	//
	// Record that report in our SQLite database
	//
	relative_path := filepath.Join(report.Fqdn, fmt.Sprintf("%d", report.At_Unix))

	addDB(report, relative_path)

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

			// Don't spam stdout when running test-cases.
			if flag.Lookup("test.v") == nil {
				fmt.Printf("Error: %s\n", err.Error())
			}
		}
	}()

	//
	// Get the node name we're going to show.
	//
	vars := mux.Vars(req)
	id := vars["id"]

	//
	// If the ID is non-numeric we're in trouble.
	//
	reg, _ := regexp.Compile("^([0-9]+)$")
	if !reg.MatchString(id) {
		status = http.StatusInternalServerError
		err = errors.New("The report ID must be numeric")
		return
	}

	//
	// Get the content.
	//
	content, err := getYAML(ReportPrefix, id)
	if err != nil {
		status = http.StatusInternalServerError
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
		err = errors.New("Failed to find asset data/report_handler.template")
		status = http.StatusInternalServerError
		return
	}

	//
	// What kind of reply should we send?
	//
	accept := req.Header.Get("Accept")

	switch accept {
	case "application/json":
		js, err := json.Marshal(report)

		if err != nil {
			status = http.StatusInternalServerError
			return
		}
		res.Header().Set("Content-Type", "application/json")
		res.Write(js)

	case "application/xml":
		x, err := xml.MarshalIndent(report, "", "  ")
		if err != nil {
			status = http.StatusInternalServerError
			return
		}

		res.Header().Set("Content-Type", "application/xml")
		res.Write(x)
	default:
		//
		// Populate & return the template.
		//
		src := string(tmpl)
		t := template.Must(template.New("tmpl").Parse(src))
		t.Execute(res, report)
	}
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

			// Don't spam stdout when running test-cases.
			if flag.Lookup("test.v") == nil {
				fmt.Printf("Error: %s\n", err.Error())
			}
		}
	}()

	//
	// Get the node name we're going to show.
	//
	vars := mux.Vars(req)
	fqdn := vars["fqdn"]

	//
	// Get the reports
	//
	reports, err := getReports(fqdn)

	//
	// Ensure that something was present.
	//
	if (reports == nil) || (len(reports) < 1) {
		status = http.StatusNotFound
		return
	}

	//
	// Handle error(s)
	//
	if err != nil {
		status = http.StatusInternalServerError
		return
	}

	//
	// Annoying struct to allow us to populate our template
	// with both the reports and the fqdn of the host.
	//
	type Pagedata struct {
		Fqdn  string
		Nodes []PuppetReportSummary
	}

	//
	// Populate this structure.
	//
	var x Pagedata
	x.Nodes = reports
	x.Fqdn = fqdn

	//
	// Define a template for the result we'll send to the browser.
	//
	tmpl, err := Asset("data/node_handler.template")
	if err != nil {
		err = errors.New("Failed to find asset data/node_handler.template")
		status = http.StatusInternalServerError
		return
	}

	//
	// What kind of reply should we send?
	//
	accept := req.Header.Get("Accept")

	switch accept {
	case "application/json":
		js, err := json.Marshal(reports)

		if err != nil {
			status = http.StatusInternalServerError
			return
		}
		res.Header().Set("Content-Type", "application/json")
		res.Write(js)

	case "application/xml":
		x, err := xml.MarshalIndent(reports, "", "  ")
		if err != nil {
			status = http.StatusInternalServerError
			return
		}

		res.Header().Set("Content-Type", "application/xml")
		res.Write(x)
	default:
		//
		//  Populate the template and return it.
		//
		src := string(tmpl)
		t := template.Must(template.New("tmpl").Parse(src))
		t.Execute(res, x)
	}
}

//
// Serve the single "/favicon.ico" file.
//
func IconHandler(res http.ResponseWriter, req *http.Request) {
	var (
		status int
		err    error
	)
	defer func() {
		if nil != err {
			http.Error(res, err.Error(), status)

			// Don't spam stdout when running test-cases.
			if flag.Lookup("test.v") == nil {
				fmt.Printf("Error: %s\n", err.Error())
			}
		}
	}()

	//
	// Load the binary-asset.
	//
	data, err := Asset("data/favicon.ico")
	if err != nil {
		err = errors.New("Failed to find asset data/favicon.ico")
		status = http.StatusInternalServerError
		return
	}
	fmt.Printf("Served favicon.ico\n")
	res.Header().Set("Content-Type", "image/vnd.microsoft.icon")
	res.Write(data)
}

//
// Show all the hosts we know about - and their last known state -
// along with a graph of recent states.
//
func IndexHandler(res http.ResponseWriter, req *http.Request) {
	var (
		status int
		err    error
	)
	defer func() {
		if nil != err {
			http.Error(res, err.Error(), status)

			// Don't spam stdout when running test-cases.
			if flag.Lookup("test.v") == nil {
				fmt.Printf("Error: %s\n", err.Error())
			}
		}
	}()

	//
	// Define a template for the result we'll send to the browser.
	//
	tmpl, err := Asset("data/index_handler.template")
	if err != nil {
		err = errors.New("Failed to find asset data/index_handler.template")
		status = http.StatusInternalServerError
		return
	}

	//
	// Annoying struct to allow us to populate our template
	// with both the nodes in the list, and the graph-data
	//
	type Pagedata struct {
		Graph []PuppetHistory
		Nodes []PuppetRuns
	}

	//
	// Get the nodes to show on our front-page
	//
	NodeList, err := getIndexNodes()
	if err != nil {
		status = http.StatusInternalServerError
		return
	}

	//
	// Get the graph-data
	//
	graphs, err := getHistory()
	if err != nil {
		status = http.StatusInternalServerError
		return
	}

	//
	// Populate this structure.
	//
	var x Pagedata
	x.Graph = graphs
	x.Nodes = NodeList

	//
	// What kind of reply should we send?
	//
	accept := req.Header.Get("Accept")

	switch accept {
	case "application/json":
		js, err := json.Marshal(NodeList)

		if err != nil {
			status = http.StatusInternalServerError
			return
		}
		res.Header().Set("Content-Type", "application/json")
		res.Write(js)

	case "application/xml":
		x, err := xml.MarshalIndent(NodeList, "", "  ")
		if err != nil {
			status = http.StatusInternalServerError
			return
		}

		res.Header().Set("Content-Type", "application/xml")
		res.Write(x)
	default:
		//
		//  Populate the template and return it.
		//
		src := string(tmpl)
		t := template.Must(template.New("tmpl").Parse(src))
		t.Execute(res, x)
	}
}

//
//  Entry-point.
//
func cmd_serve(settings serveCmd) {

	//
	// Preserve our prefix
	//
	ReportPrefix = settings.prefix

	//
	// Create a new router and our route-mappings.
	//
	router := mux.NewRouter()

	//
	// API end-points
	//
	router.HandleFunc("/api/state/{state}/", APIState).Methods("GET")
	router.HandleFunc("/api/state/{state}", APIState).Methods("GET")

	//
	//
	//
	router.HandleFunc("/radiator/", RadiatorView).Methods("GET")
	router.HandleFunc("/radiator", RadiatorView).Methods("GET")

	//
	// Upload a new report.
	//
	router.HandleFunc("/upload/", ReportSubmissionHandler).Methods("POST")
	router.HandleFunc("/upload", ReportSubmissionHandler).Methods("POST")

	//
	// Show the recent state of a node.
	//
	router.HandleFunc("/node/{fqdn}/", NodeHandler).Methods("GET")
	router.HandleFunc("/node/{fqdn}", NodeHandler).Methods("GET")

	//
	// Show "everything" about a given run.
	//
	router.HandleFunc("/report/{id}/", ReportHandler).Methods("GET")
	router.HandleFunc("/report/{id}", ReportHandler).Methods("GET")

	//
	// Handle a display of all known nodes, and their last state.
	//
	router.HandleFunc("/", IndexHandler).Methods("GET")

	//
	// FavIcon.
	//
	router.HandleFunc("/favicon.ico", IconHandler).Methods("GET")

	//
	// Bind the router.
	//
	http.Handle("/", router)

	//
	// Launch the server
	//
	fmt.Printf("Launching the server on http://%s:%d\n", settings.bind_host, settings.bind_port)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", settings.bind_host, settings.bind_port), nil)
	if err != nil {
		panic(err)
	}
}

//
// The options set by our command-line flags.
//
type serveCmd struct {
	bind_host string
	bind_port int
	db_file   string
	prefix    string
}

//
// Glue
//
func (*serveCmd) Name() string     { return "serve" }
func (*serveCmd) Synopsis() string { return "Launch the HTTP server." }
func (*serveCmd) Usage() string {
	return `serve [options]:
  Launch the HTTP server for receiving reports & viewing them
`
}

//
// Flag setup
//
func (p *serveCmd) SetFlags(f *flag.FlagSet) {
	f.IntVar(&p.bind_port, "port", 3001, "The port to bind upon.")
	f.StringVar(&p.bind_host, "host", "127.0.0.1", "The IP to listen upon.")
	f.StringVar(&p.db_file, "db-file", "ps.db", "The SQLite database to use.")
	f.StringVar(&p.prefix, "prefix", "./reports/", "The prefix to the local YAML hierarchy.")
}

//
// Entry-point.
//
func (p *serveCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	//
	// Setup the database, by opening a handle, and creating it if
	// missing.
	//
	SetupDB(p.db_file)

	//
	// Start the server
	//
	cmd_serve(*p)

	//
	// All done.
	//
	return subcommands.ExitSuccess
}
