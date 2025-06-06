package drat

import (
	"crypto/ecdh"
)

type Header struct {
	PubDH *ecdh.PublicKey
}

func (h *Header) Marshal() []byte {
	return h.PubDH.Bytes()
}

func UnmarshalHeader(data []byte) (*Header, error) {
	curve := ecdh.X25519()
	pubKey, err := curve.NewPublicKey(data)
	if err != nil {
		return nil, err
	}
	return &Header{PubDH: pubKey}, nil
}
