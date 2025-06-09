package encstr

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strconv"
	"time"

	"cloud.google.com/go/storage"
	"github.com/etclab/aes256"
	"github.com/etclab/akesod/internal/gcsx"
	"github.com/etclab/nestedaes"
)

func AkesoUpload(bkt *storage.BucketHandle, objectName string, fileData, key, dek []byte) error {
	obj := bkt.Object(objectName)
	if dek == nil {
		dek = aes256.NewRandomKey()
	}

	iv := aes256.NewRandomIV()
	nonce := aes256.NewZeroNonce()
	payload := aes256.EncryptGCM(dek, nonce, fileData, nil)

	payload, tag, err := aes256.SplitCiphertextTag(payload)
	if err != nil {
		log.Println("Error: ", err.Error())
		return fmt.Errorf("error in nestedaes.Encrypt SplitCiphertextTag: %v", err)
	}

	// create the ciphertext header
	header, err := nestedaes.NewHeader(iv, tag, dek)
	if err != nil {
		log.Println("Error: ", err.Error())
		return fmt.Errorf("error in nestedaes.Encrypt Header: %v", err)
	}

	hData, err := header.Marshal(key)
	if err != nil {
		log.Println("Error: ", err.Error())
		return fmt.Errorf("error in nestedaes.Encrypt Header Marshalling: %v", err)
	}

	/*// For an object that does not yet exist, set the DoesNotExist precondition.
	obj = obj.If(storage.Conditions{DoesNotExist: true})*/

	metadata := map[string]string{
		"akeso_strategy": "akeso",
		"akeso_deks":     base64.StdEncoding.EncodeToString(hData),
		"updated_by":     "akesod",
		"akeso_iv":       base64.StdEncoding.EncodeToString(iv),
	}
	err = gcsx.PutObjectWithMetadata(obj, payload, metadata)
	if err != nil {
		log.Println("Error: ", err.Error())
		return fmt.Errorf("error in gcsx.PutObjectWithMetadata: %v", err)
	}
	return nil
}

func AkesoDownload(bkt *storage.BucketHandle, objectName string, key []byte) ([]byte, error) {
	var err error

	obj := bkt.Object(objectName)

	// Get the object's attributes
	attrs, err := obj.Attrs(context.Background())
	if err != nil {
		log.Println("Error: ", err.Error())
		return nil, fmt.Errorf("can't get attributes for object %s: %w", objectName, err)
	}

	// Set the generation-match condition
	obj = obj.If(storage.Conditions{GenerationMatch: attrs.Generation})

	// check and unpack metadata fields
	strategy, ok := attrs.Metadata["akeso_strategy"]
	if !ok {
		log.Printf("metadata for object %s does not have an akeso_strategy entry", objectName)
		return nil, fmt.Errorf("metadata for object %s does not have an akeso_strategy entry", objectName)
	}
	if strategy != "akeso" {
		log.Printf("expected metadata object %s to have akeso_strategy = akeso, but got %s", objectName, strategy)
		return nil, fmt.Errorf("expected metadata object %s to have akeso_strategy = akeso, but got %s", objectName, strategy)
	}

	akesoDEKsReceived, ok := attrs.Metadata["akeso_deks"]
	if !ok {
		log.Printf("object %s does not have an akeso_deks metadata field", objectName)
		return nil, fmt.Errorf("object %s does not have an akeso_deks metadata field", objectName)
	}
	akesoDEKsDecoded, err := base64.StdEncoding.DecodeString(akesoDEKsReceived)
	if err != nil {
		log.Println("Error: ", err.Error())
		return nil, fmt.Errorf("object %s has a malformed akeso_deks metadata field", objectName)
	}
	akesoHeader, err := nestedaes.UnmarshalHeader(key, akesoDEKsDecoded)
	if err != nil {
		log.Println("Error: ", err.Error())
		return nil, fmt.Errorf("error in unmarshalling akeso header for object %s: %w", objectName, err)
	}

	// Download the raw data
	data, err := gcsx.GetObject(obj)
	if err != nil {
		log.Println("Error: ", err.Error())
		return nil, err
	}

	iv := aes256.CopyIV(akesoHeader.BaseIV)
	aes256.AddIV(iv, len(akesoHeader.DEKs)-1) // fast-forward to largest IV

	i := len(akesoHeader.DEKs) - 1
	for i > 0 {
		dek := akesoHeader.DEKs[i]
		aes256.DecryptCTR(dek, iv, data)
		aes256.DecIV(iv)
		i--
	}

	dek := akesoHeader.DEKs[i]
	nonce := aes256.NewZeroNonce()
	data = append(data, akesoHeader.DataTag...)
	plaintext, err := aes256.DecryptGCM(dek, nonce, data, nil)
	if err != nil {
		log.Println("Error: ", err.Error())
		return nil, err
	}

	return plaintext, nil
}

