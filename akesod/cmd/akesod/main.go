package main

import (
	"context"
	"encoding/json"
	"log"

	"cloud.google.com/go/pubsub"
)

func main() {
	// Make logging output better.
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	opts := parseOptions()

	ctx := context.Background()

	setupTopic := opts.setupTopic
	updateTopic := opts.updateTopic
	projectID := opts.project
	pubsubClient, _ := pubsub.NewClient(ctx, projectID)
	// Message to publish in the pub/sub SetupGroup channel
	type SetupGroupMessage struct {
		EKeys       map[string][]byte `json:"EKeys"`
		IKeys       map[string][]byte `json:"IKeys"`
		InPubKey    []byte            `json:"InPubKey"`
		SetupMsg    []byte            `json:"SetupMsg"`
		SetupMsgSig []byte            `json:"SetupMsgSig"`
	}

	setupGroupMessage := &SetupGroupMessage{}

	if opts.setupRequired {

		if opts.basePath == "" {
			opts.basePath = "keys/akesod"
		}

		generateKeys("ek", opts.outform, opts.basePath, opts.encoding)
		initiator_pub_ik := generateKeys("ik", opts.outform, opts.basePath, opts.encoding)
		log.Println("Keys for inititator generated.")

		// TODO: Make the members key generation and publishing automated
		bob_ek := generateKeys("ek", opts.outform, "keys/bob", opts.encoding)
		bob_ik := generateKeys("ik", opts.outform, "keys/bob", opts.encoding)

		cici_ek := generateKeys("ek", opts.outform, "keys/cici", opts.encoding)
		cici_ik := generateKeys("ik", opts.outform, "keys/cici", opts.encoding)

		dave_ek := generateKeys("ek", opts.outform, "keys/dave", opts.encoding)
		dave_ik := generateKeys("ik", opts.outform, "keys/dave", opts.encoding)
		log.Println("Keys for members generated.")

		setupGroupMessage.InPubKey = initiator_pub_ik
		setupGroupMessage.EKeys = make(map[string][]byte)
		setupGroupMessage.EKeys["bob"] = bob_ek
		setupGroupMessage.EKeys["cici"] = cici_ek
		setupGroupMessage.EKeys["dave"] = dave_ek
		setupGroupMessage.IKeys = make(map[string][]byte)
		setupGroupMessage.IKeys["bob"] = bob_ik
		setupGroupMessage.IKeys["cici"] = cici_ik
		setupGroupMessage.IKeys["dave"] = dave_ik
		setup_msg, setup_msg_sig := setup_group(opts)
		setupGroupMessage.SetupMsg = setup_msg
		setupGroupMessage.SetupMsgSig = setup_msg_sig
		jsonData, _ := json.Marshal(setupGroupMessage)
		publish(ctx, setupTopic, pubsubClient, jsonData)
		log.Printf("Setup Group Message published to %s channel.\n", setupTopic)
	}

	go handleKeyUpdateSubscription(ctx, updateTopic, pubsubClient, opts)

	select {}

}
