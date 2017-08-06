//
// Submit metrics to a graphite host.
//

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/subcommands"
	"github.com/marpaia/graphite-golang"
	"strings"
)

//
//  Entry-point.
//
func SendMetrics(host string, port int, prefix string, nop bool) {

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
	for _, o := range NodeList {

		//
		// Escape dots in the hostnames
		//
		o.Fqdn = strings.Replace(o.Fqdn, ".", "_", -1)

		//
		// Build up the metrics.
		//
		metric := fmt.Sprintf("%s.%s.runtime", prefix, o.Fqdn)
		value := o.Runtime

		//
		// Send it.
		//
		if nop {
			fmt.Printf("%s %s\n", metric, value)
		} else {
			g.SimpleSend(metric, value)
		}

		//
		// Keep track of counts.
		//
		stats[o.State] += 1
	}

	//
	// Now output states.
	//
	for i, o := range stats {
		//
		// The name + value
		//
		metric := fmt.Sprintf("%s.state.%s", prefix, i)
		value := fmt.Sprintf("%d", o)

		if nop {
			fmt.Printf("%s %s\n", metric, value)
		} else {
			g.SimpleSend(metric, value)
		}
	}

}

//
// The options set by our command-line flags.
//
type metricsCmd struct {
	db_file string
	host    string
	port    int
	prefix  string
	nop     bool
}

//
// Glue
//
func (*metricsCmd) Name() string     { return "metrics" }
func (*metricsCmd) Synopsis() string { return "Submit metrics to a central carbon server." }
func (*metricsCmd) Usage() string {
	return `metrics [options]:
  Submit metrics to a central carbon server.
`
}

//
// Flag setup
//
func (p *metricsCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.db_file, "db-file", "ps.db", "The SQLite database to use.")
	f.StringVar(&p.host, "host", "localhost", "The carbon host to send metrics to.")
	f.IntVar(&p.port, "port", 2003, "The carbon port to use, when submitting metrics.")
	f.StringVar(&p.prefix, "prefix", "puppet", "The prefix to use when submitting metrics.")
	f.BoolVar(&p.nop, "nop", false, "Print metrics rather than submitting them.")
}

//
// Entry-point.
//
func (p *metricsCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	//
	// Setup the database, by opening a handle, and creating it if
	// missing.
	//
	SetupDB(p.db_file)

	//
	// Run metrics
	//
	SendMetrics(p.host, p.port, p.prefix, p.nop)

	//
	// All done.
	//
	return subcommands.ExitSuccess
}
