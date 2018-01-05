package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	quic "github.com/lucas-clemente/quic-go"
)

// Peer used to keep track of peer information.
type Peer struct {
	PrivIP   string
	PubIP    string
	Name     string
	Friend   string
	FileName string
	FileSize int64
}

var friend Peer
var myPeerInfo *Peer

// BUFFERSIZE used to read from file.
const BUFFERSIZE = 48000

// CentServerAddr used to communicate between peer and rendevouz server.
const CentServerAddr = "18.221.47.86:8080"

// holePunch punches a hole through users NATs if they exist in different networks.
func holePunch(server *net.UDPConn, addr *net.UDPAddr) error {
	connected := false
	go func() {
		for connected != true {
			server.WriteToUDP([]byte("1"), addr)
			time.Sleep(10 * time.Millisecond)
		}
	}()
	buff := make([]byte, 100)
	server.SetReadDeadline(time.Now().Add(time.Second * 1))
	for {
		_, recvAddr, err := server.ReadFromUDP(buff)
		if err != nil {
			connected = true
			return err
		}
		if recvAddr.String() == addr.String() {
			connected = true
			time.Sleep(time.Millisecond * 500)
			return nil
		}
	}
}

// sendFile sends a file from the server to the addr using Google's quic protocol on top of UDP.
func sendFile(server net.PacketConn, file *os.File, addr string) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	notifyFrontEnd("TRYING TO DIAL")
	config := new(quic.Config)
	config.HandshakeTimeout = 5
	session, err := quic.Dial(server, udpAddr, addr, &tls.Config{InsecureSkipVerify: true}, config)
	if err != nil {
		log.Println("Error: ", err)
		notifyFrontEnd("Error " + err.Error())
	}
	notifyFrontEnd("Dialed! " + err.Error())
	defer session.Close(err)
	stream, err := session.OpenStreamSync()
	defer stream.Close()

	log.Println("transferring!")
	notifyFrontEnd("Connected to " + friend.Name + ", starting transfer!")
	start := time.Now()

	io.Copy(stream, file)

	notifier := fmt.Sprintf("Finished transfer in %f seconds!", time.Since(start).Seconds())
	log.Println(notifier)
	notifyFrontEnd(notifier)
}

