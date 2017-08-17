//
// Show a YAML file, interactively
//

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/subcommands"
	"io/ioutil"
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
// YamlDump parses the given file, and then dumps appropriate information
// from the give report.
//
func YamlDump(file string) {
	content, _ := ioutil.ReadFile(file)
	node, err := ParsePuppetReport(content)
	if err != nil {
		fmt.Printf("Failed to read %s, %v\n", file, err)
		return
	}

	fmt.Printf("Hostname: %s\n", node.Fqdn)
	fmt.Printf("Reported: %s\n", node.At)
	fmt.Printf("State   : %s\n", node.State)
	fmt.Printf("Runtime : %s\n", node.Runtime)

	fmt.Printf("\nResources\n")
	fmt.Printf("\tFailed : %s\n", node.Failed)
	fmt.Printf("\tChanged: %s\n", node.Changed)
	fmt.Printf("\tSkipped: %s\n", node.Skipped)
	fmt.Printf("\tTotal  : %s\n", node.Total)

	if node.Failed != "0" {
		fmt.Printf("\nFailed:\n")
		for i := range node.Resources_Failed {
			fmt.Printf("\t%s\n", node.Resources_Failed[i].Name)
			fmt.Printf("\t\t%s:%s\n", node.Resources_Failed[i].File, node.Resources_Failed[i].Line)
		}
	}

	if node.Changed != "0" {
		fmt.Printf("\nChanged:\n")
		for i := range node.Resources_Changed {
			fmt.Printf("\t%s\n", node.Resources_Changed[i].Name)
			fmt.Printf("\t\t%s:%s\n", node.Resources_Changed[i].File, node.Resources_Changed[i].Line)
		}
	}

	if node.Skipped != "0" {
		fmt.Printf("\nSkipped:\n")
		for i := range node.Resources_Skipped {
			fmt.Printf("\t%s\n", node.Resources_Skipped[i].Name)
			fmt.Printf("\t\t%s:%s\n", node.Resources_Skipped[i].File, node.Resources_Skipped[i].Line)
		}
	}

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
