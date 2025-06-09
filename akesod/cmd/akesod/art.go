package main

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"log"

	"encoding/base64"
	"encoding/json"

	"fmt"
	"os"
	"path/filepath"

	"github.com/etclab/art"
	"github.com/etclab/mu"
)

type KeyPairMessage struct {
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

func createKeyNames(basePath, format, keyType string) (string, string) {
	if basePath == "" {
		basePath = "keys/akesod"
	}

	pubPath := fmt.Sprintf("%s-%s-pub.%s", basePath, keyType, format)
	privPath := fmt.Sprintf("%s-%s.%s", basePath, keyType, format)

	return pubPath, privPath
}

func generateIKPair(pubPath, privPath string, encoding art.KeyEncoding, returnType string) ([]byte, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	err = art.WritePublicIKToFile(pubKey, pubPath, encoding)
	if err != nil {
		return nil, err
	}

	err = art.WritePrivateIKToFile(privKey, privPath, encoding)
	if err != nil {
		return nil, err
	}

	var ikPair KeyPairMessage
	switch returnType {
	case "both":
		ikPair = KeyPairMessage{
			PublicKey:  base64.StdEncoding.EncodeToString(pubKey),
			PrivateKey: base64.StdEncoding.EncodeToString(privKey),
		}
	case "private":
		ikPair = KeyPairMessage{
			PrivateKey: base64.StdEncoding.EncodeToString(privKey),
		}
	case "public":
		ikPair = KeyPairMessage{
			PublicKey: base64.StdEncoding.EncodeToString(pubKey),
		}
	}

	jsonData, err := json.Marshal(ikPair)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func generateEKPair(pubPath, privPath string, encoding art.KeyEncoding, returnType string) ([]byte, error) {
	curve := ecdh.X25519()
	privKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		mu.Fatalf("error: generating EK Pair: %v", err)
	}

	pubKey := privKey.PublicKey()
	err = art.WritePublicEKToFile(pubKey, pubPath, encoding)
	if err != nil {
		mu.Fatalf("error: writing public EK Pair to file: %v", err)
	}

	err = art.WritePrivateEKToFile(privKey, privPath, encoding)
	if err != nil {
		mu.Fatalf("error: writing private EK Pair to file: %v", err)
	}

	var ekPair KeyPairMessage
	switch returnType {
	case "both":
		ekPair = KeyPairMessage{
			PublicKey:  base64.StdEncoding.EncodeToString(pubKey.Bytes()),
			PrivateKey: base64.StdEncoding.EncodeToString(privKey.Bytes()),
		}
	case "private":
		ekPair = KeyPairMessage{
			PrivateKey: base64.StdEncoding.EncodeToString(privKey.Bytes()),
		}
	case "public":
		ekPair = KeyPairMessage{
			PublicKey: base64.StdEncoding.EncodeToString(pubKey.Bytes()),
		}
	}

	jsonData, err := json.Marshal(ekPair)
	if err != nil {
		mu.Fatalf("error: marshalling EK Pair: %v", err)
	}

	return jsonData, nil
}

func generateKeys(keytype string, outform, basePath string, encoding art.KeyEncoding) []byte {
	var err error
	var pubPath, privPath string

	var keyPairJSON []byte

	if keytype != "" {
		pubPath, privPath = createKeyNames(basePath, outform, keytype)
	}

	if keytype == "" {
		pubPath, privPath = createKeyNames(basePath, outform, "ik")
		_, err = generateIKPair(pubPath, privPath, encoding, "both")
		if err != nil {
			mu.Fatalf("failed to generate keypair: %v", err)
		}
		pubPath, privPath = createKeyNames(basePath, outform, "ek")
		_, err = generateEKPair(pubPath, privPath, encoding, "both")
		if err != nil {
			mu.Fatalf("failed to generate keypair: %v", err)
		}

	} else if keytype == "ek" {
		keyPairJSON, err = generateEKPair(pubPath, privPath, encoding, "private")
		if err != nil {
			mu.Fatalf("failed to generate keypair: %v", err)
		}

	} else if keytype == "ik" {
		keyPairJSON, err = generateIKPair(pubPath, privPath, encoding, "public")
		if err != nil {
			mu.Fatalf("failed to generate keypair: %v", err)
		}
	}

	if err != nil {
		mu.Fatalf("failed to generate keypair: %v", err)
	}

	return keyPairJSON
}

func setup_group(opts *Options) (setup_msg, setup_msg_sig []byte) {
	if opts.outDir == "" {
		opts.outDir = filepath.Base(opts.artConfigFile)
	}

	if opts.sigFile == "" {
		opts.sigFile = opts.msgFile + ".sig"
	}

	opts.msgFile = filepath.Join(opts.outDir, opts.msgFile)
	opts.sigFile = filepath.Join(opts.outDir, opts.sigFile)
	opts.treeStateFile = filepath.Join(opts.outDir, opts.treeStateFile)

	outDir := "./"
	err := os.MkdirAll(outDir, 0750)
	if err != nil {
		mu.Fatalf("error: can't create out-dir: %v", err)
	}

	state, setupMsg := art.SetupGroup(opts.artConfigFile, opts.initiator)
	log.Println("Setup Group completed.")

	setupMsg.Save(opts.msgFile)
	setupMsg.SaveSign(opts.sigFile, opts.msgFile, opts.privIKFile)
	log.Println("Setup Msg and Sig Saved.")
	state.Save(opts.treeStateFile)
	state.SaveStageKey(filepath.Join(opts.outDir, "stage-key.pem"))

	sig, err := art.SignFile(opts.basePath+"-ik.pem", opts.msgFile)
	if err != nil {
		mu.Fatalf("error signing message file: %v", err)
	}

	setupMsgJSON, _ := json.Marshal(setupMsg)
	return setupMsgJSON, sig
}
