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
// Get all the metrics
//
func getMetrics() map[string]string {

	// A map to store the names & values which should be sent.
	metrics := make(map[string]string)

	//
	// Get the nodes we know about.
	//
	NodeList, err := getIndexNodes()
	if err != nil {
		panic(err)
	}

	//
	// Create a map to hold state.
	//
	states := make(map[string]int)

	//
	// Each known-state will default to being empty.
	//
	states["changed"] = 0
	states["unchanged"] = 0
	states["failed"] = 0

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
		metric := fmt.Sprintf("%s.runtime", o.Fqdn)
		value := o.Runtime

		//
		// Store in our map.
		//
		metrics[metric] = value

		//
		// Keep track of counts.
		//
		states[o.State] += 1
	}

	//
	// Now record our states.
	//
	for i, o := range states {
		//
		// The name + value
		//
		metric := fmt.Sprintf("state.%s", i)
		value := fmt.Sprintf("%d", o)

		metrics[metric] = value
	}

	return metrics
}

//
//  Get and send the metrics
//
func SendMetrics(host string, port int, prefix string, nop bool) {

	// Get the metrics.
	metrics := getMetrics()

	// Create the helper.
	g, err := graphite.NewGraphite(host, port)

	//
	// If there was an error in the helper we're OK,
	// providing we are running in `-nop`-mode.
	//
	if (err != nil) && (nop == false) {
		panic(err)
	}

	//
	// For each one ..
	//
	for name, value := range metrics {

		//
		// Add the prefix.
		//
		name = fmt.Sprintf("%s.%s", prefix, name)

		//
		// Show/Send.
		//
		if nop {
			fmt.Printf("%s %s\n", name, value)
		} else {
			g.SimpleSend(name, value)
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
