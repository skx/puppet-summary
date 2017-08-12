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
			t.Errorf("Time was wrong number of epoch seconds %d", node.At_Unix)
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

//
// Test that we can detect unknown states.
//
func TestNodeStatus(t *testing.T) {

	//
	// Test-cases
	//
	type TestCase struct {
		state string
		valid bool
	}

	//
	// Possible states, and whether they are valid
	//
	fail := []TestCase{
		{"changed", true},
		{"unchanged", true},
		{"failed", true},
		{"blah", false},
		{"forced", false},
		{"unknown", false}}

	//
	// For each test-case
	//
	for _, input := range fail {

		//
		// Build up YAML
		//
		var tmp string

		tmp = "---\n" +
			"host: foo.example.com\n" +
			"time: '2017-08-07T16:37:42.659245699+00:00'\n" +
			"status: " + input.state

		//
		// Parse it.
		//
		_, err := ParsePuppetReport([]byte(tmp))

		//
		// regexp for matching error-conditions
		//
		reg, _ := regexp.Compile("status")

		//
		// Do we expect this to pass/fail?
		//
		if input.valid {

			if reg.MatchString(err.Error()) {
				t.Errorf("Expected no error relating to 'status', but got one: %v", err)
			}
		} else {

			//
			// We expect this to fail.  Did it?
			//
			if !reg.MatchString(err.Error()) {
				t.Errorf("Expected an error relating to 'status', but didn't: %v", err)
			}
		}
	}
}

//
// Test importing a valid YAML file.
//
// TODO: Test bogus ones too.
//
func TestValidYaml(t *testing.T) {

	//
	// Read the YAML file.
	//
	tmpl, err := Asset("data/valid.yaml")
	if err != nil {
		t.Fatal("Failed to load YAML asset data/valid.yaml")
	}

	report, err := ParsePuppetReport(tmpl)

	if err != nil {
		t.Fatal("Failed to parse YAML file")
	}

	//
	// Test data from YAML
	//
	if report.Fqdn != "www.steve.org.uk" {
		t.Errorf("Incorrect hostname: %v", report.Fqdn)
	}
	if report.State != "unchanged" {
		t.Errorf("Incorrect state: %v", report.State)
	}
	if report.At != "2017-07-29 23:17:01" {
		t.Errorf("Incorrect at: %v", report.At)
	}
	if report.Failed != "0" {
		t.Errorf("Incorrect failed: %v", report.Failed)
	}
	if report.Changed != "0" {
		t.Errorf("Incorrect changed: %v", report.Changed)
	}
	if report.Skipped != "2" {
		t.Errorf("Incorrect skipped: %v", report.Skipped)
	}
}
