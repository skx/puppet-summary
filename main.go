//
// Entry-point to the puppet-summary service.
//

package main

import (
	"context"
	"flag"
	"os"
	"fmt"
	"runtime/debug"

	"github.com/google/subcommands"
)

//
// Setup our sub-commands and use them.
//
func main() {
	defer func() {
		if r:= recover(); r != nil {
			fmt.Println("Panic at the disco: \n" + string(debug.Stack()))
		}
	}()

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
