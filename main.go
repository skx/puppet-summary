//
// Entry-point to the puppet-summary service.
//


package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {

	//
	// Parse the command-line arguments.
	//
	host := flag.String("host", "127.0.0.1", "The IP to listen upon")
	port := flag.Int("port", 3001, "The port to bind upon")
	days := flag.Int("days", 7, "The default number of days history to keep")
	db := flag.String("db-file", "foo.db", "The SQLite database to use")
	flag.Parse()

	//
	// Setup the database, by opening a handle, and creating it if
	// missing.
	//
	SetupDB(*db)

	//
	// Handle non-flag arguments
	//
	if len(flag.Args()) >= 1 {

		//
		// Get the sub-command
		//
		sc := flag.Args()[0]

		//
		// HTTP-Server
		//
		if sc == "serve" {
			cmd_serve(*host, *port)
			os.Exit(0)
		}

		//
		// History-pruner
		//
		if sc == "prune" {
			cmd_prune(*days)
			os.Exit(0)
		}
	}

	fmt.Printf("Usage %s [options] subcommand\n\n", os.Args[0])
	fmt.Printf("Subcommands include:\n")
	fmt.Printf("\tserve - Launch the HTTP-server\n")
	fmt.Printf("\tprune - Prune old reports\n")
	fmt.Printf("\tyaml  - Parse the given YAML report-file\n")

	fmt.Printf("\n\nExample usage:\n")
	fmt.Printf("\tpuppet-server -port 3321 -host 127.0.0.1 serve\n")
	fmt.Printf("\tpuppet-server -db-file ./data.sql -days 3 prune:\n")

}
