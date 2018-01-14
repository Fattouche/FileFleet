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
	"sync"
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

func checkPeer(peer *Peer, server *net.UDPConn) {
	addr, err := net.ResolveUDPAddr("udp4", peer.PubIP)
	if err != nil {
		fmt.Println("Error in checkPeer: " + err.Error())
	}
	for {
		if _, ok := peerMap[peer.Friend]; ok && peerMap[peer.Friend] != nil {
			if !(peer.FileName == "" || peerMap[peer.Friend].FileName == "") {
				fmt.Println("Error: Both peers trying to send a file")
				server.WriteToUDP([]byte("0"), addr)
				return
			}
			msgForPeer, err := json.Marshal(peerMap[peer.Friend])
			if err != nil {
				fmt.Println("Error marshalling in checkpeer: " + err.Error())
			}
			server.WriteToUDP([]byte("1"), addr)
			server.WriteToUDP(msgForPeer, addr)

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

func copyFile(length int, buff []byte, senderServer *net.UDPConn) {
	receiverAddr := string(buff[:length])
	var senderStream quic.Stream
	var receiverStream quic.Stream
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		session, err := quic.DialAddr(receiverAddr, &tls.Config{InsecureSkipVerify: true}, nil)
		if err != nil {
			fmt.Println("Error :" + err.Error())
		}
		receiverStream, err = session.OpenStreamSync()
		if err != nil {
			fmt.Println("Error :" + err.Error())
		}
	}()
	go func() {
		defer wg.Done()
		connection, err := quic.Listen(senderServer, generateTLSConfig(), nil)
		if err != nil {
			log.Println("Error: " + err.Error())
		}
		session, err := connection.Accept()
		if err != nil {
			log.Println("Error: " + err.Error())
		}

		senderStream, err = session.AcceptStream()
		if err != nil {
			log.Println("Error: " + err.Error())
		}
	}()
	wg.Wait()
	io.Copy(receiverStream, senderStream)
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

	fmt.Println("Waiting for connections from peers")
	for {
		//Blocks waiting for a connection
		len, addr, err := server.ReadFromUDP(buff)
		if len < 20 {
			copyFile(len, buff, server)
		}
		fmt.Println("Got a connection from " + addr.String())
		if err != nil {
			fmt.Println("Error reading from server: ", err)
			os.Exit(1)
		}
		peer, err := createPeer(len, buff, addr.String())
		if err != nil {
			fmt.Println("Error parsing peer info: " + err.Error())
			server.WriteToUDP([]byte("0"), addr)
			continue
		} else {
			fmt.Println("Connecting " + peer.Name + " and " + peer.Friend)
			server.WriteToUDP([]byte("1"), addr)
		}
		go checkPeer(peer, server)
	}
}
