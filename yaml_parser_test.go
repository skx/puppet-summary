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
