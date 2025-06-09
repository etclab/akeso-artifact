package encobject

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/etclab/aes256"
	"github.com/googleapis/google-cloudevents-go/cloud/storagedata"
	"google.golang.org/protobuf/encoding/protojson"
)

type PubSubMessage struct {
	Data       []byte            `json:"data"`
	Attributes map[string]string `json:"attributes"`
}

type MessagePublishedData struct {
	Message PubSubMessage
}

func init() {
	functions.CloudEvent("EncryptObject", encryptObject)
}

func encryptObject(ctx context.Context, e event.Event) error {

	objectUpdateStart := time.Now()

	var msg MessagePublishedData
	if err := e.DataAs(&msg); err != nil {
		return fmt.Errorf("event.DataAs: %w", err)
	}

	var data storagedata.StorageObjectData
	if err := protojson.Unmarshal(msg.Message.Data, &data); err != nil {
		return fmt.Errorf("protojson.Unmarshal: %w", err)
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("storage.NewClient: %v", err)
	}
	defer client.Close()

	bucket := client.Bucket(data.GetBucket())

	metadata := data.GetMetadata()

	if metadata["updated_by"] == "akesod" {
		log.Println("File was reencrypted by akesod itself.")
		return nil
	}

	attrs := msg.Message.Attributes
	newDEK, err := base64.StdEncoding.DecodeString(attrs["new_dek"])
	if err != nil {
		log.Fatalf("Base64 Decoding of new DEK: %v", err)
	}

	base_iv, err := base64.StdEncoding.DecodeString(metadata["akeso_iv"])
	if err != nil {
		log.Fatalf("Base64 Decoding of current IV: %v", err)
	}

	objectName := data.GetName()
	object := bucket.Object(objectName)
	payload, err := GetObject(object)
	if err != nil {
		return fmt.Errorf("error in getting object %s", objectName)
	}

	objWriter := object.NewWriter(ctx)

	objWriter.ObjectAttrs.Metadata = metadata
	objWriter.ObjectAttrs.Metadata["updated_by"] = "cloud-function"
	objWriter.ObjectAttrs.Metadata["ongoing_reencryption"] = "false"

	iv := aes256.CopyIV(base_iv)
	times, _ := strconv.Atoi(metadata["times_updated"])
	aes256.AddIV(iv, times-1)
	if _, err = objWriter.Write(aes256.EncryptCTR(newDEK, iv, payload)); err != nil {
		log.Fatalf("Writer.Write: %v", err)
	}
	if err := objWriter.Close(); err != nil {
		log.Fatalf("Writer.Close: %v", err)
	}

	objectUpdateEnd := time.Now()
	duration := objectUpdateEnd.Sub(objectUpdateStart)
	log.Printf("[ENC] %s took %v from %dns to %dns\n", objectName, duration, objectUpdateStart.UnixNano(), objectUpdateEnd.UnixNano())

	return nil
}

func GetObject(obj *storage.ObjectHandle) ([]byte, error) {
	ctx := context.Background()
	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
