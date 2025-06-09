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

// key is a KEK, and nonce is the nonce for the key
func KeyWrapUpload(bkt *storage.BucketHandle, objectName string, fileData, key []byte) error {
	// randomly generate a key nonece, data key, and data nonce
	keyNonce := aesx.GenerateRandomNonce()
	dataKey := aesx.GenerateRandomKey()
	dataNonce := aesx.GenerateRandomNonce()

	// Encrypt the raw data
	data := aesx.GcmEncrypt(fileData, nil, dataKey, dataNonce)
	ciphertext, dataTag, err := aesx.SplitCiphertextTag(data)
	if err != nil {
		log.Println("error: ", err.Error())
		return fmt.Errorf("aes256.SplitCiphertextTag failed for object %s: %w", objectName, err)
	}

	obj := bkt.Object(objectName)

	/*// For an object that does not yet exist, set the DoesNotExist precondition.
	obj = obj.If(storage.Conditions{DoesNotExist: true})*/

	// Encrypt the data key
	wrappedKey := aesx.GcmEncrypt(dataKey, nil, key, keyNonce)

	// Set the metadata fields
	metadata := map[string]string{
		"akeso_strategy":    "keywrap",
		"akeso_data_nonce":  base64.StdEncoding.EncodeToString(dataNonce),
		"akeso_data_tag":    base64.StdEncoding.EncodeToString(dataTag),
		"akeso_key_nonce":   base64.StdEncoding.EncodeToString(keyNonce),
		"akeso_wrapped_key": base64.StdEncoding.EncodeToString(wrappedKey),
	}

	err = gcsx.PutObjectWithMetadata(obj, ciphertext, metadata)
	if err != nil {
		log.Println("error: ", err.Error())
		return fmt.Errorf("gcsx.PutObjectWithMetadata(%s): %w", objectName, err)
	}

	return nil
}

func KeyWrapDownload(bkt *storage.BucketHandle, objectName string, key []byte) ([]byte, error) {
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

	// Check and unpack metadata fields
	dataKey, dataTag, dataNonce, _, err := unpackMetadata(attrs, objectName, key)
	if err != nil {
		log.Println("error: ", err.Error())
		return nil, fmt.Errorf("can't unpack metadata for object %s: %w", objectName, err)
	}

	// Download the raw data
	data, err := gcsx.GetObject(obj)
	if err != nil {
		log.Println("error: ", err.Error())
		return nil, err
	}

	// Combine tag and data, decrypt it
	data = append(data, dataTag...)
	data, err = aesx.GcmDecrypt(data, nil, dataKey, dataNonce)
	if err != nil {
		log.Println("error: ", err.Error())
		return nil, err
	}

	return data, nil
}

func unpackMetadata(attrs *storage.ObjectAttrs, objectName string, key []byte) ([]byte, []byte, []byte, []byte, error) {
	strategy, ok := attrs.Metadata["akeso_strategy"]
	if !ok {
		log.Println("metadata for object", objectName, "does not have an akeso_strategy entry")
		return nil, nil, nil, nil, fmt.Errorf("metadata for object %s does not have an akeso_strategy entry", objectName)
	}
	if strategy != "keywrap" {
		log.Println("expected metadata object", objectName, " to have akeso_strategy = keywrap, but got ", strategy)
		return nil, nil, nil, nil, fmt.Errorf("expected metadata object %s to have akeso_strategy = keywrap, but got %s", objectName, strategy)
	}

	dataNonceB64, ok := attrs.Metadata["akeso_data_nonce"]
	if !ok {
		log.Println(objectName, " does not have an akeso_data_nonce metadata field")
		return nil, nil, nil, nil, fmt.Errorf("object %s does not have an akeso_data_nonce metadata field", objectName)
	}
	dataNonce, err := base64.StdEncoding.DecodeString(dataNonceB64)
	if err != nil {
		log.Println("error: ", err.Error())
		return nil, nil, nil, nil, fmt.Errorf("object %s has a malformed akeso_data_nonce metadata field", objectName)
	}

	dataTagB64, ok := attrs.Metadata["akeso_data_tag"]
	if !ok {
		log.Println("Error: ", err)
		return nil, nil, nil, nil, fmt.Errorf("object %s does not have an akeso_data_tag metadata field", objectName)
	}
	dataTag, err := base64.StdEncoding.DecodeString(dataTagB64)
	if err != nil {
		log.Println("Error: ", err)
		return nil, nil, nil, nil, fmt.Errorf("object %s has a malformed akeso_data_tag metadata field", objectName)
	}

	keyNonceB64, ok := attrs.Metadata["akeso_key_nonce"]
	if !ok {
		log.Println("Error: ", err)
		return nil, nil, nil, nil, fmt.Errorf("object %s does not have an akeso_key_nonce metadata field", objectName)
	}
	keyNonce, err := base64.StdEncoding.DecodeString(keyNonceB64)
	if err != nil {
		log.Println("Error: ", err)
		return nil, nil, nil, nil, fmt.Errorf("object %s has a malformed akeso_key_nonce metadata field", objectName)
	}

	wrappedKeyB64, ok := attrs.Metadata["akeso_wrapped_key"]
	if !ok {
		log.Println("Error: ", err)
		return nil, nil, nil, nil, fmt.Errorf("object %s does not have an akeso_wrapped_key metadata field", objectName)
	}
	wrappedKey, err := base64.StdEncoding.DecodeString(wrappedKeyB64)
	if err != nil {
		log.Println("Error: ", err)
		return nil, nil, nil, nil, fmt.Errorf("object %s has a malformed akeso_wrapped_key metadata field", objectName)
	}

	dataKey, err := aesx.GcmDecrypt(wrappedKey, nil, key, keyNonce)
	if err != nil {
		log.Println("Error: ", err)
		return nil, nil, nil, nil, fmt.Errorf("can't unwrap key for object %s: %w", objectName, err)
	}

	return dataKey, dataTag, dataNonce, keyNonce, nil
}

func KeyWrapUpdate(bkt *storage.BucketHandle, objectName string, old_key, new_key []byte) error {
	objectUpdateStart := time.Now()

	obj := bkt.Object(objectName)

	// Get the object's attributes
	attrs, err := obj.Attrs(context.Background())
	if err != nil {
		log.Println("Error: ", err)
		return fmt.Errorf("can't get attributes for object %s: %w", objectName, err)
	}

	// Check and unpack metadata fields
	dataKey, _, _, keyNonce, err := unpackMetadata(attrs, objectName, old_key)
	if err != nil {
		log.Println("Error: ", err)
		return fmt.Errorf("can't unpack metadata for object %s: %w", objectName, err)
	}

	wrappedKey := aesx.GcmEncrypt(dataKey, nil, new_key, keyNonce)

	// Set the metadata fields
	metadata := attrs.Metadata
	metadata["akeso_wrapped_key"] = base64.StdEncoding.EncodeToString(wrappedKey)

	// Set the generation-match condition
	obj = obj.If(storage.Conditions{GenerationMatch: attrs.Generation})

	err = gcsx.UpdateObjectMetadata(obj, metadata)
	if err != nil {
		log.Println("Error: ", err)
		return fmt.Errorf("gcsx.UpdateObjectMetadata(%s): %w", objectName, err)
	}

	duration := time.Since(objectUpdateStart)
	fmt.Printf("%s %v\n", objectName, duration)
	return nil
}
