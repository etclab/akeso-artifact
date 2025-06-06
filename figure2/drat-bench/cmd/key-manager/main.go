package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	keymanager "drat/internal/key-manager"
	"drat/internal/mu"
)

const usage = `Usage: key-manager NUM

The key manager (KM) holds the master secret. Key Manager creates a two-person 
group with all the users, and sends the encrypted master secret using the double
ratchet algorithm

positional arguments:
  NUM
	Number of users to share the master secret with.
    
options:

  -help
    Display this usage statement and exit.

examples:
$ ./key-manager 2
`

type Options struct {
	// positional
	numUsers int
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "%s", usage)
}

func parseOptions() *Options {
	var err error
	options := Options{}

	flag.Usage = printUsage
	flag.Parse()

	if flag.NArg() != 1 {
		mu.Die("expected one positional argument but got %d", flag.NArg())
	}

	options.numUsers, err = strconv.Atoi(flag.Arg(0))
	if err != nil {
		mu.Die("error reading no of users: %v", err)
	}
	if options.numUsers < 1 {
		mu.Die("error: number of users must be at least 1")
	}

	return &options
}

func run(options *Options) {
	pool := &keymanager.Pool{}
	pool.Init(options.numUsers)
	pool.RotateOnce()
}

func main() {
	options := parseOptions()

	run(options)
}
