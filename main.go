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
	session, err := quic.Dial(server, udpAddr, addr, &tls.Config{InsecureSkipVerify: true}, nil)
	if err != nil {
		fmt.Println("Error: ", err)
	}
	defer session.Close(err)
	stream, err := session.OpenStreamSync()
	defer stream.Close()

	fmt.Println("Sending file!")
	start := time.Now()

	io.Copy(stream, file)

	fmt.Printf("Sent entire file to peer in %f seconds!", time.Since(start).Seconds())
}

// receiveFile recieves a file from whoever establishes a quic connection with the udp server.
func receiveFile(server net.PacketConn, addr string) {
	newFile, err := os.Create(friend.FileName)
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	defer newFile.Close()
	config := new(quic.Config)

	//Max flow control windows
	config.MaxReceiveStreamFlowControlWindow = 0
	config.MaxReceiveConnectionFlowControlWindow = 0
	connection, err := quic.Listen(server, generateTLSConfig(), config)
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

	fmt.Println("Recieving file!")
	start := time.Now()

	io.Copy(newFile, stream)

	fmt.Printf("Received file completely from peer in %f seconds!", time.Since(start).Seconds())
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
	public := true

	err = holePunch(server, addr)
	if err != nil {
		server.Close()
		time.Sleep(time.Millisecond * 500)
		server, _ = net.ListenUDP("udp", laddr)
		public = false
	}

	//If holepunching failed we know there is no peer in our network
	if myPeerInfo.FileName != "" {
		if public {
			sendFile(server, file, friend.PubIP)
		} else {
			sendFile(server, file, friend.PrivIP)
		}
	} else {
		receiveFile(server, myPeerInfo.PrivIP)
	}
}

// getPeerInfo communicates with the centralized server to exchange information between peers.
func getPeerInfo(server *net.UDPConn) error {
	buff, err := json.Marshal(myPeerInfo)
	if err != nil {
		fmt.Println("Error:" + err.Error())
		return err
	}
	centUDPAddr, err := net.ResolveUDPAddr("udp", CentServerAddr)
	if err != nil {
		fmt.Println("Error:" + err.Error())
		return err
	}
	session, err := quic.Dial(server, centUDPAddr, CentServerAddr, &tls.Config{InsecureSkipVerify: true}, nil)
	if err != nil {
		fmt.Println("Error:" + err.Error())
		return err
	}
	defer session.Close(err)
	stream, err := session.OpenStreamSync()
	if err != nil {
		fmt.Println("Error:" + err.Error())
		return err
	}
	defer stream.Close()
	stream.Write(buff)
	recvBuff := make([]byte, BUFFERSIZE)
	len, _ := stream.Read(recvBuff)
	if string(recvBuff[:len]) == "2" {
		return errors.New("Both trying to send a file, please try again")
	}
	fmt.Println("Recieved peer information from server: " + string(recvBuff[:len]))
	err = json.Unmarshal(recvBuff[:len], &friend)
	if err != nil {
		fmt.Println("Error:" + err.Error())
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
	err = getPeerInfo(server)
	if err != nil {
		fmt.Println("Error :" + err.Error())
		return
	}
	transferFile(server)
}
