package encstr

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"cloud.google.com/go/storage"
)

// uploadWithKMSKey writes an object using Cloud KMS encryption.
func CmekUpload(bkt *storage.BucketHandle, objectName string, fileData, key []byte, ctx context.Context) error {
	keyName := string(key)

	obj := bkt.Object(objectName)

	/*// For an object that does not yet exist, set the DoesNotExist precondition.
	obj = obj.If(storage.Conditions{DoesNotExist: true})*/

	// Set the metadata fields
	metadata := map[string]string{
		"akeso_strategy": "cmek",
	}
	// Encrypt the object's contents.
	wc := obj.NewWriter(ctx)
	wc.KMSKeyName = keyName
	wc.Metadata = metadata
	if _, err := wc.Write(fileData); err != nil {
		log.Println("Error: ", err)
		return fmt.Errorf("Writer.Write: %w", err)
	}
	if err := wc.Close(); err != nil {
		log.Println("Error: ", err)
		return fmt.Errorf("Writer.Close: %w", err)
	}
	return nil
}

func CmekDownload(bkt *storage.BucketHandle, objectName string, key []byte, ctx context.Context) ([]byte, error) {
	obj := bkt.Object(objectName)

	// Get the object's metadata
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		log.Println("Error: ", err)
		return nil, fmt.Errorf("Object(%q).Attrs: %w", objectName, err)
	}

	if attrs.KMSKeyName != string(key) {
		log.Println("Error: ", err)
		return nil, fmt.Errorf("object was not encrypted with the expected KMS key")
	}

	// Set the generation-match condition
	obj = obj.If(storage.Conditions{GenerationMatch: attrs.Generation})

	// Open the object for reading
	reader, err := obj.NewReader(ctx)
	if err != nil {
		log.Println("Error: ", err)
		return nil, fmt.Errorf("Object(%q).NewReader: %w", objectName, err)
	}
	defer reader.Close()

	// Read the object's contents
	data, err := io.ReadAll(reader)
	if err != nil {
		log.Println("Error: ", err)
		return nil, fmt.Errorf("io.ReadAll: %w", err)
	}

	return data, nil
}

func UpdateCMEKKey(bkt *storage.BucketHandle, objectName string, oldKey, newKey []byte, ctx context.Context) error {
	objectUpdateStart := time.Now()

	obj := bkt.Object(objectName)

	// Open the object for reading
	reader, err := obj.NewReader(ctx)
	if err != nil {
		log.Println("Error: ", err)
		return fmt.Errorf("Object(%q).NewReader: %w", objectName, err)
	}
	defer reader.Close()

	// Get the object's metadata
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		log.Println("Error: ", err)
		return fmt.Errorf("Object(%q).Attrs: %w", objectName, err)
	}

	if attrs.KMSKeyName != string(oldKey) {
		log.Println("Error: ", err)
		return fmt.Errorf("object was not encrypted with the expected KMS key")
	}

	// Set the generation-match condition
	obj = obj.If(storage.Conditions{GenerationMatch: attrs.Generation})

	// Read the object's contents
	data, err := io.ReadAll(reader)
	if err != nil {
		log.Println("Error: ", err)
		return fmt.Errorf("io.ReadAll: %w", err)
	}

	// Set the metadata fields
	metadata := map[string]string{
		"akeso_strategy": "cmek",
	}
	// Encrypt the object's contents.
	wc := obj.NewWriter(ctx)
	wc.KMSKeyName = string(newKey)
	wc.Metadata = metadata
	if _, err := wc.Write(data); err != nil {
		log.Println("Error: ", err)
		return fmt.Errorf("Writer.Write: %w", err)
	}
	if err := wc.Close(); err != nil {
		log.Println("Error: ", err)
		return fmt.Errorf("Writer.Close: %w", err)
	}

	duration := time.Since(objectUpdateStart)
	fmt.Printf("%s %v\n", objectName, duration)

	return nil
}
