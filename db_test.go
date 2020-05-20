//
//  Basic testing of our DB primitives
//

package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
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
	stmt, err := tx.Prepare("INSERT INTO reports(fqdn,environment,yaml_file,executed_at) values(?,?,?,?)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	count := 0

	for count < 30 {
		now := time.Now().Unix()
		days := int64(60 * 60 * 24 * count)
		env := "production"
		if count > 2 {
			env = "test"
		}

		fqdn := fmt.Sprintf("node%d.example.com", count)
		now -= days
		stmt.Exec(fqdn, env, "/../data/valid.yaml", now)
		count++
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
	n.Failed = "0"
	n.Total = "1"
	n.Changed = "2"
	n.Skipped = "3"
	addDB(n, "")

	n.Fqdn = "bar.example.com"
	n.State = "failed"
	n.Runtime = "2.718"
	n.Failed = "0"
	n.Total = "1"
	n.Changed = "2"
	n.Skipped = "3"
	addDB(n, "")

	n.Fqdn = "foo.example.com"
	n.State = "unchanged"
	n.Runtime = "2.718"
	n.Failed = "0"
	n.Total = "1"
	n.Changed = "2"
	n.Skipped = "3"
	addDB(n, "")

	//
	// Here we're trying to fake an orphaned node.
	//
	// When a report is added the exected_at field is set to
	// "time.Now().Unix()".  To make an orphaned record we need
	// to change that to some time >24 days ago.
	//
	// We do that by finding the last report-ID, and then editing
	// the field.
	//
	var maxID string
	row := db.QueryRow("SELECT MAX(id) FROM reports")
	err := row.Scan(&maxID)

	switch {
	case err == sql.ErrNoRows:
	case err != nil:
		panic("failed to find max report ID")
	default:
	}

	//
	// Now we can change the executed_at field of that last
	// addition
	//
	sqlStmt := fmt.Sprintf("UPDATE reports SET executed_at=300 WHERE id=%s",
		maxID)
	_, err = db.Exec(sqlStmt)
	if err != nil {
		panic("Failed to change report ")
	}

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
// Test that functions return errors if setup hasn't been called.
//
func TestMissingInit(t *testing.T) {

	//
	// Regexp to match the error we expect to receive.
	//
	reg, _ := regexp.Compile("SetupDB not called")

	var x PuppetReport
	err := addDB(x, "")
	if !reg.MatchString(err.Error()) {
		t.Errorf("Got wrong error: %v", err)
	}

	_, err = countReports()
	if !reg.MatchString(err.Error()) {
		t.Errorf("Got wrong error: %v", err)
	}

	_, err = getYAML("", "")
	if !reg.MatchString(err.Error()) {
		t.Errorf("Got wrong error: %v", err)
	}

	_, err = getIndexNodes("")
	if !reg.MatchString(err.Error()) {
		t.Errorf("Got wrong error: %v", err)
	}

	_, err = getReports("example.com")
	if !reg.MatchString(err.Error()) {
		t.Errorf("Got wrong error: %v", err)
	}

	_, err = getHistory("")
	if !reg.MatchString(err.Error()) {
		t.Errorf("Got wrong error: %v", err)
	}

	err = pruneReports("", "", 3, false)
	if !reg.MatchString(err.Error()) {
		t.Errorf("Got wrong error: %v", err)
	}

}

//
// Test creating a new DB fails when given a directory.
//
func TestBogusInit(t *testing.T) {

	// Create a fake database
	FakeDB()

	err := SetupDB(path)

	if err == nil {
		t.Errorf("We should have seen a create-error")
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
	if old != 30 {
		t.Errorf("We have %d reports, not 30", old)
	}

	//
	// Run the prune
	//
	pruneReports("", "", 5, false)

	//
	// Count them again
	//
	new, err := countReports()
	if err != nil {
		t.Errorf("Error counting reports")
	}

	if new != 6 {
		t.Errorf("We have %d reports, not 6", new)
	}

	//
	// Test pruning of specific environments by pruning all test envs
	//
	pruneReports("test", "", 0, false)

	//
	// Final count
	//
	fnl, err := countReports()

	if err != nil {
		t.Errorf("Error counting reports")
	}
	if fnl != 3 {
		t.Errorf("We have %d production environment reports, not 3", fnl)
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
// Add some nodes and verify they are reaped, if unchanged.
//
func TestPruneUnchanged(t *testing.T) {

	// Create a fake database
	FakeDB()

	// With some reports.
	addFakeNodes()

	//
	// Count records and assume we have some.
	//
	old, err := countReports()

	if err != nil {
		t.Errorf("Error counting reports")
	}
	if old != 3 {
		t.Errorf("We have %d reports, not 3", old)
	}

	//
	// Run the prune
	//
	pruneUnchanged("", "", false)

	//
	// Count them again
	//
	new, err := countReports()
	if err != nil {
		t.Errorf("Error counting reports")
	}

	//
	// The value won't have changed.
	//
	if new != old {
		t.Errorf("We have %d reports, not %d", new, old)
	}

	//
	// But we'll expect that several will have updated
	// to show that their paths have been changed to 'reaped'
	//
	pruned, err := countUnchangedAndReapedReports()
	if err != nil {
		t.Errorf("Error counting reaped reports")
	}

	if pruned != 1 {
		t.Errorf("We have %d pruned reports, not 1", pruned)
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
	runs, err := getIndexNodes("")
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
func TestMissiongReport(t *testing.T) {
	FakeDB()

	_, err := getYAML("", "")

	reg, _ := regexp.Compile("failed to find report with specified ID")
	if !reg.MatchString(err.Error()) {
		t.Errorf("Got wrong error: %v", err)
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
	runs, err := getHistory("")
	if err != nil {
		t.Errorf("getHistory failed: %v", err)
	}

	//
	// Should have 2 runs, becase we have only one unique date..
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
