//
// Prune history by removing old reports.
//

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/subcommands"
)

//
// The options set by our command-line flags.
//
type pruneCmd struct {
	dbFile    string
	days      int
	unchanged bool
	prefix    string
	verbose   bool
}

//
// Run a prune
//
func runPrune(x pruneCmd) error {

	//
	// Removing unchanged reports?
	//
	if x.unchanged {
		if x.verbose {
			fmt.Printf("Pruning 'unchanged' reports from beneath %s\n", ReportPrefix)
		}
		return (pruneUnchanged(x.prefix, x.verbose))
	}

	//
	// Otherwise just removing reports older than the given
	// number of days.
	//
	if x.verbose {
		fmt.Printf("Pruning reports older than %d days from beneath %s\n", x.days, ReportPrefix)
	}

	err := pruneReports(x.prefix, x.days, x.verbose)
	return err
}

//
// Glue
//
func (*pruneCmd) Name() string     { return "prune" }
func (*pruneCmd) Synopsis() string { return "Prune/delete old reports." }
func (*pruneCmd) Usage() string {
	return `prune [options]:
  Remove old report-files from disk, and our database.
`
}

//
// Flag setup
//
func (p *pruneCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&p.verbose, "verbose", false, "Be verbose in reporting output")
	f.IntVar(&p.days, "days", 7, "Remove reports older than this many days.")
	f.BoolVar(&p.unchanged, "unchanged", false, "Remove reports from hosts which had no changes.")
	f.StringVar(&p.dbFile, "db-file", "ps.db", "The SQLite database to use.")
	f.StringVar(&p.prefix, "prefix", "./reports/", "The prefix to the local YAML hierarchy.")
}

//
// Entry-point.
//
func (p *pruneCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	//
	// Setup the database, by opening a handle, and creating it if
	// missing.
	//
	SetupDB(p.dbFile)

	//
	// Invoke the prune
	//
	err := runPrune(*p)

	if err == nil {
		return subcommands.ExitSuccess
	}

	fmt.Printf("Error pruning: %s\n", err.Error())
	return subcommands.ExitFailure
}
