package main

import (
	"os"
	"testing"
)

func TestPruneCommand(t *testing.T) {

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

	tmp := pruneCmd{days: 5, verbose: false}
	runPrune(tmp)

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
