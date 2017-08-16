package main

import (
	"os"
	"testing"
)

func TestMetrics(t *testing.T) {

	// Create a fake database
	FakeDB()

	// Add some hosts.
	addFakeNodes()

	// Get the metrics
	metrics := getMetrics()

	// Now test we can find things.
	if len(metrics) != 4 {
		t.Errorf("Unexpected metrics-size: %v", len(metrics))
	}

	// Some values
	if metrics["state.changed"] != "1" {
		t.Errorf("Unexpected metrics value")
	}
	if metrics["state.unchanged"] != "0" {
		t.Errorf("Unexpected metrics value")
	}
	if metrics["state.failed"] != "1" {
		t.Errorf("Unexpected metrics value")
	}
	if metrics["state.orphaned"] != "0" {
		t.Errorf("Unexpected metrics value")
	}

	//
	// Cleanup here because otherwise later tests will
	// see an active/valid DB-handle.
	//
	db.Close()
	db = nil
	os.RemoveAll(path)
}
