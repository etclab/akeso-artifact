package main

import (
	"bufio"
	"crypto/ecdh"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"drat/internal/drat"
	"drat/internal/mu"
)

const usage = `Usage: bob [options] BOB_HOST

The "Bob" (initial-receiver) party of a double-ratchet communication.

positional arguments:
  BOB_HOST 
    The "[host]:port" string for Bob's address.  Host can be either an
    IP address, a host name (e.g., 'localhost') or omitted altogether.
    
options:
  -secret SECRET_KEY
    The file that contains the 32-byte shared secret that serves as the
    initial root key.

    Default: secret.key

  -pub PUBLIC_KEY
    A file with Bob's raw public X25519 key (that is, the file is the raw key
    bytes, rather than a DER or PEM-encoding of these bytes).

    Default: bob-pub.key

  -priv PRIVATE_KEY
    A file with Bob's raw private X25519 key.

    Default: bob-priv.key

  -help
    Display this usage statement and exit.

examples:
$ ./bob -secret assets/secret.key -pub assets/bob-pub.key -priv assets/bob-priv.key 127.0.0.1:12345
`

type Options struct {
	// positional
	hostport string
	// options
	pubKeyFile    string
	pubKey        *ecdh.PublicKey // derived
	privKeyFile   string
	privKey       *ecdh.PrivateKey // derived
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
	flag.StringVar(&options.pubKeyFile, "pub", "bob-pub.key", "")
	flag.StringVar(&options.privKeyFile, "priv", "bob-priv.key", "")
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

	options.pubKey, err = drat.LoadPubDH(options.pubKeyFile)
	if err != nil {
		mu.Die("error loading Bob's public key from %s: %v", options.pubKeyFile, err)
	}

	options.privKey, err = drat.LoadPrivDH(options.privKeyFile)
	if err != nil {
		mu.Die("error loading Bob's private key from %s: %v", options.privKeyFile, err)
	}

	return &options
}

func createServer(hostport string) net.Listener {
	server, err := net.Listen("tcp", hostport)
	if err != nil {
		log.Fatal(err)
	}
	return server
}

func handleConn(conn net.Conn, done chan bool, options Options) {
	defer conn.Close()
	defer close(done)

	sending_ch := make(chan []byte)
	receiving_ch := make(chan drat.PDU)

	// goroutine to take user input
	go drat.ReadChannelFrom(*bufio.NewScanner(os.Stdin), sending_ch)

	// goroutine to accept message from alice
	go func() {
		defer close(receiving_ch)

		for {
			pdu, err := drat.ReceivePDU(conn)

			if err != nil {
				mu.Die("read from alice %v", err)
			} else {
				receiving_ch <- *pdu
			}
		}
	}()

	var rootKey []byte
	var chainKey []byte
	var alicePubKey *ecdh.PublicKey
	var receivingMessageKey []byte
	var sendingMessageKey []byte
	var header *drat.Header
	var aliceKeyChanged bool = true
	var err error

	bobPrivKey := options.privKey

	for {
		select {
		case userInput, ok := <-sending_ch:
			if ok {
				if len(rootKey) == 0 {
					mu.Die("cannot send before alice")
				} else {
					if aliceKeyChanged {
						alicePubKey = header.PubDH
						bobPrivKey, err = drat.DHKeyGen()
						if err != nil {
							mu.Die("unable to generate key pair")
						}
						dhOutput := drat.DH(bobPrivKey, alicePubKey)
						rootKey, chainKey = drat.KDF_RK(rootKey, dhOutput)

						aliceKeyChanged = false
					}

					header = &drat.Header{PubDH: bobPrivKey.PublicKey()}
					chainKey, sendingMessageKey = drat.KDF_CK(chainKey)

					// fmt.Printf("sending msg key bob %x\n", sendingMessageKey)

					encMsg := drat.Encrypt(sendingMessageKey, userInput, header.Marshal())

					pdu := &drat.PDU{Header: header, Body: encMsg}
					err := pdu.Send(conn)

					if err != nil {
						mu.Die("error: socket write failed: %v", err)
					}
				}
			} else {
				mu.Die("error: unable to send to alice")
			}
		case pdu, ok := <-receiving_ch:
			if ok {
				header = pdu.Header
				encBody := pdu.Body

				if alicePubKey == nil {
					// when alice sends the first message
					alicePubKey = header.PubDH
					dhOutput := drat.DH(bobPrivKey, alicePubKey)
					rootKey, chainKey = drat.KDF_RK(options.secretKey, dhOutput)
				} else {
					// fmt.Printf("%x\n", alicePubKey.Bytes())
					// fmt.Printf("%x\n", header.PubDH.Bytes())

					// when alice sends a new public key
					if !alicePubKey.Equal(header.PubDH) {
						// fmt.Println("=================alice sent a new public key")
						alicePubKey = header.PubDH
						dhOutput := drat.DH(bobPrivKey, alicePubKey)
						rootKey, chainKey = drat.KDF_RK(rootKey, dhOutput)
						aliceKeyChanged = true
					}
				}

				chainKey, receivingMessageKey = drat.KDF_CK(chainKey)

				// fmt.Printf("receiving msg key bob %x\n", receivingMessageKey)

				msg, err := drat.Decrypt(receivingMessageKey, encBody, header.Marshal())

				if err != nil {
					mu.Die("error: unable to decrypt message %v", err)
				}

				fmt.Printf("alice: %s\n", string(msg[:]))
			} else {
				mu.Die("error: unable to receive from alice")
			}
		}
	}
}

func serveForever(server net.Listener, options Options) {
	done := make(chan bool)

	for {
		select {
		case _, ok := <-done:
			if !ok {
				return
			}
		default:
			conn, err := server.Accept()
			if err != nil {
				log.Print(err) // e.g., connection aborted
				continue
			}
			go handleConn(conn, done, options) // handle connections concurrently
		}
	}
}

func main() {
	options := parseOptions()

	server := createServer(options.hostport)
	defer server.Close()
	serveForever(server, *options)
}

// ./alice -secret assets/secret.key -bob-pub assets/bob-pub.key 127.0.0.1:12345

// ./bob -secret assets/secret.key -pub assets/bob-pub.key -priv assets/bob-priv.key 127.0.0.1:12345
