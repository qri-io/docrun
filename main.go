// Docrun is a utility to extract source code examples from markdown files, and execute those
// examples in order to verify that they work correctly.
package main

import (
	"flag"
	"fmt"
	"os"
)

func displayOptions() {
	fmt.Printf("options:\n")
	fmt.Printf("   --v    verbose logging\n")
	fmt.Printf("   --vv   very verbose logging\n")
	fmt.Printf("\n")
}

func displayCommands() {
	fmt.Printf("commands:\n")
	fmt.Printf("   run [filename]   execute docrun on one file\n")
	fmt.Printf("   report           run over all directories in manifest.txt\n")
	fmt.Printf("\n")
}

func main() {
	verbosePtr := flag.Bool("v", false, "verbose logging to show more info")
	veryVerbosePtr := flag.Bool("vv", false, "very verbose logging to show debug info")
	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Printf("Usage: docrun [command] [options]\n")
		fmt.Printf("\n")
		displayOptions()
		displayCommands()
		os.Exit(1)
	}

	command := flag.Args()[0]
	logLevel := 0
	if *verbosePtr {
		logLevel = 1
	}
	if *veryVerbosePtr {
		logLevel = 2
	}

	setLogLevel(logLevel)

	if command == "run" {
		filename := flag.Args()[1]
		docAnalyze(filename)
	} else if command == "report" {
		createReport()
	} else {
		fmt.Printf("Error, unknown command \"%s\"\n", command)
		fmt.Printf("\n")
		displayCommands()
		os.Exit(1)
	}
}
