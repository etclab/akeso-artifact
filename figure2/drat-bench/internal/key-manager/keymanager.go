package keymanager

import (
	"crypto/ecdh"
	"crypto/rand"
	"drat/internal/drat"
	"drat/internal/mu"
	"fmt"
	"io"
)

func randomBytes(size int) ([]byte, error) {
	key := make([]byte, size)
	_, err := io.ReadFull(rand.Reader, key)
	if err != nil {
		return nil, fmt.Errorf("io.ReadFull failed: %v", err)
	}
	return key, nil
}

func (p *Pool) Init(numUsers int) {
	p.size = numUsers

	kmSk, err := drat.DHKeyGen()
	if err != nil {
		mu.Die("error: unable to generate key pair: %v", err)
	}

	for i := 0; i < numUsers; i++ {
		// initialize the group
		g := &Group{size: 2}
		g.buffer = make([]*drat.PDU, 1)

		// prepare keys
		{
			g.memberSks = make([]*ecdh.PrivateKey, g.size)
			g.memberSks[0] = kmSk

			key, err := drat.DHKeyGen()
			if err != nil {
				mu.Die("error: unable to generate key pair: %v", err)
			}
			// key for the second member
			g.memberSks[1] = key
		}

		// prepare initial shared secret between the group members
		{
			sharedSecret, err := randomBytes(drat.AESKeySize)
			if err != nil {
				mu.Die("error: unable to generate shared secret: %v", err)
			}
			g.sharedSecret = sharedSecret
		}

		// add the group to the pool
		p.groups = append(p.groups, g)
	}
}

type Pool struct {
	size    int
	groups  []*Group
	message []byte // the AES key that is to be shared with all members
}

type Group struct {
	size int
	// 32-byte shared secret that serves as the initial root key
	sharedSecret []byte
	// secret keys of all members
	// first member is always the Key Manager
	// Key Manager encrypts the message and sends it to the second member
	// Second member decrypts the message
	memberSks []*ecdh.PrivateKey
	buffer    []*drat.PDU // buffer for the message
}

func (g *Group) encryptMessage(message []byte) *drat.PDU {
	sharedSecret := g.sharedSecret

	// key manager is the sender
	senderPrivKey := g.memberSks[0]
	senderPubKey := senderPrivKey.PublicKey()

	// the other member is the receiver
	receiverPrivKey := g.memberSks[1]
	receiverPubKey := receiverPrivKey.PublicKey()

	dhOutput := drat.DH(senderPrivKey, receiverPubKey)
	_, chainKey := drat.KDF_RK(sharedSecret, dhOutput)

	header := &drat.Header{PubDH: senderPubKey}
	_, sendingMessageKey := drat.KDF_CK(chainKey)
	encMsg := drat.Encrypt(sendingMessageKey, message, header.Marshal())

	pdu := &drat.PDU{Header: header, Body: encMsg}

	return pdu
}

func (g *Group) decryptMessage(pdu *drat.PDU) []byte {
	sharedSecret := g.sharedSecret

	header := pdu.Header
	body := pdu.Body

	// key manager is the sender
	senderPrivKey := g.memberSks[0]
	senderPubKey := senderPrivKey.PublicKey()

	// the other member is the receiver
	receiverPrivKey := g.memberSks[1]

	dhOutput := drat.DH(receiverPrivKey, senderPubKey)
	_, chainKey := drat.KDF_RK(sharedSecret, dhOutput)
	_, receivingMessageKey := drat.KDF_CK(chainKey)

	msg, err := drat.Decrypt(receivingMessageKey, body, header.Marshal())
	if err != nil {
		mu.Die("error: unable to decrypt message %v", err)
	}

	// fmt.Printf("message at member: %x\n", string(msg[:]))
	return msg
}

func (p *Pool) RotateOnce() []byte {
	aesKey, err := randomBytes(drat.AESKeySize)
	if err != nil {
		mu.Die("error: unable to generate AES key: %v", err)
	}
	p.message = aesKey
	// fmt.Printf("message at key manager: %x\n", string(p.message[:]))

	var msg []byte
	for _, group := range p.groups {
		pdu := group.encryptMessage(p.message)
		msg = group.decryptMessage(pdu)
	}
	return msg
}
