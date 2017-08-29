package main

import (
	"testing"
)

func TestDescriptions(t *testing.T) {

	type TestCase struct {
		Seconds int64
		Result  string
	}

	cases := []TestCase{{20, "20 seconds ago"}, {13, "13 seconds ago"},
		{64, "1 minute ago"},
		{300, "5 minutes ago"},
		{60 * 60 * 2.4, "2 hours ago"},
		{60 * 60 * 24, "24 hours ago"},
		{60 * 60 * 24 * 3, "3 days ago"},
	}

	for _, o := range cases {

		out := timeDescr(o.Seconds)

		if out != o.Result {
			t.Errorf("Expected '%s' received '%s' for %d", o.Result, out, o.Seconds)
		}
	}
}
