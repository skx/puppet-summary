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
)

//
// Get all the metrics
//
func getMetrics() map[string]string {

	// A map to store the names & values which should be sent.
	metrics := make(map[string]string)

	// Get the node-states.
	data, err := getStates()
	if err != nil {
		panic(err)
	}

	// Now record the metrics we would send.
	for i := range data {
		//
		// The name + value
		//
		metric := fmt.Sprintf("state.%s", data[i].State)
		value := fmt.Sprintf("%d", data[i].Count)

		metrics[metric] = value
	}

	// And return them
	return metrics
}

//
//  SendMetrics submits the metrics discovered to the specified carbon
// server - unless `nop` is in-use, in which case they are dumped to
// STDOUT.
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
