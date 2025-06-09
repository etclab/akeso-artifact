package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/etclab/mu"
)

const usage = `Usage: aesgcm [options] INPUT_FILE OUTPUT_FILE

AES-GCM encrypt or decrypt a file.

positional arguments:
  INPUT_FILE 
    The file to either encrypt or decrypt

  OUTPUT_FILE
    The output file.

options:
  -decrypt
    Decrypt the input file.

  -encrypt
    Encrypt the input file.  This is the default.

  -help
    Display this usage statement and exit.

  -key KEY_FILE
    The key file.  This file must have exactly 32 bytes.
    Default: key.bin

  -nonce NONCE_FILE
    The nonce file.  This file must have exactly 12 bytes.
    Default: nonce.bin
    
example:
  $ ./aesgcm -decrypt -nonce data/nonce.bin -key data/key.bin encrypted.dat decrypted.dat
`

type Options struct {
	// positional
	inputFile  string
	outputFile string

	// optional
	decrypt   bool
	encrypt   bool
	keyFile   string
	nonceFile string
}

func printUsage() {
	fmt.Fprintf(os.Stdout, "%s", usage)
}

func parseOptions() *Options {
	opts := Options{}

	flag.Usage = printUsage
	// general options
	flag.BoolVar(&opts.decrypt, "decrypt", false, "")
	flag.BoolVar(&opts.encrypt, "encrypt", false, "")
	flag.StringVar(&opts.keyFile, "key", "key.bin", "")
	flag.StringVar(&opts.nonceFile, "nonce", "nonce.bin", "")

	flag.Parse()

	if flag.NArg() != 2 {
		mu.Fatalf("error: expected two positional argument but got %d", flag.NArg())
	}
	opts.inputFile = flag.Arg(0)
	opts.outputFile = flag.Arg(1)

	if opts.encrypt && opts.decrypt {
		mu.Fatalf("error: can't specify both -encrypt and -decrypt")
	}

	if !opts.encrypt && !opts.decrypt {
		opts.encrypt = true
	}

	return &opts
}
