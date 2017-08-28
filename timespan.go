//
// Utility function to create relative time-spans.
//

package main

import (
	"fmt"
	"strconv"
	"time"
)

//
// Given a string containing teh seconds past the epoch return
// a human-friendly description of how long ago that was.
//
func timeRelative(epoch string) string {

	//
	// Get now.
	//
	var now = time.Now().Unix()

	//
	// Convert the given string to an int
	//
	var unix, _ = strconv.ParseInt(epoch, 10, 64)

	//
	// The result
	//
	var result string

	//
	// How long ago was that, in seconds?
	//
	ago := now - unix

	//
	// Hacky code to divide that up into human-readable periods.
	//
	switch {
	case ago < 1:
		return "just now"
	case ago < 2:
		return "1 second ago"
	case ago < 60:
		return fmt.Sprintf("%d seconds ago", ago/60)
	case ago < 120:
		return "1 minute ago"
	case ago < 60*60:
		return fmt.Sprintf("%d minutes ago", ago/(60))
	case ago < 2*60*60:
		return "1 hour ago"
	case ago < 48*60*60:
		return fmt.Sprintf("%d hours ago", ago/(60*60))
	default:
		return fmt.Sprintf("%d days ago", ago/(60*60*24))
	}

	return result
}
