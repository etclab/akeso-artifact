package drat

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/hkdf"

	"drat/internal/mu"
)

const (
	NonceSize      = 12
	RootKeySize    = 32
	ChainKeySize   = 32
	MessageKeySize = 32
	PublicKeySize  = 32
	AESKeySize     = 32
)

func LoadSecretKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) != RootKeySize {
		return nil, fmt.Errorf("expected a %d-byte secret; loaded %d-bytes", RootKeySize, len(data))
	}
	return data, nil
}

func DH(privKey *ecdh.PrivateKey, pubKey *ecdh.PublicKey) []byte {
	sharedSecret, err := privKey.ECDH(pubKey)
	if err != nil {
		mu.Panicf("ECDH() failed: %v", err)
	}
	return sharedSecret
}

func LoadPrivDH(path string) (*ecdh.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	curve := ecdh.X25519()
	return curve.NewPrivateKey(data)
}

func LoadPubDH(path string) (*ecdh.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	curve := ecdh.X25519()
	return curve.NewPublicKey(data)
}

func KDF_RK(rk, dhOut []byte) (newRK, ck []byte) {
	kdf := hkdf.New(sha256.New, rk, nil, dhOut)

	newRK = make([]byte, RootKeySize)
	_, err := io.ReadFull(kdf, newRK)
	if err != nil {
		mu.Panicf("io.ReadFull failed: %v", err)
	}

	ck = make([]byte, ChainKeySize)
	_, err = io.ReadFull(kdf, ck)
	if err != nil {
		mu.Panicf("io.ReadFull failed: %v", err)
	}

	return
}

func KDF_CK(ck []byte) (newCK, mk []byte) {
	kdf := hkdf.New(sha256.New, ck, nil, nil)

	newCK = make([]byte, ChainKeySize)
	_, err := io.ReadFull(kdf, newCK)
	if err != nil {
		mu.Panicf("io.ReadFull failed: %v", err)
	}

	mk = make([]byte, MessageKeySize)
	_, err = io.ReadFull(kdf, mk)
	if err != nil {
		mu.Panicf("io.ReadFull failed: %v", err)
	}

	return
}

func newAESGCM(key []byte) cipher.AEAD {
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		mu.Panicf("aes.NewCipher failed: %v", err)
	}

	aesgcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		mu.Panicf("cipher.NewGCM failed: %v", err)
	}

	return aesgcm
}

func Encrypt(mk, plaintext, assocData []byte) []byte {
	aesgcm := newAESGCM(mk)
	nonce := make([]byte, NonceSize) // a zero nonce
	return aesgcm.Seal(nil, nonce, plaintext, assocData)
}

func Decrypt(mk, ciphertext, assocData []byte) ([]byte, error) {
	aesgcm := newAESGCM(mk)
	nonce := make([]byte, NonceSize) // a zero nonce
	return aesgcm.Open(nil, nonce, ciphertext, assocData)
}

func DHKeyGen() (*ecdh.PrivateKey, error) {
	curve := ecdh.X25519()
	return curve.GenerateKey(rand.Reader)
}
