// Docrun is a utility to extract source code examples from markdown files, and execute those
// examples in order to verify that they work correctly.
package main

import (
	"flag"
	"fmt"
	"os"
)

// TODO(dlong): This first implementation is being used to act as a prototype in order to
// finalize the design of docrun. In the near future, add tests and documentation.

func main() {
	verbosePtr := flag.Bool("v", false, "verbose logging to show more info")
	veryVerbosePtr := flag.Bool("vv", false, "very verbose logging to show debug info")
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Printf("Usage: docrun [options] filename\n")
		os.Exit(1)
	}

	filename := flag.Args()[0]
	logLevel := 0
	if *verbosePtr {
		logLevel = 1
	}
	if *veryVerbosePtr {
		logLevel = 2
	}

	docAnalyze(filename, logLevel)
}
