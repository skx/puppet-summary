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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"github.com/gorilla/mux"
	"testing"
	"fmt"
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
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}

	// Check the response body is what we expect.
	expected := "The report ID must be numeric\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got '%v' want '%v'",
			rr.Body.String(), expected)
	}
}

//
// API state must be known must be alphanumeric
//
func TestUnknownAPIState(t *testing.T) {

	r := mux.NewRouter()
	r.HandleFunc("/api/state/{state}", APIState).Methods("GET")
	r.HandleFunc("/api/state/{state}/", APIState).Methods("GET")

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Table driven test
	names := []string{"kate", "matt", "emma"}

	for _, name := range names {
		url := ts.URL + "/api/state/" + name

		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}

		if status := resp.StatusCode; status != http.StatusInternalServerError {
			t.Fatalf("wrong status code: got %d want %d", status, http.StatusOK)
		}
	}
}


//
// API state must be known.
//
// Call each one and assume we'll get a DB-Setup error.
//
func TestKnownAPIState(t *testing.T) {

	r := mux.NewRouter()
	r.HandleFunc("/api/state/{state}", APIState).Methods("GET")
	r.HandleFunc("/api/state/{state}/", APIState).Methods("GET")

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Table driven test
	names := []string{"changed", "failed", "unchanged"}

	for _, name := range names {
		url := ts.URL + "/api/state/" + name

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
			t.Fatalf("wrong status code: got %d want %d", status, http.StatusOK)
		}
		if ( body_str != "SetupDB not called\n" ) {
			t.Fatalf("Unexpected body: '%s'",body)
		}
	}

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
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}

	// Check the response body is what we expect.
	expected := "Must be called via HTTP-POST\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got '%v' want '%v'",
			rr.Body.String(), expected)
	}
}
