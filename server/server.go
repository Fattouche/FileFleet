package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"time"

	quic "github.com/lucas-clemente/quic-go"
)

type Peer struct {
	PrivIP   string
	PubIP    string
	Name     string
	Friend   string
	FileName string
	FilePath string
	FileSize int64
}

var peerMap map[string]*Peer
var connMap map[string]quic.Stream

func createPeer(length int, buff []byte, publicIP string) (*Peer, error) {
	peer := new(Peer)
	err := json.Unmarshal(buff[:length], &peer)
	if err != nil {
		fmt.Println("Error in createPeer: " + err.Error())
		return nil, err
	}
	peer.PubIP = publicIP
	peer.FilePath = ""
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

func sendToPeers(conn quic.Stream) {
	peer := new(Peer)
	buff := make([]byte, 1024)
	length, err := conn.Read(buff)
	if err != nil {
		fmt.Println("Error reading: ", err)
	}
	err = json.Unmarshal(buff[:length], &peer)
	if err != nil {
		fmt.Println("Error: ", err)
	}
	if _, ok := connMap[peer.Friend]; ok && connMap[peer.Friend] != nil {
		fmt.Println("Recieved both quic connections, copying")
		conn2 := connMap[peer.Friend]
		defer conn2.Close()
		defer conn.Close()
		if peer.FileName != "" {
			conn.Write([]byte("1"))
			_, err := io.Copy(conn2, conn)
			if err != nil {
				fmt.Println("Error: ", err)
			}
		} else {
			conn2.Write([]byte("1"))
			_, err := io.Copy(conn, conn2)
			if err != nil {
				fmt.Println("Error: ", err)
			}
		}
		fmt.Println("Finished copying!")
		delete(connMap, peer.Friend)
		delete(connMap, peer.Name)
	} else {
		connMap[peer.Name] = conn
	}
}

func waitTransfer() {
	server, err := quic.ListenAddr(":8080", generateTLSConfig(), nil)
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
	defer server.Close()
	for {
		connection, err := server.Accept()
		if err != nil {
			log.Println("Error: ", err)
			continue
		}
		stream, err := connection.AcceptStream()
		if err != nil {
			log.Println("Error: ", err)
			continue
		}
		go sendToPeers(stream)
	}
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
	go waitTransfer()

	buff := make([]byte, 1000)
	peerMap = make(map[string]*Peer)
	connMap = make(map[string]quic.Stream)
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
