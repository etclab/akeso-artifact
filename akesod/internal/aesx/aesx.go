package aesx

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/etclab/mu"
	"golang.org/x/crypto/hkdf"
)

const KeySize = 32
const NonceSize = 12
const TagSize = 16

func GenerateRandomKey() []byte {
	key := make([]byte, KeySize)
	_, err := rand.Read(key)
	if err != nil {
		mu.Panicf("aesx.GenerateRandomKey: rand.Read failed: %v", err)
	}
	return key
}

func GenerateRandomNonce() []byte {
	nonce := make([]byte, NonceSize)
	_, err := rand.Read(nonce)
	if err != nil {
		mu.Panicf("aesx.GenerateRandomNonce: rand.Read failed: %v", err)
	}
	return nonce
}

func ReadKeyFile(path string) ([]byte, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid key size: expected %d, got %d", KeySize, len(key))
	}

	return key, nil
}

func ReadNonceFile(path string) ([]byte, error) {
	nonce, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("invalid nonce size: expected %d, got %d", NonceSize, len(nonce))
	}

	return nonce, nil
}

func NewGcm(key []byte) cipher.AEAD {
	block, err := aes.NewCipher(key)
	if err != nil {
		mu.Panicf("aes.NewCipher: %v", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		mu.Panicf("cipher.NewGCM: %v", err)
	}

	return aead
}

func GcmEncrypt(data, additionalData, key, nonce []byte) []byte {
	aead := NewGcm(key)
	return aead.Seal(data[:0], nonce, data, additionalData)
}

func GcmDecrypt(data, additionalData, key, nonce []byte) ([]byte, error) {
	aead := NewGcm(key)
	return aead.Open(data[:0], nonce, data, additionalData)
}

func SplitCiphertextTag(ciphertext []byte) ([]byte, []byte, error) {
	if len(ciphertext) < TagSize {
		return nil, nil, fmt.Errorf("ciphertext (%d bytes) < AES GCM tag size (%d)", len(ciphertext), TagSize)
	}

	tag := ciphertext[len(ciphertext)-TagSize:]
	ciphertext = ciphertext[:len(ciphertext)-TagSize]
	return ciphertext, tag, nil
}

func AESFromPEM(pemFile string, salt []byte) ([]byte, error) {
	// Read the ED25519 private key from the PEM file
	privateKeyPEM, err := os.ReadFile(pemFile)
	if err != nil {
		mu.Panicf("Error reading private key file: %v\n", err)
	}

	// Parse the PEM block to extract the private key
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil || block.Type != "ED25519 PRIVATE KEY" {
		mu.Panicf("Failed to decode PEM block containing private key")
	}

	// Parse the private key
	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		mu.Panicf("Error parsing private key: %v\n", err)
	}

	// Extract the actual ED25519 private key
	ed25519PrivateKey, ok := privateKey.(ed25519.PrivateKey)
	if !ok {
		mu.Panicf("Invalid private key type")
	}

	// Use HKDF to derive an AES-256 key from the ED25519 private key
	info := []byte("aes-256-key from ed25519")
	hash := sha256.New
	aesKey := make([]byte, 32) // 32 bytes for AES-256

	kdf := hkdf.New(hash, ed25519PrivateKey.Seed(), salt, info)
	_, err = kdf.Read(aesKey)
	if err != nil {
		mu.Panicf("Error deriving AES key: %v\n", err)
	}

	fmt.Printf("AESKEY: %s\n", aesKey)
	return aesKey, nil

}
