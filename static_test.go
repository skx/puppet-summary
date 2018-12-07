//
// Simple testing of our embedded resource.
//

package main

import (
	"io/ioutil"
	"strings"
	"testing"
)

//
// Test that we have one embedded resource.
//
func TestResourceCount(t *testing.T) {
	out := getResources()
	if len(out) != 8 {
		t.Errorf("We expected seven resources but found %d.", len(out))
	}
}

//
// Test that each of our resources is identical to the master
// version.
//
func TestResourceMatches(t *testing.T) {

	//
	// For each resource
	//
	all := getResources()

	for _, entry := range all {

		//
		// Get the data from our embedded copy
		//
		data, err := getResource(entry.Filename)
		if err != nil {
			t.Errorf("Loading our resource failed:%s", entry.Filename)
		}

		//
		// Get the data from our master-copy.
		//
		master, err := ioutil.ReadFile(entry.Filename)
		if err != nil {
			t.Errorf("Loading our master-resource failed:%s", entry.Filename)
		}

		//
		// Test the lengths match
		//
		if len(master) != len(data) {
			t.Errorf("Embedded and real resources have different sizes.")
		}

		//
		// Now test the length is the same as generated in the file.
		//
		for i, o := range all {
			if o.Filename == entry.Filename {
				if len(master) != getResources()[i].Length {
					t.Errorf("Data length didn't match the generated size")
				}
			}
		}

		//
		// Test the data-matches
		//
		if string(master) != string(data) {
			t.Errorf("Embedded and real resources have different content.")
		}
	}
}

//
// Test that a missing resource is handled.
//
func TestMissingResource(t *testing.T) {

	//
	// Get the data from our embedded copy
	//
	data, err := getResource("moi/kissa")
	if data != nil {
		t.Errorf("We expected to find no data, but got some.")
	}
	if err == nil {
		t.Errorf("We expected an error loading a missing resource, but got none.")
	}
	if !strings.Contains(err.Error(), "Failed to find resource") {
		t.Errorf("Error message differed from expectations.")
	}
}
