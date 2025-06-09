package main

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/etclab/mu"
)

func publishMessage(projectID, topicID string, data []byte, attrs map[string]string) {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		mu.Fatalf("pubsub.NewClient: %w", err)
	}
	defer client.Close()

	topic := client.Topic(topicID)

	res := topic.Publish(ctx, &pubsub.Message{
		Data:       data,
		Attributes: attrs,
	})

	id, err := res.Get(ctx)
	if err != nil {
		mu.Fatalf("Failed to publish: %v", err)
	}

	fmt.Printf("Published message with msg ID: %v\n", id)
}

func main() {
	opts := parseOptions()

	msgAttrs := map[string]string{"messageType": opts.messageType, "messageFor": opts.messageFor}
	publishMessage(opts.projectId, opts.topicId, []byte(opts.message), msgAttrs)

}
