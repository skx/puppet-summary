//
// Prune history by removing old reports.
//


package main

import (
	"fmt"
)

//
//  Entry-point.
//
func cmd_prune(days int) {
	fmt.Printf("Pruning reports older than %d days\n", days)
}
