package main

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/etclab/mu"
)

const usage = `Usage: gcs-utils [options] BUCKET_NAME

Utils to manage configs for a cloud storage.

positional arguments:
  BUCKET_NAME 
    A valid bucket name starting with 'gs://'

options:
  -notification-config
    A notification config includes the project id, topic id, event type, and 
	custom attributes if any. Overwrites the previous notification configs 
	with the same options.

  -topic-id TOPIC_ID
	Topic where the notification will be sent

  -project-id PROJECT_ID
    Project where the topic belongs

  -event-type EVENT_TYPE
	The type of event that triggers the notification. Possible values: [
		OBJECT_FINALIZE, OBJECT_METADATA_UPDATE, OBJECT_DELETE, OBJECT_ARCHIVE
	]

  -custom-attributes CUSTOM_ATTRIBUTES
	Custom attributes attached to the notification message. Key-value pairs 
	represented as key1=value1,key2=value2,...

  -help
    Display this usage statement and exit.
    
example:
  $ ./gcs-utils -notification-config -topic-id TOPIC_ID -project-id PROJECT_ID 
  		-event-type EVENT_TYPE -custom-attributes=CUSTOM_ATTRIBUTES BUCKET_NAME
`

type Options struct {
	// positional
	bucketName string

	// optional
	topicId             string
	projectId           string
	eventType           string
	customAttributes    string
	customAttributesMap map[string]string
	notificationConfig  bool
}

var allEventTypes = []string{
	storage.ObjectFinalizeEvent,
	storage.ObjectMetadataUpdateEvent,
	storage.ObjectDeleteEvent,
	storage.ObjectArchiveEvent,
}

func printUsage() {
	fmt.Fprintf(os.Stdout, "%s", usage)
}

func parseOptions() *Options {
	opts := Options{}

	flag.Usage = printUsage
	// general options
	flag.BoolVar(&opts.notificationConfig, "notification-config", false, "")
	flag.StringVar(&opts.topicId, "topic-id", "", "")
	flag.StringVar(&opts.projectId, "project-id", "", "")
	flag.StringVar(&opts.eventType, "event-type", "", "")
	flag.StringVar(&opts.customAttributes, "custom-attributes", "", "")

	flag.Parse()

	if flag.NArg() != 1 {
		mu.Fatalf("error: expected one positional argument but got %d", flag.NArg())
	}
	opts.bucketName = flag.Arg(0)

	if opts.notificationConfig && (opts.topicId == "" || opts.projectId == "" || opts.eventType == "") {
		mu.Fatalf("error: topic-id, project-id, and event-type required for adding notification config")
	}

	opts.customAttributesMap = make(map[string]string)
	if opts.customAttributes != "" {
		attrs := strings.Split(opts.customAttributes, ",")
		for _, v := range attrs {
			pair := strings.SplitN(v, "=", 2)
			opts.customAttributesMap[pair[0]] = pair[1]
		}
	}

	if opts.eventType != "" && !slices.Contains(allEventTypes, opts.eventType) {
		mu.Fatalf("error: invalid event type")
	}

	return &opts
}
