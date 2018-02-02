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
	FilePath string
	FileSize int64
}

var friend Peer
var myPeerInfo *Peer
var saveLocation string

// BUFFERSIZE used to read from file.
const BUFFERSIZE = 48000

// CentServerAddr used to communicate between peer and rendevouz server.
const CentServerAddr = "18.221.47.86:8080"
const CentServerTrans = "18.221.47.86:8081"

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
	server.SetReadDeadline(time.Now().Add(time.Second * 5))
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

func sendThroughServer(file *os.File, addr string) error {
	notifyFrontEnd("Server")
	//conn, err := net.Dial("tcp", CentServerAddr)
	conn,err:=quic.DialAddr(CentServerTrans, &tls.Config{InsecureSkipVerify: true}, nil)
	defer conn.Close(err)
	if err != nil {
		log.Println("Couldnt connect to central server")
		notifyFrontEnd("We are experiencing network problems, try again later.")
		return err
	}
	stream, err := conn.OpenStreamSync()
	if err != nil {
		log.Println("Couldnt connect to central server")
		notifyFrontEnd("We are experiencing network problems, try again later.")
		return err
	}
	buff, _ := json.Marshal(myPeerInfo)
	stream.Write(buff)
	recvBuff := make([]byte, 10)
	_, err = stream.Read(recvBuff)
	if err != nil {
		return err
	}
	log.Println("Sending through server")
	start := time.Now()
	_,err = io.Copy(stream, file)
	if err!=nil{
		notifyFrontEnd("Couldn't complete the transfer, something went wrong")
		return err
	}
	notifier := fmt.Sprintf("Finished transfer in %.2f seconds!", time.Since(start).Seconds())
	log.Println(notifier)
	notifyFrontEnd(notifier)
	return nil
}

// sendFile sends a file from the server to the addr using Google's quic protocol on top of UDP.
func sendFile(server net.PacketConn, file *os.File, addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	session, err := quic.Dial(server, udpAddr, addr, &tls.Config{InsecureSkipVerify: true}, nil)
	if err != nil {
		log.Println("Error: ", err)
		server.Close()
		err := sendThroughServer(file, addr)
		return err
	}
	defer session.Close(err)
	stream, err := session.OpenStreamSync()
	defer stream.Close()

	log.Println("Sending file!")
	notifyFrontEnd("Connected")
	start := time.Now()

	_,err = io.Copy(stream, file)
	if err!=nil{
		notifyFrontEnd("Couldn't complete the transfer, something went wrong")
		return err
	}

	notifier := fmt.Sprintf("Finished transfer in %.2f seconds!", time.Since(start).Seconds())
	log.Println(notifier)
	notifyFrontEnd(notifier)
	return nil
}

func receieveFromServer(file *os.File) error {
	notifyFrontEnd("Server")
	//conn, err := net.Dial("tcp", CentServerAddr)
	conn,err:=quic.DialAddr(CentServerTrans, &tls.Config{InsecureSkipVerify: true}, nil)
	defer conn.Close(err)
	if err != nil {
		log.Println("Couldnt connect to central server")
		notifyFrontEnd("We are experiencing network problems, try again later.")
		return err
	}
	stream, err := conn.OpenStreamSync()
	if err != nil {
		log.Println("Couldnt connect to central server")
		notifyFrontEnd("We are experiencing network problems, try again later.")
		return err
	}
	buff, _ := json.Marshal(myPeerInfo)
	stream.Write(buff)
	log.Println("Receiving from server")
	start := time.Now()
	_, err = io.Copy(file, stream)
	if err != nil {
		fmt.Println("Error receiving")
		return err
	}
	notifier := fmt.Sprintf("Finished transfer in %.2f seconds!", time.Since(start).Seconds())
	log.Println(notifier)
	notifyFrontEnd(notifier)
	return nil
}

// receiveFile recieves a file from whoever establishes a quic connection with the udp server.
func receiveFile(server net.PacketConn, addr string) error {
	newFile, err := os.Create(saveLocation+"/"+friend.FileName)
	if err != nil {
		log.Println("Error: " + err.Error())
		notifyFrontEnd(err.Error()+"")
		return err
	}
	defer newFile.Close()
	server.SetReadDeadline(time.Now().Add(time.Second * 10))
	connection, err := quic.Listen(server, generateTLSConfig(), nil)
	if err != nil {
		log.Println("Error: " + err.Error())
		return err
	}
	defer connection.Close()
	session, err := connection.Accept()
	server.SetReadDeadline(time.Now().Add(time.Hour*24))
	if err != nil {
		log.Println("Error: " + err.Error())
		server.Close()
		err := receieveFromServer(newFile)
		return err
	}
	defer session.Close(err)

	stream, err := session.AcceptStream()
	if err != nil {
		log.Println("Error: " + err.Error())
		return err
	}
	defer stream.Close()

	log.Println("Recieving file!")
	notifyFrontEnd("Connected")
	start := time.Now()

	io.Copy(newFile, stream)

	notifier := fmt.Sprintf("Finished transfer in %.2f seconds!", time.Since(start).Seconds())
	log.Println(notifier)
	notifyFrontEnd(notifier)
	return nil
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
	var file *os.File
	var err error
	if myPeerInfo.FileName != "" {
		file, err = os.Open(myPeerInfo.FilePath)
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
	//If holepunching failed we know there is no peer in our network
	if myPeerInfo.FileName != "" {
		if public {
			return sendFile(server, file, friend.PubIP)
		}
		return sendFile(server, file, friend.PrivIP)
	}
	return receiveFile(server, myPeerInfo.PrivIP)
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

func initTransfer(peer1, peer2, filePath, directory string) {
	myPeerInfo = new(Peer)
	myPeerInfo.Name = strings.ToLower(peer1)
	myPeerInfo.Friend = strings.ToLower(peer2)
	myPeerInfo.FilePath = filePath
	saveLocation = directory

	if myPeerInfo.FilePath != "" {
		transferFile, err := os.Open(myPeerInfo.FilePath)
		if err != nil {
			log.Println("Error: " + err.Error())
			transferFile.Close()
			notifyFrontEnd("Couldn't open " + myPeerInfo.FilePath)
			return
		}
		fileInfo, _ := transferFile.Stat()
		myPeerInfo.FileName = fileInfo.Name()
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
		notifyFrontEnd("Couldn't establish a connection, please try again!")
		return
	}
	log.Println("Listening on :" + server.LocalAddr().String())
	defer server.Close()
	myPeerInfo.PrivIP = server.LocalAddr().String()
	err = getPeerInfo(server)
	if err != nil {
		log.Println("Error :" + err.Error())
		notifyFrontEnd("Couldn't establish a connection, please try again!")
		return
	}
	err = transferFile(server)
	if err != nil {
		log.Println("Error :" + err.Error())
		if strings.Contains(err.Error(),"permission"){
			notifyFrontEnd(""+err.Error())
		}else{
			notifyFrontEnd("We are experiencing issues, please try again later")
		}
		return
	}
}