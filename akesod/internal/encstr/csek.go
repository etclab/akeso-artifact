package encstr

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/storage"
	"github.com/etclab/akesod/internal/gcsx"
)

func CsekUpload(bkt *storage.BucketHandle, objectName string, fileData, key []byte) error {
	obj := bkt.Object(objectName)

	// set the Customer-Supplied Encryption Key (CSEK, which is a KEK)
	obj = obj.Key(key)

	/*// For an object that does not yet exist, set the DoesNotExist precondition.
	obj = obj.If(storage.Conditions{DoesNotExist: true})*/

	// Set the metadata fields
	metadata := map[string]string{
		"akeso_strategy": "csek",
	}

	err := gcsx.PutObjectWithMetadata(obj, fileData, metadata)
	if err != nil {
		log.Println("error: ", err.Error())
		return fmt.Errorf("gcsx.PutObject(%s): %w", objectName, err)
	}

	return nil
}

func CsekDownload(bkt *storage.BucketHandle, objectName string, key []byte) ([]byte, error) {
	var err error

	obj := bkt.Object(objectName)

	// set the Customer-Supplied Encryption Key (CSEK, which is a KEK)
	obj = obj.Key(key)

	// Get the object's attributes
	attrs, err := obj.Attrs(context.Background())
	if err != nil {
		log.Println("error: ", err.Error())
		return nil, fmt.Errorf("can't get attributes for object %s: %w", objectName, err)
	}

	// Set the generation-match condition
	obj = obj.If(storage.Conditions{GenerationMatch: attrs.Generation})

	// check akeso_strategy key-value entry
	strategy, ok := attrs.Metadata["akeso_strategy"]
	if !ok {
		log.Println("Error: ", err)
		return nil, fmt.Errorf("metadata for object %s does not have an akeso_strategy entry", objectName)
	}
	if strategy != "csek" {
		log.Println("Error: ", err)
		return nil, fmt.Errorf("expected metadata object %s to have akeso_strategy = csek, but got %s", objectName, strategy)
	}

	// Download the raw data
	data, err := gcsx.GetObject(obj)
	if err != nil {
		log.Println("Error: ", err)
		return nil, err
	}

	return data, nil
}

// rotateEncryptionKey encrypts an object with the newKey.
func RotateCSEKKey(bkt *storage.BucketHandle, objectName string, key, newKey []byte) error {
	objectUpdateStart := time.Now()

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Println("Error: ", err)
		return fmt.Errorf("storage.NewClient: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	obj := bkt.Object(objectName)

	// Set the generation-match condition
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		log.Println("Error: ", err)
		return fmt.Errorf("object.Attrs: %w", err)
	}
	obj = obj.If(storage.Conditions{GenerationMatch: attrs.Generation})

	_, err = obj.Key(newKey).CopierFrom(obj.Key(key)).Run(ctx)
	if err != nil {
		log.Println("Error: ", err)
		return fmt.Errorf("Key(%q).CopierFrom(%q).Run: %w", newKey, key, err)
	}
	duration := time.Since(objectUpdateStart)
	fmt.Printf("%s %v\n", objectName, duration)
	return nil
}
