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
	db_file string
	days    int
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
	f.StringVar(&p.db_file, "db-file", "ps.db", "The SQLite database to use.")
	f.IntVar(&p.days, "days", 7, "Remove reports older than this man ydays.")
}

//
// Entry-point.
//
func (p *pruneCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	//
	// Setup the database, by opening a handle, and creating it if
	// missing.
	//
	SetupDB(p.db_file)

	//
	// Run the prune
	//
	fmt.Printf("Pruning reports older than %d days\n", p.days)
	pruneReports(p.days)

	//
	// All done.
	//
	return subcommands.ExitSuccess
}
