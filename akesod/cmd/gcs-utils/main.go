package main

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/storage"
	"github.com/etclab/akesod/internal/gcsx"
	"github.com/etclab/mu"
)

func main() {
	opts := parseOptions()
	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	if err != nil {
		mu.Fatalf("storage.NewClient failed: %v", err)
	}

	bucketName, _, err := gcsx.ParseUrl(opts.bucketName)
	if err != nil {
		mu.Fatalf("gcsx.ParseUrl failed: %v", err)
	}
	bucket := client.Bucket(bucketName)

	topicId := opts.topicId
	projectId := opts.projectId
	metadataUpdateEvent := opts.eventType
	customAttrs := opts.customAttributesMap
	customAttrs["time"] = time.Now().Format(time.RFC3339)

	if opts.notificationConfig {
		err = gcsx.RemoveNotification(ctx, bucket, topicId, projectId, metadataUpdateEvent)
		if err != nil {
			mu.Fatalf("gcsx.RemoveNotification failed: %v", err)
		}

		notif, err := gcsx.AddNotification(ctx, bucket, &storage.Notification{
			TopicID:          topicId,
			TopicProjectID:   projectId,
			EventTypes:       []string{metadataUpdateEvent},
			CustomAttributes: customAttrs,
			PayloadFormat:    storage.JSONPayload,
		})
		if err != nil {
			mu.Fatalf("gcsx.AddNotification failed: %v", err)
		}

		fmt.Printf("Notification: %+v\n", notif)
	}
}
