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

type Packet struct {
	SeqNo int
	Info  []byte
}

var friend Peer
var myPeerInfo *Peer

const BUFFERSIZE = 1024

func holePunch(server *net.UDPConn, addr *net.UDPAddr) {
	connected := false
	go func() {
		for connected != true {
			fmt.Println("Holedpunching")
			server.WriteToUDP([]byte("1"), addr)
			time.Sleep(100 * time.Millisecond)
		}
	}()
	buff := make([]byte, 100)
	for {
		_, recvAddr, _ := server.ReadFromUDP(buff)
		if recvAddr.String() == addr.String() {
			fmt.Println("GOT A HOLEPUNCH!")
			connected = true
			time.Sleep(time.Millisecond * 500)
			return
		}
	}
}

func sendFile(server net.PacketConn, file *os.File, addr string) bool {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	config := new(quic.Config)
	config.HandshakeTimeout = time.Millisecond * 2000
	session, err := quic.Dial(server, udpAddr, addr, &tls.Config{InsecureSkipVerify: true}, config)
	if err != nil {
		return false
	}
	stream, err := session.OpenStreamSync()

	fmt.Println("Sending file!")
	message := make([]byte, 1024)

	for {
		len, err := file.Read(message)
		if err == io.EOF {
			break
		}
		_, err = stream.Write(message[:len])
	}
	fmt.Println("Sent entire file to peer!")
	return true
}

func receiveFile(server net.PacketConn, addr string) bool {
	newFile, err := os.Create(friend.FileName)
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	defer newFile.Close()
	config := new(quic.Config)
	config.IdleTimeout = time.Millisecond * 2000
	connection, err := quic.Listen(server, generateTLSConfig(), config)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		return false
	}
	conn, err := connection.Accept()
	stream, err := conn.AcceptStream()

	receivedBytes := int64(0)
	fmt.Println("Recieving file!")
	for {
		if (friend.FileSize - receivedBytes) < BUFFERSIZE {
			io.CopyN(newFile, stream, (friend.FileSize - receivedBytes))
			stream.Read(make([]byte, (receivedBytes+BUFFERSIZE)-friend.FileSize))
			break
		}
		io.CopyN(newFile, stream, BUFFERSIZE)
		receivedBytes += BUFFERSIZE
	}
	fmt.Println("Received file completely from peer!")
	return true
}

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
	Sent := false
	Recieved := false
	if myPeerInfo.FileName != "" {
		Sent = sendFile(server, file, friend.PrivIP)
	} else {
		Recieved = receiveFile(server, myPeerInfo.PrivIP)
	}
	if Sent == true || Recieved == true {
		return
	}
	server.Close()
	time.Sleep(time.Millisecond * 1000)

	addr, _ := net.ResolveUDPAddr("udp4", friend.PubIP)
	laddr, _ := net.ResolveUDPAddr("udp4", myPeerInfo.PrivIP)
	server, err = net.ListenUDP("udp4", laddr)
	defer server.Close()

	holePunch(server, addr)

	if myPeerInfo.FileName != "" {
		sendFile(server, file, friend.PubIP)
	} else {
		receiveFile(server, myPeerInfo.PrivIP)
	}
}

func getPeerInfo(server *net.UDPConn) {
	buff, err := json.Marshal(myPeerInfo)
	if err != nil {
		fmt.Println("Error:" + err.Error())
	}
	serverAddr, err := net.ResolveUDPAddr("udp4", "18.221.47.86:8080")
	if err != nil {
		fmt.Println("Error:" + err.Error())
	}
	server.WriteToUDP(buff, serverAddr)
	connected := false
	buf := make([]byte, 1000)
	go func() {
		for connected != true {
			server.WriteToUDP(buff, serverAddr)
			time.Sleep(3000 * time.Millisecond)
		}
	}()
	for {
		len, _, _ := server.ReadFromUDP(buf)
		if string(buf[:len]) == "1" {
			connected = true
		} else if string(buf[:len]) == "0" {
			fmt.Println("Server error, please reconnect")
			return
		} else {
			fmt.Println("Recieved peer information from server: " + string(buf[:len]))
			err = json.Unmarshal(buf[:len], &friend)
			if err != nil {
				fmt.Println("Error: " + err.Error())
			}
			break
		}
	}
}

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
	myPeerInfo.Name = os.Args[1]
	myPeerInfo.Friend = os.Args[2]
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

	addr, err := net.ResolveUDPAddr("udp4", machineIP+":0")
	server, err := net.ListenUDP("udp4", addr)
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
