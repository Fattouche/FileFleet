package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	quic "github.com/lucas-clemente/quic-go"
)

type Peer struct {
	PrivIP   string
	PubIP    string
	Name     string
	Friend   string
	FileName string
	FileSize int64
}

var peerMap map[string]*Peer

func createPeer(length int, buff []byte, publicIP string) (*Peer, error) {
	peer := new(Peer)
	err := json.Unmarshal(buff[:length], &peer)
	if err != nil {
		fmt.Println("Error in createPeer: " + err.Error())
		return nil, err
	}
	peer.PubIP = publicIP
	file := strings.Split(peer.FileName, "/")
	peer.FileName = file[len(file)-1]
	peerMap[peer.Name] = peer
	return peer, nil
}

func checkPeer(peer *Peer, stream quic.Stream) {
	for {
		if _, ok := peerMap[peer.Friend]; ok && peerMap[peer.Friend] != nil {
			if !(peer.FileName == "" || peerMap[peer.Friend].FileName == "") {
				fmt.Println("Error: Both peers trying to send a file")
				stream.Write([]byte("2"))
				time.Sleep(time.Millisecond * 500)
				delete(peerMap, peer.Name)
				return
			}
			msgForPeer, err := json.Marshal(peerMap[peer.Friend])
			if err != nil {
				fmt.Println("Error marshalling in checkpeer: " + err.Error())
			}
			stream.Write(msgForPeer)

			time.Sleep(time.Millisecond * 500)
			delete(peerMap, peer.Name)
			return
		}
	}
}

//  generateTLSConfig is used to create a basic tls configuration for quic protocol.
func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{Certificates: []tls.Certificate{tlsCert}}
}

func main() {
	addr, err := net.ResolveUDPAddr("udp4", ":8080")
	server, err := net.ListenUDP("udp4", addr)
	fmt.Println("Listening on :8080")
	if err != nil {
		fmt.Println("Error: " + err.Error())
		server.Close()
		panic(err)
	}
	defer server.Close()

	buff := make([]byte, 1000)
	peerMap = make(map[string]*Peer)
	connection, _ := quic.Listen(server, generateTLSConfig(), nil)

	fmt.Println("Waiting for connections from peers")
	for {

		//Blocks waiting for a connection
		session, err := connection.Accept()
		fmt.Println("Got a connection from " + session.RemoteAddr().String())
		if err != nil {
			fmt.Println("Error: ", err)
		}
		defer session.Close(err)

		stream, err := session.AcceptStream()
		if err != nil {
			fmt.Println("Error: ", err)
		}
		defer stream.Close()

		len, err := stream.Read(buff)
		if err != nil {
			fmt.Println("Error: ", err)
		}
		peer, err := createPeer(len, buff, session.RemoteAddr().String())
		if err != nil {
			fmt.Println("Error parsing peer info: " + err.Error())
			continue
		} else {
			fmt.Println("Connecting " + peer.Name + " and " + peer.Friend)
		}
		go checkPeer(peer, stream)
	}
}