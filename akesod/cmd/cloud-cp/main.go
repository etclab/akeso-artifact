package main

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/storage"
	"github.com/etclab/aes256"
	"github.com/etclab/akesod/internal/encstr"
	"github.com/etclab/mu"
)

func upload(bkt *storage.BucketHandle, fileName, objectName, strategy string, key []byte, ctx context.Context) error {
	fileData, err := os.ReadFile(fileName)
	if err != nil {
		log.Println("error: ", err.Error())
		return err
	}
	dek := aes256.NewRandomKey()

	switch strategy {
	case "strawman":
		err = encstr.StrawmanUpload(bkt, objectName, fileData, key)
	case "csek":
		err = encstr.CsekUpload(bkt, objectName, fileData, key)
	case "keywrap":
		err = encstr.KeyWrapUpload(bkt, objectName, fileData, key)
	case "akeso":
		err = encstr.AkesoUpload(bkt, objectName, fileData, key, dek)
	case "cmek":
		err = encstr.CmekUpload(bkt, objectName, fileData, key, ctx)
	default:
		mu.Panicf("unknown strategy: %s", strategy)
	}

	return err
}

func download(bkt *storage.BucketHandle, objectName, fileName, strategy string, key []byte, ctx context.Context) error {
	var data []byte
	var err error

	switch strategy {
	case "strawman":
		data, err = encstr.StrawmanDownload(bkt, objectName, key)
	case "csek":
		data, err = encstr.CsekDownload(bkt, objectName, key)
	case "keywrap":
		data, err = encstr.KeyWrapDownload(bkt, objectName, key)
	case "akeso":
		data, err = encstr.AkesoDownload(bkt, objectName, key)
	case "cmek":
		data, err = encstr.CmekDownload(bkt, objectName, key, ctx)
	default:
		mu.Panicf("unknown strategy: %s", strategy)
	}

	if err != nil {
		log.Println("error: ", err.Error())
		return err
	}

	return os.WriteFile(fileName, data, 0644)
}

func update(bkt *storage.BucketHandle, objectName, strategy string, maxReencryptions int, oldKey, newKey, dekOverride []byte, ctx context.Context) error {
	var err error

	switch strategy {
	case "strawman":
		err = encstr.StrawmanUpdate(bkt, objectName, oldKey, newKey)
	case "keywrap":
		err = encstr.KeyWrapUpdate(bkt, objectName, oldKey, newKey)
	case "akeso":
		err = encstr.AkesoUpdate(bkt, objectName, maxReencryptions, oldKey, newKey, dekOverride, ctx)
	case "csek":
		err = encstr.RotateCSEKKey(bkt, objectName, oldKey, newKey)
	case "cmek":
		err = encstr.UpdateCMEKKey(bkt, objectName, oldKey, newKey, ctx)
	default:
		mu.Panicf("unknown strategy: %s", strategy)
	}
	if err != nil {
		log.Println("error: ", err.Error())
		return err
	}

	return nil
}

func main() {
	// Setting Logger
	fileName := "logFile.log"

	// open log file
	logFile, err := os.OpenFile(fileName, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}
	defer logFile.Close()

	// set log out put
	log.SetOutput(logFile)

	// optional: log date-time, filename, and line number
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	opts := parseOptions()

	shouldRetry := func(err error) bool {
		switch {
		case err == nil:
			return false
		case err.Error() == "http2: client connection lost":
			return true
		default:
			return storage.ShouldRetry(err)
		}
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		mu.Fatalf("storage.NewClient failed: %v", err)
	}
	defer client.Close()
	client.SetRetry(storage.WithErrorFunc(shouldRetry))

	bkt := client.Bucket(opts.bucketName)

	if opts.updateKey != nil {
		err = update(bkt, opts.objectName, opts.strategy, opts.maxReencryptions, opts.key, opts.updateKey, opts.dekOverride, ctx)
	} else if opts.isUpload {
		err = upload(bkt, opts.fileName, opts.objectName, opts.strategy, opts.key, ctx)
	} else {
		err = download(bkt, opts.objectName, opts.fileName, opts.strategy, opts.key, ctx)
	}
	if err != nil {
		log.Println("error: ", err.Error())
		mu.Fatalf("error: %v", err)
	}
}
