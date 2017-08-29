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
// Describe the given number of seconds.  Negative values are treated
// identically to positive ones.
//
func timeDescr(seconds int64) string {

	//
	// We don't deal with future-time
	//
	if seconds < 0 {
		seconds *= -1
	}

	//
	// Divide up, from most recent to most distant.
	//
	switch {
	case seconds < 1:
		return "just now"
	case seconds < 2:
		return "1 second ago"
	case seconds < 60:
		return fmt.Sprintf("%d seconds ago", seconds)
	case seconds < 120:
		return "1 minute ago"
	case seconds < 60*60:
		return fmt.Sprintf("%d minutes ago", seconds/(60))
	case seconds < 2*60*60:
		return "1 hour ago"
	case seconds < 48*60*60:
		return fmt.Sprintf("%d hours ago", seconds/(60*60))
	default:
		return fmt.Sprintf("%d days ago", seconds/(60*60*24))
	}
}

//
// Given a string containing the seconds past the epoch return
// a human-friendly description of how long ago that was.
//
// (Using a string is weird.  I blame SQLite :)
//
func timeRelative(epoch string) string {

	//
	// Get the current time.
	//
	var now = time.Now().Unix()

	//
	// Convert the given string to an int
	//
	var unix, _ = strconv.ParseInt(epoch, 10, 64)

	//
	// How long ago was that, in an absolute number of seconds?
	//
	ago := now - unix
	if ago < 0 {
		ago *= -1
	}

	return (timeDescr(ago))
}
