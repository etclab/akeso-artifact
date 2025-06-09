package main

import (
	"flag"
	"fmt"
	"os"
)

type Options struct {
	// optional
	topicId     string
	projectId   string
	message     string
	messageType string
	messageFor  string
}

const usage = `Usage: trigger-key-update [options]

Util to trigger key update:
	- "update_key" message is sent to trigger a key update 
	- the message is sent to "KeyUpdate" topic by default; so all members receive it

Default options:
	- topic-id: KeyUpdate
	- project-id: wild-flame-123456
	- message: Update key
	- message-type: update_key
	- message-for: bob

examples: 
	$ ./trigger-key-update 

`

func printUsage() {
	fmt.Fprintf(os.Stdout, "%s", usage)
}

func parseOptions() *Options {
	opts := Options{}

	flag.Usage = printUsage

	flag.StringVar(&opts.topicId, "topic-id", "KeyUpdate", "")
	flag.StringVar(&opts.projectId, "project-id", "wild-flame-123456", "")
	flag.StringVar(&opts.message, "message", "Update key", "")
	flag.StringVar(&opts.messageType, "message-type", "update_key", "")
	flag.StringVar(&opts.messageFor, "message-for", "bob", "")

	flag.Parse()

	return &opts
}