// receiveFile recieves a file from whoever establishes a quic connection with the udp server.
func receiveFile(server net.PacketConn, addr string) {
	newFile, err := os.Create(friend.FileName)
	if err != nil {
		notifyFrontEnd("Couldn't create " + friend.FileName)
		log.Println("Error: " + err.Error())
		return
	}
	defer newFile.Close()
	config := new(quic.Config)

	//Max flow control windows
	config.MaxReceiveStreamFlowControlWindow = 0
	config.MaxReceiveConnectionFlowControlWindow = 0
	config.HandshakeTimeout = 10
	connection, err := quic.Listen(server, generateTLSConfig(), config)
	if err != nil {
		log.Println("Error: " + err.Error())
		connection.Close()
		notifyFrontEnd("Couldn't establish a connection, please try again3!")
		return
	}
	defer connection.Close()
	notifyFrontEnd("WAITING FOR CONNECTION")
	session, err := connection.Accept()
	notifyFrontEnd("Got a connection!")
	if err != nil {
		notifyFrontEnd("Couldn't establish a connection, please try again4!")
		log.Println("Error: " + err.Error())
		return
	}
	defer session.Close(err)

	stream, err := session.AcceptStream()
	if err != nil {
		notifyFrontEnd("Couldn't establish a connection, please try again5!")
		log.Println("Error: " + err.Error())
		return
	}
	defer stream.Close()

	log.Println("transferring!")
	notifyFrontEnd("Connected to " + friend.Name + ", starting transfer!")
	start := time.Now()

	io.Copy(newFile, stream)

	notifier := fmt.Sprintf("Finished transfer in %f seconds!", time.Since(start).Seconds())
	log.Println(notifier)
	notifyFrontEnd(notifier)
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

// transferFile is the catalyst for setting up quic connections and initiating holepunching.
func transferFile(server *net.UDPConn) error {
	notifyFrontEnd("TRANSFER")
	var file *os.File
	var err error
	if myPeerInfo.FileName != "" {
		file, err = os.Open(myPeerInfo.FileName)
		if err != nil {
			return err
		}
	}
	addr, _ := net.ResolveUDPAddr("udp", friend.PubIP)
	laddr, _ := net.ResolveUDPAddr("udp", myPeerInfo.PrivIP)
	public := true
	err = holePunch(server, addr)
	if err != nil {
		server.Close()
		time.Sleep(time.Millisecond * 500)
		server, _ = net.ListenUDP("udp", laddr)
		public = false
	}
	notifyFrontEnd("STARTING TRANSFER")
	//If holepunching failed we know there is no peer in our network
	if myPeerInfo.FileName != "" {
		if public {
			notifyFrontEnd("PUBLIC")
			sendFile(server, file, friend.PubIP)
		} else {
			notifyFrontEnd("PRIVATE")
			sendFile(server, file, friend.PrivIP)
		}
	} else {
		receiveFile(server, myPeerInfo.PrivIP)
	}
	return nil
}

// getPeerInfo communicates with the centralized server to exchange information between peers.
func getPeerInfo(server *net.UDPConn) error {
	buff, err := json.Marshal(myPeerInfo)
	if err != nil {
		log.Println("Error:" + err.Error())
		return err
	}
	centUDPAddr, err := net.ResolveUDPAddr("udp", CentServerAddr)
	if err != nil {
		log.Println("Error:" + err.Error())
		return err
	}
	session, err := quic.Dial(server, centUDPAddr, CentServerAddr, &tls.Config{InsecureSkipVerify: true}, nil)
	if err != nil {
		log.Println("Error:" + err.Error())
		return err
	}
	defer session.Close(err)
	stream, err := session.OpenStreamSync()
	if err != nil {
		log.Println("Error:" + err.Error())
		return err
	}
	defer stream.Close()
	stream.Write(buff)
	recvBuff := make([]byte, BUFFERSIZE)
	len, _ := stream.Read(recvBuff)

	if string(recvBuff[:len]) == "2" {
		return errors.New("Both trying to send a file, please try again")
	}
	log.Println("Recieved peer information from server: " + string(recvBuff[:len]))
	err = json.Unmarshal(recvBuff[:len], &friend)
	if err != nil {
		log.Println("Error:" + err.Error())
		return err
	}

	return nil
}

// externalIP searches through the machines interfaces to collect its private IP within the subnet.
func externalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("Not connected to network")
}

func initTransfer(peer1, peer2, fileName string) {
	myPeerInfo = new(Peer)
	myPeerInfo.Name = strings.ToLower(peer1)
	myPeerInfo.Friend = strings.ToLower(peer2)
	myPeerInfo.FileName = fileName

	if myPeerInfo.FileName != "" {
		transferFile, err := os.Open(myPeerInfo.FileName)
		if err != nil {
			log.Println("Error: " + err.Error())
			transferFile.Close()
			notifyFrontEnd("Couldn't open " + myPeerInfo.FileName)
			return
		}
		fileInfo, _ := transferFile.Stat()
		myPeerInfo.FileSize = fileInfo.Size()
	}

	machineIP, err := externalIP()
	if err != nil {
		notifyFrontEnd("Machine might not connected to a network, couldn't find it's IP!")
		log.Println("Error getting machine ip: " + err.Error())
		return
	}

	addr, err := net.ResolveUDPAddr("udp", machineIP+":0")
	server, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Println("Error: " + err.Error())
		server.Close()
		notifyFrontEnd("Couldn't establish a connection, please try again1!")
		return
	}
	log.Println("Listening on :" + server.LocalAddr().String())
	defer server.Close()
	myPeerInfo.PrivIP = server.LocalAddr().String()
	err = getPeerInfo(server)
	if err != nil {
		log.Println("Error :" + err.Error())
		notifyFrontEnd("Couldn't establish a connection, please try again2!")
		return
	}
	err = transferFile(server)
	if err != nil {
		log.Println("Error :" + err.Error())
		notifyFrontEnd("Couldn't open " + myPeerInfo.FileName)
		return
	}
}
