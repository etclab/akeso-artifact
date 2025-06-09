package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"github.com/etclab/aes256"
	"github.com/etclab/akesod/internal/aesx"
	"github.com/etclab/akesod/internal/encstr"
	"github.com/etclab/akesod/internal/gcsx"
	"github.com/etclab/art"
	"github.com/etclab/mu"
	"google.golang.org/api/iterator"
)

type UpdateKeyMessage struct {
	UpdateMsg    art.UpdateMessage `json:"updateMsg"`
	UpdateMsgMac []byte            `json:"updateMsgMac"`
}

// List all objects in a given bucket
func listObjects(ctx context.Context, bkt *storage.BucketHandle) []string {

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	query := &storage.Query{Prefix: ""}

	var names []string
	it := bkt.Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			mu.Fatalf("Listing Objects failed: %v", err)
		}
		names = append(names, attrs.Name)
	}
	return names
}

// Publishes to Pub/Sub Topic
func publish(ctx context.Context, topicID string, client *pubsub.Client, content []byte) {
	topic := client.Topic(topicID)
	result := topic.Publish(ctx, &pubsub.Message{
		Data: content,
		Attributes: map[string]string{
			"initiator": "akesod",
			"timedate":  time.Now().Format(time.RFC3339),
		},
	})

	id, err := result.Get(ctx)
	if err != nil {
		mu.Panicf(err.Error())
	}
	log.Printf("Published to %s; msg id: %v\n", topic, id)
}

// Handles subscription to Pub/Sub Topic (KeyUpdate specifically)
func handleKeyUpdateSubscription(ctx context.Context, updateTopic string, pubsubClient *pubsub.Client, opts *Options) error {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	client, err := storage.NewClient(ctx)
	if err != nil {
		mu.Fatalf("storage.NewClient failed: %v", err)
	}
	defer client.Close()

	bucket := opts.bucket
	bkt := client.Bucket(bucket)

	// Create Subscription if doesn't exist
	topic := pubsubClient.Topic(updateTopic)

	var sub *pubsub.Subscription
	sub, err = pubsubClient.CreateSubscription(ctx, updateTopic+"-akesod", pubsub.SubscriptionConfig{
		Topic:                 topic,
		AckDeadline:           20 * time.Second,
		EnableMessageOrdering: true,
	})
	if err != nil {
		sub = pubsubClient.Subscription(updateTopic + "-akesod")
	}

	// Channel to receive messages
	msgChan := make(chan *pubsub.Message)

	// Handle key update from received message
	go func() {
		for {
			err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
				log.Printf("Received Message in %s. Message ID: %s\n", updateTopic, msg.ID)

				msg.Ack()
				msgChan <- msg
			})
			if err != nil {
				mu.Panicf("error in subcription handling: %v", err)
			}
		}
	}()

	// Handle key updates
	for {
		select {
		case msg := <-msgChan:
			if msg.Attributes["messageType"] == "update_key" {
				continue
			}

			log.Printf("Key Updates triggered by Message ID: %s\n", msg.ID)

			treeStateFile := "keys/state.json"
			updateMsgFile := "keys/update_key.msg"
			updateMsgMacFile := "keys/update_key.msg.mac"

			// Process the received message to update_key and update_key_mac
			var updateMsg *UpdateKeyMessage
			json.Unmarshal(msg.Data, &updateMsg)

			updateMsg.UpdateMsg.Save(updateMsgFile)
			os.WriteFile(updateMsgMacFile, updateMsg.UpdateMsgMac, 0666)

			// update treeState using update_key
			treeStageKey := "keys/stage-key.pem"
			old_key, err := aesx.AESFromPEM(treeStageKey, opts.kdfSalt)
			if err != nil {
				mu.Fatalf("error: %v", err)
			}

			updatedTreeState := art.ProcessUpdateMessage(1, treeStateFile, updateMsgFile, updateMsgMacFile)
			os.Remove(treeStateFile)
			updatedTreeState.Save(treeStateFile)
			os.Remove(treeStageKey)
			updatedTreeState.SaveStageKey(treeStageKey)

			// generate new aes key from the updated stage key
			new_key, err := aesx.AESFromPEM(treeStageKey, opts.kdfSalt)
			if err != nil {
				mu.Fatalf("error: %v", err)
			}

			dek := aes256.NewRandomKey()

			files := listObjects(ctx, bkt)

			// Configure Notifications to trigger Cloud Function in case akeso strategy is being run
			if opts.strategy == "akeso" {
				err = gcsx.RemoveNotification(ctx, bkt, opts.metadataUpdateTopic, opts.project, "OBJECT_METADATA_UPDATE")
				if err != nil {
					log.Printf("gcsx.RemoveNotification failed: %v", err)
					return nil
				}
				keys := map[string]string{
					"new_dek": base64.StdEncoding.EncodeToString(dek),
				}

				_, err := gcsx.AddNotification(ctx, bkt, &storage.Notification{
					TopicID:          opts.metadataUpdateTopic,
					TopicProjectID:   opts.project,
					EventTypes:       []string{"OBJECT_METADATA_UPDATE"},
					CustomAttributes: keys,
					PayloadFormat:    storage.JSONPayload,
				})
				log.Printf("Notification Keys is \"%v\"", keys) // Testing Jul 5

				if err != nil {
					log.Printf("gcsx.AddNotification failed: %v", err)
					return nil
				}
			}

			var bucketUpdateStart time.Time

			var wg sync.WaitGroup
			sem := make(chan struct{}, opts.maxConcUpdates)
			errChan := make(chan error, len(files))

			for _, file := range files {
				// rencrypt and upload the list of objects using new aes key
				bucketUpdateStart = time.Now()

				wg.Add(1)
				sem <- struct{}{} // Acquire semaphore

				go func(file string) {
					defer wg.Done()
					defer func() { <-sem }() // Release semaphore

					switch opts.strategy {
					case "strawman":
						err = encstr.StrawmanUpdate(bkt, file, old_key, new_key)
					case "keywrap":
						err = encstr.KeyWrapUpdate(bkt, file, old_key, new_key)
					case "akeso":
						err = encstr.AkesoUpdate(bkt, file, opts.maxReencryptions, old_key, new_key, dek, ctx)
					case "csek":
						err = encstr.RotateCSEKKey(bkt, file, old_key, new_key)
					case "cmek":
						err = encstr.UpdateCMEKKey(bkt, file, old_key, new_key, ctx)
					}
					if err != nil {
						errChan <- err
						mu.Fatalf("error: %v", err)
					}
				}(file)

				go func() {
					wg.Wait()
					close(errChan)
				}()

				for err := range errChan {
					if err != nil {
						log.Printf("error: %v\n", err)
					}
				}

			}
			duration := time.Since(bucketUpdateStart)
			log.Printf("Duration for update/rotate keys of bucket by %s strategy is %v\n", opts.strategy, duration)
			continue

		case <-ctx.Done():
			log.Printf("\nExiting")
			return nil
		}

	}

}
