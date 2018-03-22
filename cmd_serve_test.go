//
// Simple testing of the HTTP-server
//
//
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/gorilla/mux"
)

//
// Report IDs must be alphanumeric.  Submit some bogus requests to
// ensure they fail with a suitable error-message.
//
func TestNonNumericReport(t *testing.T) {
	router := mux.NewRouter()
	router.HandleFunc("/report/{id}/", ReportHandler).Methods("GET")
	router.HandleFunc("/report/{id}", ReportHandler).Methods("GET")

	// Table driven test
	ids := []string{"/report/1a", "/report/steve", "/report/bob/", "/report/3a.3/"}

	for _, id := range ids {
		req, err := http.NewRequest("GET", id, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Check the status code is what we expect.
		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("Unexpected status-code: %v", status)
		}

		// Check the response body is what we expect.
		expected := "The report ID must be numeric\n"
		if rr.Body.String() != expected {
			t.Errorf("handler returned unexpected body: got '%v' want '%v'",
				rr.Body.String(), expected)
		}
	}
}

//
// API-state must use known values.  Submit some bogus values to ensure
// a suitable error is returned.
//
func TestUknownAPIState(t *testing.T) {

	// Wire up the route
	r := mux.NewRouter()
	r.HandleFunc("/api/state/{state}", APIState).Methods("GET")
	r.HandleFunc("/api/state/{state}/", APIState).Methods("GET")

	// Get the test-server
	ts := httptest.NewServer(r)
	defer ts.Close()

	// These are all bogus
	states := []string{"foo", "bart", "liza", "moi kissa", "steve/"}

	for _, state := range states {
		url := ts.URL + "/api/state/" + state

		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}

		//
		// Get the body
		//
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			t.Errorf("Failed to read response-body %v\n", err)
		}

		content := fmt.Sprintf("%s", body)
		if status := resp.StatusCode; status != http.StatusInternalServerError {
			t.Errorf("Unexpected status-code: %v", status)
		}
		if content != "Invalid state\n" {
			t.Fatalf("Unexpected body: '%s'", body)
		}
	}

}

//
// Test that our report-view returns content that seems reasonable,
// in all three cases:
//
//   * text/html
//   * application/json
//   * application/xml
//
func TestReportView(t *testing.T) {

	// Create a fake database
	FakeDB()

	// Add some data.
	addFakeReports()

	//
	// We'll make one test for each supported content-type
	//
	type TestCase struct {
		Type     string
		Response string
	}

	//
	// The tests
	//
	tests := []TestCase{
		{"text/html", "Report of www.steve.org.uk which ran 2017-07-29 23:17:01"},
		{"application/json", "\"State\":\"unchanged\","},
		{"application/xml", "<State>unchanged</State>"}}

	//
	// Run each one.
	//
	for _, test := range tests {

		//
		// Create a router.
		//
		router := mux.NewRouter()
		router.HandleFunc("/report/{id}/", ReportHandler).Methods("GET")
		router.HandleFunc("/report/{id}", ReportHandler).Methods("GET")

		//
		// Get a valid report ID, and use it to build a URL.
		//
		id, _ := validReportID()
		url := fmt.Sprintf("/report/%d", id)

		//
		// Make the request, with the appropriate Accept: header
		//
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Add("Accept", test.Type)

		//
		// Fake out the request
		//
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		//
		// Test the status-code is OK
		//
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Unexpected status-code: %v", status)
		}

		//
		// Test that the body contained our expected content.
		//
		if !strings.Contains(rr.Body.String(), test.Response) {
			t.Fatalf("Unexpected body: '%s'", rr.Body.String())
		}
	}

	//
	// Cleanup here because otherwise later tests will
	// see an active/valid DB-handle.
	//
	db.Close()
	db = nil
	os.RemoveAll(path)
}

//
// API state must be known.
//
func TestKnownAPIState(t *testing.T) {

	// Create a fake database
	FakeDB()

	// Add some data.
	addFakeNodes()

	// Wire up the router.
	r := mux.NewRouter()
	r.HandleFunc("/api/state/{state}", APIState).Methods("GET")
	r.HandleFunc("/api/state/{state}/", APIState).Methods("GET")

	// Get the test-server
	ts := httptest.NewServer(r)
	defer ts.Close()

	//
	// We'll make one test for each known-state
	//
	type TestCase struct {
		State    string
		Response string
	}

	tests := []TestCase{
		{"changed", "[\"foo.example.com\"]"},
		{"unchanged", "[]"},
		{"failed", "[\"bar.example.com\"]"},
		{"orphaned", "[]"}}

	//
	// Run each one.
	//
	for _, test := range tests {

		//
		// Make the request
		//
		url := ts.URL + "/api/state/" + test.State

		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}

		//
		// Get the body
		//
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			t.Errorf("Failed to read response-body %v\n", err)
		}

		content := fmt.Sprintf("%s", body)

		if status := resp.StatusCode; status != http.StatusOK {
			t.Errorf("Unexpected status-code: %v", status)
		}
		if content != test.Response {
			t.Fatalf("Unexpected body: '%s'", body)
		}
	}

	//
	// Cleanup here because otherwise later tests will
	// see an active/valid DB-handle.
	//
	db.Close()
	db = nil
	os.RemoveAll(path)

}