func AkesoUpdate(bkt *storage.BucketHandle, objectName string, max_reencryptions int, old_key, new_key []byte, dek []byte, ctx context.Context) error {
	var err error
	if dek == nil {
		dek = aes256.NewRandomKey()
	}

	objectUpdateStart := time.Now()

	obj := bkt.Object(objectName)

	// Get the object's attributes
	attrs, err := obj.Attrs(context.Background())
	if err != nil {
		log.Println("error: ", err.Error())
		return fmt.Errorf("can't get attributes for object %s: %w", objectName, err)
	}

	// Set the generation-match condition
	obj = obj.If(storage.Conditions{GenerationMatch: attrs.Generation})

	// check and unpack metadata fields
	strategy, ok := attrs.Metadata["akeso_strategy"]
	if !ok {
		log.Println("Error: ", err)
		return fmt.Errorf("metadata for object %s does not have an akeso_strategy entry", objectName)
	}
	if strategy != "akeso" {
		log.Println("Error: ", err)
		return fmt.Errorf("expected metadata object %s to have akeso_strategy = akeso, but got %s", objectName, strategy)
	}

	akesoDEKsReceived, ok := attrs.Metadata["akeso_deks"]
	if !ok {
		log.Println("Error: ", err)
		return fmt.Errorf("object %s does not have an akeso_deks metadata field", objectName)
	}
	akesoDEKsDecoded, err := base64.StdEncoding.DecodeString(akesoDEKsReceived)
	if err != nil {
		log.Println("error: ", err.Error())
		return fmt.Errorf("object %s has a malformed akeso_deks metadata field", objectName)
	}

	akesoHeader, err := nestedaes.UnmarshalHeader(old_key, akesoDEKsDecoded)
	if err != nil {
		log.Println("error: ", err.Error())
		return fmt.Errorf("error in unmarshalling akeso header for object %s", objectName)
	}

	nonce := aes256.NewZeroNonce()

	akesoHeader.AddDEK(dek)

	if len(akesoHeader.DEKs) < max_reencryptions {
		// Only update the Header, so that Cloud Function does the actual update
		hData, err := akesoHeader.Marshal(new_key)
		if err != nil {
			log.Println("error: ", err.Error())
			return fmt.Errorf("error in nestedaes.Encrypt Header Marshalling: %w", err)
		}
		attrs.Metadata["akeso_deks"] = base64.StdEncoding.EncodeToString(hData)
		attrs.Metadata["updated_by"] = "akesod-metadata-updater"
		attrs.Metadata["ongoing_reencryption"] = "true"
		attrs.Metadata["times_updated"] = strconv.Itoa(len(akesoHeader.DEKs))

		err = gcsx.UpdateObjectMetadata(obj, attrs.Metadata)
		if err != nil {
			log.Println("error: ", err.Error())
			return fmt.Errorf("error in updating akeso header for object %s", objectName)
		}
	} else {
		decryptedReceivedData, err := AkesoDownload(bkt, objectName, old_key)
		if err != nil {
			log.Println("error: ", err.Error())
			return fmt.Errorf("error decrypting object %s: %w", objectName, err)
		}

		payload := aes256.EncryptGCM(dek, nonce, decryptedReceivedData, nil)
		payload, tag, err := aes256.SplitCiphertextTag(payload)
		if err != nil {
			log.Println("error: ", err.Error())
			return fmt.Errorf("error in nestedaes.Encrypt SplitCiphertextTag: %w", err)
		}
		iv := aes256.NewRandomIV()

		// create the ciphertext header
		header, err := nestedaes.NewHeader(iv, tag, dek)
		if err != nil {
			log.Println("error: ", err.Error())
			return fmt.Errorf("error in nestedaes.Encrypt Header: %w", err)
		}

		hData, err := header.Marshal(new_key)
		if err != nil {
			log.Println("error: ", err.Error())
			return fmt.Errorf("error in nestedaes.Encrypt Header Marshalling: %w", err)
		}

		metadata := map[string]string{
			"akeso_strategy": "akeso",
			"akeso_deks":     base64.StdEncoding.EncodeToString(hData),
			"updated_by":     "akesod",
			"akeso_iv":       base64.StdEncoding.EncodeToString(iv),
		}

		gcsx.PutObjectWithMetadata(obj, payload, metadata)
	}

	objectUpdateEnd := time.Now()
	duration := objectUpdateEnd.Sub(objectUpdateStart)
	fmt.Printf("%s took %v from %dns to %dns\n", objectName, duration, objectUpdateStart.UnixNano(), objectUpdateEnd.UnixNano())

	return err
}
