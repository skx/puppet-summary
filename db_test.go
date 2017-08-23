//
//  Basic testing of our DB primitives
//

package main

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

//
//  Temporary location for database
//
var path string

//
// Create a temporary database
//
func FakeDB() {
	p, err := ioutil.TempDir(os.TempDir(), "prefix")
	if err == nil {
		path = p
	}

	//
	// Setup the tables.
	//
	SetupDB(p + "/db.sql")

}

//
// Add some fake reports
//
func addFakeReports() {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	//
	// Add some records
	stmt, err := tx.Prepare("INSERT INTO reports(yaml_file,executed_at) values(?,?)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	count := 0

	for count < 10 {
		now := time.Now().Unix()
		days := int64(60 * 60 * 24 * count)

		now -= days
		stmt.Exec("/../data/valid.yaml", now)
		count += 1
	}
	tx.Commit()
}

//
// Add some (repeated) nodes in various states
//
func addFakeNodes() {

	var n PuppetReport
	n.Fqdn = "foo.example.com"
	n.State = "changed"
	n.Runtime = "3.134"
	n.At_Unix = time.Now().Unix()
	n.Failed = "0"
	n.Total = "1"
	n.Changed = "2"
	n.Skipped = "3"
	addDB(n, "")

	n.Fqdn = "bar.example.com"
	n.State = "failed"
	n.Runtime = "2.718"
	n.At_Unix = time.Now().Unix()
	n.Failed = "0"
	n.Total = "1"
	n.Changed = "2"
	n.Skipped = "3"
	addDB(n, "")

	n.Fqdn = "foo.example.com"
	n.State = "unchanged"
	n.Runtime = "2.718"
	n.At_Unix = time.Now().Unix() - 100
	n.Failed = "0"
	n.Total = "1"
	n.Changed = "2"
	n.Skipped = "3"
	addDB(n, "")

}

//
// Get a valid report ID.
//
func validReportID() (int, error) {
	var count int
	row := db.QueryRow("SELECT MAX(id) FROM reports")
	err := row.Scan(&count)
	return count, err
}

//
// Add some nodes and verify they are reaped.
//
func TestPrune(t *testing.T) {

	// Create a fake database
	FakeDB()

	// With some reports.
	addFakeReports()

	//
	// Count records and assume we have some.
	//
	old, err := countReports()

	if err != nil {
		t.Errorf("Error counting reports")
	}
	if old != 10 {
		t.Errorf("We have %d reports, not 10", old)
	}

	//
	// Run the prune
	//
	pruneReports("", 5, false)

	//
	// Count them again
	//
	new, err := countReports()
	if err != nil {
		t.Errorf("Error counting reports")
	}

	if new != 6 {
		t.Errorf("We have %d reports, not 5", new)
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
//  Test the index nodes are valid
//
func TestIndex(t *testing.T) {

	//
	// Create a fake database.
	//
	FakeDB()

	// Add some fake nodes.
	addFakeNodes()

	//
	// We have three fake nodes now, two of which have the
	// same hostname.
	//
	runs, err := getIndexNodes()
	if err != nil {
		t.Errorf("getIndexNodes failed: %v", err)
	}

	//
	// Should have two side
	//
	if len(runs) != 2 {
		t.Errorf("getIndexNodes returned wrong number of results: %d", len(runs))
	}

	//
	// But three reports
	//
	total, err := countReports()
	if err != nil {
		t.Errorf("Failed to count reports")
	}

	if total != 3 {
		t.Errorf("We found the wrong number of reports, %d", total)
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
//  Test the report-run are valid
//
func TestReports(t *testing.T) {

	//
	// Add fake reports.
	//
	FakeDB()
	addFakeNodes()

	//
	// We have three fake nodes now, two of which have the
	// same hostname.
	//
	runs, err := getReports("foo.example.com")
	if err != nil {
		t.Errorf("getReports failed: %v", err)
	}

	//
	// Should have two runs against the host
	//
	if len(runs) != 2 {
		t.Errorf("getReports returned wrong number of results: %d", len(runs))
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
//  Test the report-run are valid
//
func TestHistory(t *testing.T) {

	//
	// Add fake reports.
	//
	FakeDB()
	addFakeNodes()

	//
	// We have three fake nodes now, two of which have the same hostname.
	//
	runs, err := getHistory()
	if err != nil {
		t.Errorf("getHistory failed: %v", err)
	}

	//
	// Should have 1 run, becase we have only one unique date..
	//
	if len(runs) != 1 {
		t.Errorf("getReports returned wrong number of results: %d", len(runs))
	}

	//
	// Cleanup here because otherwise later tests will
	// see an active/valid DB-handle.
	//
	db.Close()
	db = nil
	os.RemoveAll(path)
}
