//
// Simple testing of the HTTP-server
//
//  TODO:
//   * Add a setup-method to create a temporary DB
//   * Add a cleanup method to remove that.
//   * Populate some records.
//   * Ensure the handlers all return valid content.
//
package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
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

		body_str := fmt.Sprintf("%s", body)
		if status := resp.StatusCode; status != http.StatusInternalServerError {
			t.Errorf("Unexpected status-code: %v", status)
		}
		if body_str != "Invalid state\n" {
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
		body_str := fmt.Sprintf("%s", body)

		if status := resp.StatusCode; status != http.StatusInternalServerError {
			t.Fatalf("Unexpected status code: %d", status)
		}

		if body_str != "Failed to find report with specified ID\n" {
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
	body_str := fmt.Sprintf("%s", body)

	if status := resp.StatusCode; status != http.StatusOK {
		t.Errorf("Unexpected status-code: %v", status)
	}
	if body_str != "[\"foo.example.com\"]" {
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
