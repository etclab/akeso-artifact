package encstr

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/storage"
	"github.com/etclab/akesod/internal/aesx"
	"github.com/etclab/akesod/internal/gcsx"
)

func StrawmanUpload(bkt *storage.BucketHandle, objectName string, fileData, key []byte) error {
	// randomly generate a data nonce
	nonce := aesx.GenerateRandomNonce()

	// Encrypt the raw data
	data := aesx.GcmEncrypt(fileData, nil, key, nonce)
	ciphertext, tag, err := aesx.SplitCiphertextTag(data)
	if err != nil {
		log.Println("error: ", err.Error())
		return fmt.Errorf("aes256.SplitCiphertextTag failed for object %s: %w", objectName, err)
	}

	// Upload the encrypted data
	obj := bkt.Object(objectName)

	// Set the metadata fields
	metadata := map[string]string{
		"akeso_strategy":   "strawman",
		"akeso_data_nonce": base64.StdEncoding.EncodeToString(nonce),
		"akeso_data_tag":   base64.StdEncoding.EncodeToString(tag),
	}
	err = gcsx.PutObjectWithMetadata(obj, ciphertext, metadata)
	if err != nil {
		log.Println("error: ", err.Error())
		return fmt.Errorf("gcsx.PutObjectWithMetadata(%s): %w", objectName, err)
	}

	return nil
}

func StrawmanDownload(bkt *storage.BucketHandle, objectName string, key []byte) ([]byte, error) {
	var err error

	obj := bkt.Object(objectName)

	// Get the object's attributes
	attrs, err := obj.Attrs(context.Background())
	if err != nil {
		log.Println("error: ", err.Error())
		return nil, fmt.Errorf("can't get attributes for object %s: %w", objectName, err)
	}

	// Set the generation-match condition
	obj = obj.If(storage.Conditions{GenerationMatch: attrs.Generation})

	// Check akeso_strategy key-value entry
	strategy, ok := attrs.Metadata["akeso_strategy"]
	if !ok {
		log.Println("Error: ", err)
		return nil, fmt.Errorf("metadata for object %s does not have an akeso_strategy entry", objectName)
	}
	if strategy != "strawman" {
		log.Println("Error: ", err)
		return nil, fmt.Errorf("expected metadata object %s to have akeso_strategy = strawman, but got %s", objectName, strategy)
	}

	// Get cryptographic nonce from metadata
	nonceB64, ok := attrs.Metadata["akeso_data_nonce"]
	if !ok {
		log.Println("Error: ", err)
		return nil, fmt.Errorf("object %s does not have an akeso_data_nonce metadata field", objectName)
	}
	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		log.Println("Error: ", err)
		return nil, fmt.Errorf("object %s has a malformed akeso_data_nonce metadata field", objectName)
	}

	// Get AES-GCM tag from metadata
	tagB64, ok := attrs.Metadata["akeso_data_tag"]
	if !ok {
		log.Println("Error: ", err)
		return nil, fmt.Errorf("object %s does not have an akeso_data_tag metadata field", objectName)
	}
	tag, err := base64.StdEncoding.DecodeString(tagB64)
	if err != nil {
		log.Println("Error: ", err)
		return nil, fmt.Errorf("object %s has a malformed akeso_data_tag metadata field", objectName)
	}

	// Download the raw data
	data, err := gcsx.GetObject(obj)
	if err != nil {
		log.Println("Error: ", err)
		return nil, err
	}

	// Combine tag and data, decrypt it
	data = append(data, tag...)
	data, err = aesx.GcmDecrypt(data, nil, key, nonce)
	if err != nil {
		log.Println("Error: ", err)
		return nil, err
	}

	return data, nil
}

func StrawmanUpdate(bkt *storage.BucketHandle, objectName string, old_key, new_key []byte) error {
	objectUpdateStart := time.Now()

	data, err := StrawmanDownload(bkt, objectName, old_key)
	if err != nil {
		log.Println("Error: ", err)
		return fmt.Errorf("can't download using strawman for object %s: %w", objectName, err)
	}

	err = StrawmanUpload(bkt, objectName, data, new_key)
	if err != nil {
		log.Println("Error: ", err)
		return fmt.Errorf("can't upload using strawman for object %s: %w", objectName, err)
	}

	duration := time.Since(objectUpdateStart)
	fmt.Printf("%s %v\n", objectName, duration)
	return nil
}