//
// API state should accept XML, JSON, and plain-text
//
func TestAPITypes(t *testing.T) {

	// Create a fake database
	FakeDB()

	// Add some data.
	addFakeNodes()

	// Wire up the router.
	r := mux.NewRouter()
	r.HandleFunc("/api/state/{state}", APIState).Methods("GET")
	r.HandleFunc("/api/state/{state}/", APIState).Methods("GET")

	// Get the test-server
	ts := httptest.NewServer(r)
	defer ts.Close()

	//
	// We'll make one test for each known-state
	//
	type TestCase struct {
		Type     string
		Response string
	}

	tests := []TestCase{
		{"application/json", "[\"foo.example.com\"]"},
		{"application/xml", "<string>foo.example.com</string>"},
		{"text/plain", "foo.example.com\n"},
		{"", "[\"foo.example.com\"]"},
	}

	//
	// Run each one.
	//
	for _, test := range tests {

		//
		// Make the request
		//
		url := ts.URL + "/api/state/changed?accept=" + test.Type

		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}

		//
		// Get the body
		//
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			t.Errorf("Failed to read response-body %v\n", err)
		}

		content := fmt.Sprintf("%s", body)

		if status := resp.StatusCode; status != http.StatusOK {
			t.Errorf("Unexpected status-code: %v", status)
		}
		if content != test.Response {
			t.Fatalf("Unexpected body for %s: '%s'", test.Type, body)
		}
	}

	//
	// Cleanup here because otherwise later tests will
	// see an active/valid DB-handle.
	//
	db.Close()
	db = nil
	os.RemoveAll(path)

}

//
// Searching must be done via a POST.
//
func TestSearchMethod(t *testing.T) {

	req, err := http.NewRequest("GET", "/search", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(SearchHandler)

	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Unexpected status-code: %v", status)
	}

	// Check the response body is what we expect.
	expected := "Must be called via HTTP-POST\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got '%v' want '%v'",
			rr.Body.String(), expected)
	}

}

//
// The search handler must have a term-parameter.
//
func TestSearchEmpty(t *testing.T) {

	req, err := http.NewRequest("POST", "/search", bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(SearchHandler)

	// Our handlers satisfy http.Handler, so we can call
	// their ServeHTTP method directly and pass in our
	// Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Unexpected status-code: %v", status)
	}

	// Check the response body is what we expect.
	expected := "Missing search term\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got '%v' want '%v'",
			rr.Body.String(), expected)
	}
}

//
// The search handler should run a search
//
func TestSearch(t *testing.T) {

	// Create a fake database
	FakeDB()

	// Add some data.
	addFakeNodes()

	// The term we're going to search for: "example"
	data := url.Values{}
	data.Set("term", "example")

	req, err := http.NewRequest("POST", "/search", bytes.NewBufferString(data.Encode()))
	if err != nil {
		t.Fatal(err)
	}

	//
	// Ensure we're POSTing a FORM
	//
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(SearchHandler)

	// Our handlers satisfy http.Handler, so we can call
	// their ServeHTTP method directly and pass in our
	// Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Unexpected status-code: %v", status)
	}

	// Check the response body is what we expect.
	expected := "/node/bar.example.com"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Fatalf("Unexpected body: '%s'", rr.Body.String())
	}

	//
	// Cleanup here because otherwise later tests will
	// see an active/valid DB-handle.
	//
	db.Close()
	db = nil
	os.RemoveAll(path)
}

//
// Submitting reports must be done via a POST.
//
func TestUploadReportMethod(t *testing.T) {

	req, err := http.NewRequest("GET", "/upload", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(ReportSubmissionHandler)

	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Unexpected status-code: %v", status)
	}

	// Check the response body is what we expect.
	expected := "Must be called via HTTP-POST\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got '%v' want '%v'",
			rr.Body.String(), expected)
	}

}

