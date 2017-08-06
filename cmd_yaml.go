//
// Show a YAML file, interactively
//

package main

import (
	"context"
	"flag"
	"github.com/google/subcommands"
)

//
// The options set by our command-line flags.
//
type yamlCmd struct {
}

//
// Glue
//
func (*yamlCmd) Name() string     { return "yaml" }
func (*yamlCmd) Synopsis() string { return "Show a summary of a YAML report." }
func (*yamlCmd) Usage() string {
	return `yaml file1 file2 .. fileN:
  Show a summary of the specified YAML reports.
`
}

//
// Flag setup: NOP
//
func (p *yamlCmd) SetFlags(f *flag.FlagSet) {
}

//
// Entry-point.
//
func (p *yamlCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	//
	// Show each file.
	//
	for _, arg := range f.Args() {
		YamlDump(arg)
	}

	//
	// All done.
	//
	return subcommands.ExitSuccess
}
