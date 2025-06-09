package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/etclab/akesod/internal/aesx"
	"github.com/etclab/akesod/internal/gcsx"
	"github.com/etclab/mu"
)

const usage = `Usage: cloud-cp [options] SRC DST

Copy files and objects.

positional arguments:
  SRC
    The source file.  This can either be a local file, or
    a cloud object.  If a cloud object, SRC must be a URL
    of the form gs://BUCKET/OBJECT.

  DST
    The destination file.  This can either be a local file, or
    a cloud object.  If a cloud object, DST must be a URL
    of the form gs://BUCKET/OBJECT.

  Note that one of SRC/DST must be a local file, and one must
  be a cloud object for upload and download. 

  For update/rotate key, only one cloud object can be specified.

options:
  -help
    Display this usage statement and exit.

  -strategy STRATEGY
    * strawman [default]
	* akeso
    * csek
	* cmek
    * keywrap

  -key KEY_FILE
    The key file.  This file must have exactly 32 bytes.
    Default: keys/key

-updateKey KEY_FILE
    The updated key file.  This file must have exactly 32 bytes.
    Default: keys/key

example:
$ ./cloud-cp -key keys/key data/alice.txt gs://wmsr-test-bucket/wonderland.txt
$ ./cloud-cp -key keys/key -strategy csek data/alice.txt gs://wmsr-test-bucket/wonderland.txt
$ ./cloud-cp -key keys/key -updateKey keys/key2.key -strategy akeso -maxReenc 4 gs://wmsr-test-bucket/wonderland.txt
`

type Options struct {
	// positional
	src string
	dst string

	// derived
	fileName   string
	bucketName string
	objectName string
	isUpload   bool

	// optional
	strategy         string
	keyFile          string
	updateKeyFile    string
	dekOverrideFile  string
	cmekKey          string
	cmekUpdateKey    string
	key              []byte // derived
	updateKey        []byte // derived
	dekOverride      []byte // derived
	maxReencryptions int
}

func printUsage() {
	fmt.Fprintf(os.Stdout, "%s", usage)
}

func parseOptions() *Options {
	var err error
	opts := Options{}

	flag.Usage = printUsage
	// general options
	flag.StringVar(&opts.strategy, "strategy", "strawman", "")
	flag.StringVar(&opts.keyFile, "key", "keys/key", "")
	flag.StringVar(&opts.updateKeyFile, "updateKey", "keys/key", "")
	flag.StringVar(&opts.dekOverrideFile, "dekOverride", "", "")
	flag.StringVar(&opts.cmekKey, "cmekKey", "projects/projectId/locations/global/keyRings/keyRingID/cryptoKeys/cryptoKeyID", "")
	flag.StringVar(&opts.cmekUpdateKey, "cmekUpdateKey", "projects/projectId/locations/global/keyRings/keyRingID/cryptoKeys/cryptoKeyID", "")
	flag.IntVar(&opts.maxReencryptions, "maxReenc", 2, "-maxReenc <NUM>")

	flag.Parse()

	if flag.NArg() != 1 && flag.NArg() != 2 {
		mu.Fatalf("error: expected two positional argument for upload/download and one for update but got %d", flag.NArg())
	}

	if flag.NArg() == 2 {

		opts.src = flag.Arg(0)
		opts.dst = flag.Arg(1)

		if strings.HasPrefix(opts.src, "gs://") && strings.HasPrefix(opts.dst, "gs://") {
			mu.Fatalf("error: SRC and DST can't both be GCS URLs")
		}

		if !strings.HasPrefix(opts.src, "gs://") && !strings.HasPrefix(opts.dst, "gs://") {
			mu.Fatalf("error: either SRC or DST must be a GCS URL")
		}

		if strings.HasPrefix(opts.src, "gs://") {
			opts.bucketName, opts.objectName, err = gcsx.ParseUrl(opts.src)
			if err != nil {
				mu.Fatalf("error: %v", err)
			}
			opts.fileName = opts.dst
		} else {
			opts.bucketName, opts.objectName, err = gcsx.ParseUrl(opts.dst)
			if err != nil {
				mu.Fatalf("error: %v", err)
			}
			opts.fileName = opts.src
			if opts.objectName == "" || strings.HasSuffix(opts.objectName, "/") {
				baseName := filepath.Base(opts.fileName)
				opts.objectName += baseName
			}
			opts.isUpload = true
		}
	} else {
		opts.src = flag.Arg(0)
		if !strings.HasPrefix(opts.src, "gs://") {
			mu.Fatalf("error: positional argument should be GCS URL")
		}
		opts.bucketName, opts.objectName, err = gcsx.ParseUrl(opts.src)
		if err != nil {
			mu.Fatalf("error: %v", err)
		}
	}

	if opts.strategy != "strawman" && opts.strategy != "csek" && opts.strategy != "keywrap" && opts.strategy != "akeso" && opts.strategy != "cmek" {
		mu.Fatalf("invalid -strategy.  Must be stramwan, csek, akeso, cmek or keywrap")
	}

	if opts.strategy == "cmek" && opts.cmekKey == "" {
		mu.Fatalf("error: -cmekKey not given")
	}
	// TODO: Doesn't check if cmek updateKey exists for now
	if opts.strategy != "cmek" {
		opts.key, err = aesx.ReadKeyFile(opts.keyFile)
		if err != nil {
			mu.Fatalf("error: %v", err)
		}
		if flag.NArg() == 1 {
			opts.updateKey, err = aesx.ReadKeyFile(opts.updateKeyFile)
			if err != nil {
				mu.Fatalf("error: %v", err)
			}
		}
	} else {
		opts.key = []byte(opts.cmekKey)
		if flag.NArg() == 1 {
			opts.updateKey = []byte(opts.cmekUpdateKey)
		}
	}

	if opts.strategy == "akeso" && opts.dekOverrideFile != "" {
		opts.dekOverride, _ = aesx.ReadKeyFile(opts.dekOverrideFile)
	}

	return &opts
}
