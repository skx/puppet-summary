//
// Simple testing of the HTTP-server
//
//
package main

import (
	"bytes"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"unicode"
)

//
// Report IDs must be alphanumeric
//
func TestNonNumericReport(t *testing.T) {
	req, err := http.NewRequest("GET", "/report/3a", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(ReportHandler)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

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

//
// API-state must use known values.
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
	states := []string{"foo", "bart", "liza", "moi kiss"}

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
// Reports must be numeric.
//
func TestNumericReports(t *testing.T) {

	// Create a fake database
	FakeDB()

	// Add some data.
	addFakeReports()

	// Wire up the router.
	r := mux.NewRouter()
	r.HandleFunc("/report/{id}", ReportHandler).Methods("GET")

	// Get the test-server
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Table driven test
	ids := []string{"1", "100", "303021"}

	for _, id := range ids {
		url := ts.URL + "/report/" + id

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
			t.Fatalf("Unexpected status code: %d", status)
		}

		if content != "Failed to find report with specified ID\n" {
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
	// Get the unchanged result - which should be foo.example.com
	//
	url := ts.URL + "/api/state/changed"

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
	if content != "[\"foo.example.com\"]" {
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
// Submitting reports must be done via a POST.
//
func TestUploadReportMethod(t *testing.T) {

	req, err := http.NewRequest("GET", "/upload", nil)
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
	expected := "Must be called via HTTP-POST\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got '%v' want '%v'",
			rr.Body.String(), expected)
	}

}

//
// Submitting a pre-cooked method.
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

		count += 1
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
// Submitting a pre-cooked method.
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
// Valid-node is handled.
//
func TestKnownNode(t *testing.T) {

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
	// Test a known-good node-name
	//
	url := ts.URL + "/node/foo.example.com"

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

	//
	// Test that the body contained our run-time(s).
	//
	if !strings.Contains(content, "3.134") {
		t.Fatalf("Unexpected body: '%s'", body)
	}
	if !strings.Contains(content, "2.718") {
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
// Our index contains the nodes we expect.
//
func TestIndexView(t *testing.T) {

	// Create a fake database
	FakeDB()

	// Add some data.
	addFakeNodes()

	// Wire up the router.
	r := mux.NewRouter()
	r.HandleFunc("/", IndexHandler).Methods("GET")

	// Get the test-server
	ts := httptest.NewServer(r)
	defer ts.Close()

	//
	// Get the front-page
	//
	url := ts.URL + "/"

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

	//
	// Test that the body contained our run-time(s).
	//
	if !strings.Contains(content, "foo.example.com") {
		t.Fatalf("Unexpected body: '%s'", body)
	}
	if !strings.Contains(content, "bar.example.com") {
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
// Our radiator-view contains a 50% count.   Yeah this test is woolly.
//
func TestRadiatorView(t *testing.T) {

	// Create a fake database
	FakeDB()

	// Add some data.
	addFakeNodes()

	// Wire up the router.
	r := mux.NewRouter()
	r.HandleFunc("/radiator/", RadiatorView).Methods("GET")

	// Get the test-server
	ts := httptest.NewServer(r)
	defer ts.Close()

	//
	// Get the front-page
	//
	url := ts.URL + "/radiator/"

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

	//
	// Test that the body contained 50% count.
	//
	if !strings.Contains(content, " <p class=\"percent\" style=\"width: 50%\">") {
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
