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
const BUFFERSIZE = 1024

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
func sendFile(server net.PacketConn, file *os.File, addr string) bool {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	config := new(quic.Config)
	config.HandshakeTimeout = time.Millisecond * 500
	session, err := quic.Dial(server, udpAddr, addr, &tls.Config{InsecureSkipVerify: true}, config)
	if err != nil {
		return false
	}
	defer session.Close(err)
	stream, err := session.OpenStreamSync()
	defer stream.Close()

	fmt.Println("Sending file!")
	message := make([]byte, BUFFERSIZE)

	for {
		len, err := file.Read(message)
		if err == io.EOF {
			break
		}
		len, err = stream.Write(message[:len])
	}
	fmt.Println("Sent entire file to peer!")
	return true
}

// receiveFile recieves a file from whoever establishes a quic connection with the udp server.
func receiveFile(server net.PacketConn, addr string) {
	newFile, err := os.Create(friend.FileName)
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	defer newFile.Close()
	connection, err := quic.Listen(server, generateTLSConfig(), nil)
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	defer connection.Close()
	session, err := connection.Accept()
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	defer session.Close(err)

	stream, err := session.AcceptStream()
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	defer stream.Close()

	receivedBytes := int64(0)
	fmt.Println("Recieving file!")
	for receivedBytes < friend.FileSize {
		_, err := io.Copy(newFile, stream)
		if err != nil {
			fmt.Println("Error in reading: ", err)
		}
		receivedBytes += BUFFERSIZE
	}
	fmt.Println("Received file completely from peer!")
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
func transferFile(server *net.UDPConn) {
	var file *os.File
	var err error
	if myPeerInfo.FileName != "" {
		file, err = os.Open(myPeerInfo.FileName)
		if err != nil {
			fmt.Println("Error: " + err.Error())
			return
		}
	}
	addr, _ := net.ResolveUDPAddr("udp", friend.PubIP)
	laddr, _ := net.ResolveUDPAddr("udp", myPeerInfo.PrivIP)

	err = holePunch(server, addr)
	if err != nil {
		time.Sleep(time.Millisecond * 500)
		server, _ = net.ListenUDP("udp", laddr)
	}

	if myPeerInfo.FileName != "" {
		if sendFile(server, file, friend.PubIP) {
			return
		}
	} else {
		receiveFile(server, myPeerInfo.PrivIP)
		return
	}

	addr, _ = net.ResolveUDPAddr("udp", friend.PrivIP)
	server.Close()
	time.Sleep(time.Millisecond * 1000)
	server, _ = net.ListenUDP("udp", laddr)

	sendFile(server, file, friend.PrivIP)
}

// getPeerInfo communicates with the centralized server to exchange information between peers.
func getPeerInfo(server *net.UDPConn) {
	buff, err := json.Marshal(myPeerInfo)
	if err != nil {
		fmt.Println("Error:" + err.Error())
	}
	centUDPAddr, err := net.ResolveUDPAddr("udp", CentServerAddr)
	if err != nil {
		fmt.Println("Error:" + err.Error())
	}
	session, err := quic.Dial(server, centUDPAddr, CentServerAddr, &tls.Config{InsecureSkipVerify: true}, nil)
	if err != nil {
		fmt.Println("Error:" + err.Error())
	}
	//defer session.Close(err)
	stream, err := session.OpenStreamSync()
	if err != nil {
		fmt.Println("Error:" + err.Error())
	}
	//defer stream.Close()
	stream.Write(buff)
	recvBuff := make([]byte, BUFFERSIZE)
	len, _ := stream.Read(recvBuff)
	if string(recvBuff[:len]) == "2" {
		fmt.Println("Both trying to send a file, please re establish")
	}
	fmt.Println("Recieved peer information from server: " + string(recvBuff[:len]))
	json.Unmarshal(recvBuff[:len], &friend)
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

func main() {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		fmt.Println("Required myName friendName 'filename'(if you are the one sending) for input")
		return
	}
	myPeerInfo = new(Peer)
	myPeerInfo.Name = strings.ToLower(os.Args[1])
	myPeerInfo.Friend = strings.ToLower(os.Args[2])
	if len(os.Args) == 4 {
		myPeerInfo.FileName = os.Args[3]
		transferFile, err := os.Open(myPeerInfo.FileName)
		if err != nil {
			fmt.Println("Error: " + err.Error())
			transferFile.Close()
			panic(err)
		}
		fileInfo, _ := transferFile.Stat()
		myPeerInfo.FileSize = fileInfo.Size()
	}

	machineIP, err := externalIP()
	if err != nil {
		fmt.Println("Error getting machine ip: " + err.Error())
	}

	addr, err := net.ResolveUDPAddr("udp", machineIP+":0")
	server, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		server.Close()
		panic(err)
	}
	fmt.Println("Listening on :" + server.LocalAddr().String())
	defer server.Close()
	myPeerInfo.PrivIP = server.LocalAddr().String()
	getPeerInfo(server)
	transferFile(server)
}
