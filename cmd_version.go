//
// Show our version.
//

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/subcommands"
)

var (
	version string
)

type versionCmd struct{}

//
// Glue
//
func (*versionCmd) Name() string     { return "version" }
func (*versionCmd) Synopsis() string { return "Show our version." }
func (*versionCmd) Usage() string {
	return `version :
  Report upon our version, and exit.
`
}

//
// Flag setup
//
func (p *versionCmd) SetFlags(f *flag.FlagSet) {
}

//
// Entry-point.
//
func (p *versionCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	//
	// Show the version
	//
	fmt.Printf("%s\n", version)

	//
	// All done.
	//
	return subcommands.ExitSuccess
}