//
// Submitting a pre-cooked report should succeed.
//
func TestUploadReport(t *testing.T) {

	// Create a fake database
	FakeDB()

	// Ensure we point our report-upload directory at
	// our temporary location.
	ReportPrefix = path

	//
	// Read the YAML file.
	//
	tmpl, err := Asset("data/valid.yaml")
	if err != nil {
		t.Fatal(err)
	}

	//
	// Call this two times.
	//
	count := 0

	//
	// The two expected results
	//
	expected := []string{"{\"host\":\"www.steve.org.uk\"}", "Ignoring duplicate submission"}

	for count < 2 {
		req, err := http.NewRequest("POST", "/upload", bytes.NewReader(tmpl))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(ReportSubmissionHandler)

		// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
		// directly and pass in our Request and ResponseRecorder.
		handler.ServeHTTP(rr, req)

		// Check the status code is what we expect.
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Unexpected status-code: %v", status)
		}

		if rr.Body.String() != expected[count] {
			t.Errorf("Body was '%v' we wanted '%v'",
				rr.Body.String(), expected[count])
		}

		count++
	}

	//
	// Cleanup here because otherwise later tests will
	// see an active/valid DB-handle.
	//
	db.Close()
	db = nil
	os.RemoveAll(path)
}

//
// Submitting a pre-cooked report which is bogus should fail.
//
func TestUploadBogusReport(t *testing.T) {

	// Create a fake database
	FakeDB()

	// Ensure we point our report-upload directory at
	// our temporary location.
	ReportPrefix = path

	//
	// Read the YAML file.
	//
	tmpl, err := Asset("data/valid.yaml")
	if err != nil {
		t.Fatal(err)
	}

	//
	// Upper-case the YAML
	//
	for i := range tmpl {
		tmpl[i] = byte(unicode.ToUpper(rune(tmpl[i])))
	}

	req, err := http.NewRequest("POST", "/upload", bytes.NewReader(tmpl))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(ReportSubmissionHandler)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("Unexpected status-code: %v", status)
	}

	// Check the response body is what we expect.
	expected := "Failed to get 'host' from YAML\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got '%v' want '%v'",
			rr.Body.String(), expected)
	}

	//
	// Cleanup here because otherwise later tests will
	// see an active/valid DB-handle.
	//
	db.Close()
	db = nil
	os.RemoveAll(path)
}

//
// Unknown-nodes are handled.
//
func TestUnknownNode(t *testing.T) {

	// Create a fake database
	FakeDB()

	// Add some data.
	addFakeNodes()

	// Wire up the router.
	r := mux.NewRouter()
	r.HandleFunc("/node/{fqdn}", NodeHandler).Methods("GET")

	// Get the test-server
	ts := httptest.NewServer(r)
	defer ts.Close()

	//
	// Test a bogus name.
	//
	url := ts.URL + "/node/missing.invalid.tld"

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}

	//
	// Get the body
	//
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		t.Errorf("Failed to read response-body %v\n", err)
	}

	content := fmt.Sprintf("%s", body)

	if status := resp.StatusCode; status != http.StatusNotFound {
		t.Errorf("Unexpected status-code: %v", status)
	}
	if content != "Failed to find reports for missing.invalid.tld\n" {
		t.Fatalf("Unexpected body: '%s'", body)
	}

	//
	// Cleanup here because otherwise later tests will
	// see an active/valid DB-handle.
	//
	db.Close()
	db = nil
	os.RemoveAll(path)

}

//
// Test that our node-view returns content that seems reasonable,
// in all three cases:
//
//   * text/html
//   * application/json
//   * application/xml
//
//
func TestKnownNode(t *testing.T) {

	// Create a fake database
	FakeDB()

	// Add some data.
	addFakeNodes()

	//
	// We'll make one test for each supported content-type
	//
	type TestCase struct {
		Type     string
		Response string
	}

	//
	// The tests
	//
	tests := []TestCase{
		{"text/html", "3.134"},
		{"application/json", "\"State\":\"unchanged\","},
		{"application/xml", "<PuppetReportSummary>"}}

	//
	// Run each one.
	//
	for _, test := range tests {

		//
		// Create a router.
		//
		router := mux.NewRouter()
		router.HandleFunc("/node/{fqdn}/", NodeHandler).Methods("GET")
		router.HandleFunc("/node/{fqdn}", NodeHandler).Methods("GET")

		//
		// Make the request, with the appropriate Accept: header
		//
		req, err := http.NewRequest("GET", "/node/foo.example.com", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Add("Accept", test.Type)

		//
		// Fake out the request
		//
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		//
		// Test the status-code is OK
		//
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Unexpected status-code: %v", status)
		}

		//
		// Test that the body contained our expected content.
		//
		if !strings.Contains(rr.Body.String(), test.Response) {
			t.Fatalf("Unexpected body: '%s'", rr.Body.String())
		}
	}

	//
	// Cleanup here because otherwise later tests will
	// see an active/valid DB-handle.
	//
	db.Close()
	db = nil
	os.RemoveAll(path)

}

