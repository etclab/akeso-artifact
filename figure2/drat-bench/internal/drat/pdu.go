package drat

import (
	"encoding/gob"
	"fmt"
	"net"
)

type PDU struct {
	Header *Header
	Body   []byte
}

func (pdu *PDU) GobEncode() ([]byte, error) {
	data := pdu.Header.Marshal()
	data = append(data, pdu.Body...)
	return data, nil
}

func (pdu *PDU) GobDecode(data []byte) error {
	if len(data) < PublicKeySize {
		return fmt.Errorf("gob decode pdu: data len (%d) < public key size (%d)", len(data), PublicKeySize)
	}

	header, err := UnmarshalHeader(data[:PublicKeySize])
	if err != nil {
		return fmt.Errorf("gob decode pdu: failed to unmarshal header: %v", err)
	}
	pdu.Header = header
	pdu.Body = data[PublicKeySize:]
	return nil
}

func (pdu *PDU) Send(conn net.Conn) error {
	return gob.NewEncoder(conn).Encode(pdu)
}

func ReceivePDU(conn net.Conn) (*PDU, error) {
	var pdu PDU

	err := gob.NewDecoder(conn).Decode(&pdu)
	if err != nil {
		return nil, err
	}

	return &pdu, nil
}
