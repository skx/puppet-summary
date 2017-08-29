package main

import (
	"fmt"
	"testing"
	"time"
)

func TestDescriptions(t *testing.T) {

	type TestCase struct {
		Seconds int64
		Result  string
	}

	cases := []TestCase{{20, "20 seconds ago"}, {13, "13 seconds ago"},
		{-43, "43 seconds ago"},
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

//
// Test the wrapping method accepts sane values.
//
func TestString(t *testing.T) {

	//
	// Test "just now".
	//
	str := fmt.Sprintf("%d", time.Now().Unix())
	out := timeRelative(str)

	if out != "just now" {
		t.Errorf("Invalid time-value - got %s", out)
	}

	//
	// Test again with a negative time.
	// (Since "now + 1" will become negative when the test is run.)
	//
	str = fmt.Sprintf("%d", time.Now().Unix()+1)
	out = timeRelative(str)

	if out != "1 second ago" {
		t.Errorf("Invalid time-value - got %s", out)
	}

}
