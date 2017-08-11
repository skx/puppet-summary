//
//  Our YAML parser is our single biggest potential source of
// failure - whether users give us bogus input, or puppet-versions
// change what thye submit.
//
//  We should have good, thorough, and real test-cases here.
//
//

package main

import (
	"fmt"
	"regexp"
	"testing"
)

//
// Test that we can handle dates of various forms.
//
func TestYamlDates(t *testing.T) {

	tests := []string{"---\ntime: '2017-03-10T10:22:33.659245699+00:00'\nhost: bart\n",
		"---\ntime: 2017-03-10 10:22:33.493526494 +00:00\nhost: foo\n"}

	for _, input := range tests {
		fmt.Printf("Testing: %s\n", input)

		//
		// Error will be set here, since we only supply
		// `host` + `time` we'll expect something like
		//
		// "Failed to get `status' from YAML
		//
		node, _ := ParsePuppetReport([]byte(input))

		if node.At != "2017-03-10 10:22:33" {
			t.Errorf("Invalid time result, got '%s'", node.At)
		}
		if node.At_Unix != 1489141353 {
			t.Errorf("Time was wrong number of epoch seconds:", node.At_Unix)
		}

	}

}

//
// Test that we can handle filter out bogus hostnames.
//
// Here we look for an exception of the form "blah invalid|missing host"
// to know whether we passed/failed.
//
func TestHostName(t *testing.T) {

	//
	// Test-cases
	//
	type HostTest struct {
		hostname string
		valid    bool
	}

	//
	// Possible Hostnames
	//
	fail := []HostTest{
		{"../../../etc/passwd%00", false},
		{"node1.example.com../../../etc", false},
		{"steve_example com", false},
		{"node1./example.com", false},
		{"steve1.example.com", true},
		{"steve-example.com", true},

		{"example3-3_2.com", true}}

	//
	// For each test-case
	//
	for _, input := range fail {

		//
		// Build up YAML
		//
		var tmp string

		tmp = "---\n" +
			"host: " + input.hostname

		//
		// Parse it.
		//
		_, err := ParsePuppetReport([]byte(tmp))

		//
		// Host-regexp.
		//
		reg, _ := regexp.Compile("host")

		//
		// Do we expect this to pass/fail?
		//
		if input.valid {

			if reg.MatchString(err.Error()) {
				t.Errorf("Expected no error relating to 'host', but got one: %v", err)
			}
		} else {

			//
			// We expect this to fail.  Did it?
			//
			if !reg.MatchString(err.Error()) {
				t.Errorf("Expected an error relating to 'host', but didn't: %v", err)
			}
		}
	}
}
