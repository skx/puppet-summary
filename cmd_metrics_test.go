package main

import (
	"bytes"
	"context"
	"os"
	"strings"
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

//
// Actually attempt to send the metrics to stdout.
//
func TestMetricNop(t *testing.T) {

	//
	// Fake out the STDOUT
	//
	bak := out
	out = new(bytes.Buffer)
	defer func() { out = bak }()

	// Create a fake database
	FakeDB()

	// Add some hosts.
	addFakeNodes()

	//
	// Dump our metrics to STDOUT, due to `nop`, which will end up
	// in our faux buffer.
	s := metricsCmd{nop: true}
	s.Execute(context.TODO(), nil)

	//
	// Now see what we got.
	//
	read := out.(*bytes.Buffer).String()

	//
	// And test it against each of the things we
	// expect.
	//
	// NOTE: We have to do this as the output is ordered
	// randomly.
	//
	desired := []string{".state.changed 0",
		".state.failed 0",
		".state.orphaned 0",
		".state.unchanged 0"}

	for _, str := range desired {
		if !strings.Contains(read, str) {
			t.Errorf("Unexpected metric-output - %s", read)
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