//
// Test that our index-view returns content that seems reasonable,
// in all three cases:
//
//   * text/html
//   * application/json
//   * application/xml
//
func TestIndexView(t *testing.T) {

	// Create a fake database
	FakeDB()

	// Add some data.
	addFakeNodes()

	//
	// We'll make one test for each supported content-type
	//
	type TestCase struct {
		Type     string
		Response string
	}

	//
	// The tests
	//
	tests := []TestCase{
		{"text/html", "foo.example.com"},
		{"application/json", "\"State\":\"failed\","},
		{"application/xml", "<PuppetRuns>"}}

	//
	// Run each one.
	//
	for _, test := range tests {

		//
		// Make the request, with the appropriate Accept: header
		//
		req, err := http.NewRequest("GET", "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Add("Accept", test.Type)

		//
		// Fake it out
		//
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(IndexHandler)
		handler.ServeHTTP(rr, req)

		//
		// Test the status-code is OK
		//
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Unexpected status-code: %v", status)
		}

		//
		// Test that the body contained our expected content.
		//
		if !strings.Contains(rr.Body.String(), test.Response) {
			t.Fatalf("Unexpected body: '%s'", rr.Body.String())
		}
	}

	//
	// Cleanup here because otherwise later tests will
	// see an active/valid DB-handle.
	//
	db.Close()
	db = nil
	os.RemoveAll(path)

}

//
// Our icon is correct.
//
func TestFavIcon(t *testing.T) {

	// Wire up the router.
	r := mux.NewRouter()
	r.HandleFunc("/favicon.ico", IconHandler).Methods("GET")

	// Get the test-server
	ts := httptest.NewServer(r)
	defer ts.Close()

	//
	// Get the icon
	//
	url := ts.URL + "/favicon.ico"

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}

	//
	// Get the body
	//
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		t.Errorf("Failed to read response-body %v\n", err)
	}

	//
	// Test the size is that we expect.
	//
	if len(body) != 1150 {
		t.Errorf("Icon was the wrong size %v\n", len(body))
	}

	//
	// Test that the content-type was what we expect.
	//
	headers := resp.Header
	ctype := headers["Content-Type"][0]
	if ctype != "image/vnd.microsoft.icon" {
		t.Errorf("content type header does not match: got %v", ctype)
	}

	//
	// Now test we were served the data we expect.
	//
	// Load the resource
	//
	tmpl, err := Asset("data/favicon.ico")
	if err != nil {
		t.Fatal(err)
	}

	//
	// Compare byte by byte
	//
	for _, b := range tmpl {
		if body[b] != tmpl[b] {
			t.Errorf("favicon.ico content is corrupt?")
		}
	}
}

//
// Test that our radiator-view returns content that seems reasonable,
// in all three cases:
//
//   * text/html
//   * application/json
//   * application/xml
//
func TestRadiatorView(t *testing.T) {

	// Create a fake database
	FakeDB()

	// Add some data.
	addFakeNodes()

	//
	// We'll make one test for each supported content-type
	//
	type TestCase struct {
		Type     string
		Response string
	}

	//
	// The tests
	//
	tests := []TestCase{
		{"text/html", "<p class=\"percent\" style=\"width: 50%\">"},
		{"application/json", "\"State\":\"failed\","},
		{"application/xml", "<PuppetState>"}}

	//
	// Run each one.
	//
	for _, test := range tests {

		//
		// Make the request, with the appropriate Accept: header
		//
		req, err := http.NewRequest("GET", "/radiator/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Add("Accept", test.Type)

		//
		// Fake it out
		//
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(RadiatorView)
		handler.ServeHTTP(rr, req)

		//
		// Test the status-code is OK
		//
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Unexpected status-code: %v", status)
		}

		//
		// Test that the body contained our expected content.
		//
		if !strings.Contains(rr.Body.String(), test.Response) {
			t.Fatalf("Unexpected body: '%s'", rr.Body.String())
		}
	}

	//
	// Cleanup here because otherwise later tests will
	// see an active/valid DB-handle.
	//
	db.Close()
	db = nil
	os.RemoveAll(path)

}
