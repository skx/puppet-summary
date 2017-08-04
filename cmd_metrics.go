//
// Submit metrics to a graphite host.
//

package main

import (
	"fmt"
	"strings"
	"github.com/marpaia/graphite-golang"
)

//
//  Entry-point.
//
func cmd_metrics(host string, port int, prefix string) {

	//
	// Get the nodes to show on our front-page
	//
	NodeList, err := getIndexNodes()
	if err != nil {
		panic(err)
	}


	//
	// Create a map to hold state.
	//
	stats := make(map[string]int)

	//
	// Create the helper
	//
	g, err := graphite.NewGraphite(host, port)

	//
	// Sum up the number of nodes in each state.
	//
	for _,o := range NodeList {

		//
		// Escape dots in the hostnames
		//
		o.Fqdn =  strings.Replace(o.Fqdn, ".", "_", -1)

		//
		// Build up the metrics.
		//
		metric := fmt.Sprintf("%s.%s.runtime", prefix, o.Fqdn)
		value  := o.Runtime

		//
		// Send it.
		//
		g.SimpleSend(metric,value)

		//
		// Keep track of counts.
		//
		stats[o.State] += 1
	}

	//
	// Now output states.
	//
	for i,o := range stats {
		//
		// The name + value
		//
		metric := fmt.Sprintf("%s.state.%s", prefix,i)
		value  := fmt.Sprintf("%d", o )

		g.SimpleSend(metric,value)
	}

}
