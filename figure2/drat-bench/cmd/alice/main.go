package main

import (
	"bufio"
	"crypto/ecdh"
	"flag"
	"fmt"
	"net"
	"os"

	"drat/internal/drat"
	"drat/internal/mu"
)

const usage = `Usage: alice [options] BOB_HOST

The "Alice" (initial sender) party of a double-ratchet communication.

positional arguments:
  BOB_HOST
    The a "host:port" string for Bob.  Host can be either an IP address or a
    hostname.
    
options:
  -secret SECRET_KEY
    The file that contains the 32-byte shared secret that serves as the
    initial root key.

    Default: secret.key

  -bob-pub BOB_PUBLIC_KEY
    A file with Bob's raw public X25519 key (that is, the file is the raw key
    bytes, rather than a DER or PEM-encoding of these bytes).

    Default: bob-pub.key

  -help
    Display this usage statement and exit.

examples:
$ ./alice -secret assets/secret.key -bob-pub assets/bob-pub.key 127.0.0.1:12345
`

type Options struct {
	// positional
	hostport string
	//options
	bobPubKeyFile string
	bobPubKey     *ecdh.PublicKey // derived
	secretKeyFile string
	secretKey     []byte // derived
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "%s", usage)
}

func parseOptions() *Options {
	var err error
	options := Options{}

	flag.Usage = printUsage
	flag.StringVar(&options.bobPubKeyFile, "bob-pub", "bob-pub.key", "")
	flag.StringVar(&options.secretKeyFile, "secret", "secret.key", "")
	flag.Parse()

	if flag.NArg() != 1 {
		mu.Die("expected one positional argument but got %d", flag.NArg())
	}

	options.hostport = flag.Arg(0)

	options.secretKey, err = drat.LoadSecretKey(options.secretKeyFile)
	if err != nil {
		mu.Die("error loading secret key from %s: %v", options.secretKeyFile, err)
	}

	options.bobPubKey, err = drat.LoadPubDH(options.bobPubKeyFile)
	if err != nil {
		mu.Die("error loading Bob's public key from %s: %v", options.bobPubKeyFile, err)
	}

	return &options
}

func createConn(hostport string) net.Conn {
	conn, err := net.Dial("tcp", hostport)
	if err != nil {
		mu.Die("error: %v", err)
	}
	return conn
}

func handleConn(conn net.Conn, options Options) {
	defer conn.Close()

	sending_ch := make(chan []byte)
	receiving_ch := make(chan drat.PDU) // what is a PDU?

	// goroutine to take user input
	go drat.ReadChannelFrom(*bufio.NewScanner(os.Stdin), sending_ch)

	// goroutine to accept message from bob
	go func() {
		defer close(receiving_ch)

		for {
			pdu, err := drat.ReceivePDU(conn)

			if err != nil {
				mu.Die("read from bob failed %v", err)
			} else {
				receiving_ch <- *pdu
			}
		}

	}()

	var header *drat.Header // what's a header?
	var receivingMessageKey []byte
	var sendingMessageKey []byte
	var chainKey []byte
	var bobKeyChanged bool

	bobPubKey := options.bobPubKey

	alicePrivKey, err := drat.DHKeyGen() // okay KeyGen here
	if err != nil {
		mu.Die("error: unable to generate key pair: %v", err)
	}

	dhOutput := drat.DH(alicePrivKey, bobPubKey)
	rootKey, chainKey := drat.KDF_RK(options.secretKey, dhOutput)

	for {
		select {
		case userInput, ok := <-sending_ch:
			if ok {
				if bobKeyChanged {
					alicePrivKey, err = drat.DHKeyGen()
					if err != nil {
						mu.Die("error: unable to generate key pair: %v", err)
					}

					dhOutput := drat.DH(alicePrivKey, bobPubKey)
					rootKey, chainKey = drat.KDF_RK(rootKey, dhOutput)

					bobKeyChanged = false
				}
				header = &drat.Header{PubDH: alicePrivKey.PublicKey()}

				chainKey, sendingMessageKey = drat.KDF_CK(chainKey)

				// fmt.Printf("sending msg key alice %x\n", sendingMessageKey)

				encMsg := drat.Encrypt(sendingMessageKey, userInput, header.Marshal())

				pdu := &drat.PDU{Header: header, Body: encMsg}
				err = pdu.Send(conn)

				if err != nil {
					mu.Die("error: socket write failed: %v", err)
				}
			} else {
				mu.Die("error: unable to send to bob")
			}
		case pdu, ok := <-receiving_ch:
			if ok {
				header := pdu.Header
				encBody := pdu.Body

				// fmt.Printf("%x\n", bobPubKey.Bytes())
				// fmt.Printf("%x\n", header.PubDH.Bytes())

				if !bobPubKey.Equal(header.PubDH) {
					// fmt.Println("===============bob sent a new public key")
					bobPubKey = header.PubDH
					dhOutput := drat.DH(alicePrivKey, bobPubKey)
					rootKey, chainKey = drat.KDF_RK(rootKey, dhOutput)
					bobKeyChanged = true
				}

				chainKey, receivingMessageKey = drat.KDF_CK(chainKey)

				// fmt.Printf("receiving msg key alice %x\n", receivingMessageKey)

				msg, err := drat.Decrypt(receivingMessageKey, encBody, header.Marshal())

				if err != nil {
					mu.Die("unable to decrypt message %v", err)
				}

				fmt.Printf("bob: %s\n", string(msg[:]))
			} else {
				mu.Die("error: unable to receive from bob")
			}
		}
	}
}

func main() {
	options := parseOptions()

	conn := createConn(options.hostport)
	handleConn(conn, *options)
}
