package main

import (
	"os"

	"github.com/etclab/akesod/internal/aesx"
	"github.com/etclab/mu"
)

func main() {
	var err error

	opts := parseOptions()

	data, err := os.ReadFile(opts.inputFile)
	if err != nil {
		mu.Fatalf("error: %v", err)
	}

	key, err := aesx.ReadKeyFile(opts.keyFile)
	if err != nil {
		mu.Fatalf("error: %v", err)
	}

	nonce, err := aesx.ReadNonceFile(opts.nonceFile)
	if err != nil {
		mu.Fatalf("error: %v", err)
	}

	aead := aesx.NewGcm(key)

	if opts.encrypt {
		data = aead.Seal(data[:0], nonce, data, nil)
		err = os.WriteFile(opts.outputFile, data, 0664)
		if err != nil {
			mu.Fatalf("error: %v", err)
		}
	} else {
		data, err = aead.Open(data[:0], nonce, data, nil)
		if err != nil {
			mu.Fatalf("error: %v", err)
		}
		err = os.WriteFile(opts.outputFile, data, 0664)
		if err != nil {
			mu.Fatalf("error: %v", err)
		}
	}
}
