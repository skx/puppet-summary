//
// Entry-point to the puppet-summary service.
//

package main

import (
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"
)

//
// Setup our sub-commands and use them.
//
func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")
	subcommands.Register(&metricsCmd{}, "")
	subcommands.Register(&pruneCmd{}, "")
	subcommands.Register(&serveCmd{}, "")
	subcommands.Register(&versionCmd{}, "")
	subcommands.Register(&yamlCmd{}, "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
