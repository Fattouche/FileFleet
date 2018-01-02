package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/oxtoacart/go-udt/udt"
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
			server.WriteToUDP([]byte("1"), addr)
			time.Sleep(100 * time.Millisecond)
		}
	}()
	buff := make([]byte, 100)
	for {
		_, recvAddr, _ := server.ReadFromUDP(buff)
		if recvAddr.String() == addr.String() {
			connected = true
			time.Sleep(time.Millisecond * 500)
			return
		}
	}
}

//If in same network, dont need holepunching!
func transferInsideNetwork(file *os.File) error {
	if myPeerInfo.FileName != "" {
		addr, _ := net.ResolveTCPAddr("tcp", myPeerInfo.PrivIP)
		server, err := net.ListenTCP("tcp", addr)
		server.SetDeadline(time.Now().Add(time.Millisecond * 1000))
		if err != nil {
			fmt.Println("Error listetning: ", err)
			return err
		}
		defer server.Close()
		connection, err := server.AcceptTCP()
		if err != nil {
			return err
		}
		sendBuffer := make([]byte, BUFFERSIZE)
		for {
			_, err = file.Read(sendBuffer)
			if err == io.EOF {
				break
			}
			connection.Write(sendBuffer)
		}
		fmt.Println("File has been sent, closing connection with peer!")
	} else {
		localAddr, _ := net.ResolveTCPAddr("tcp", myPeerInfo.PrivIP)
		dialer := &net.Dialer{Timeout: 1 * time.Second, LocalAddr: localAddr}
		connection, err := dialer.Dial("tcp", friend.PrivIP)
		if err != nil {
			return err
		}
		defer connection.Close()
		newFile, err := os.Create(friend.FileName)
		defer newFile.Close()

		var receivedBytes int64

		for {
			if (friend.FileSize - receivedBytes) < BUFFERSIZE {
				io.CopyN(newFile, connection, (friend.FileSize - receivedBytes))
				connection.Read(make([]byte, (receivedBytes+BUFFERSIZE)-friend.FileSize))
				break
			}
			io.CopyN(newFile, connection, BUFFERSIZE)
			receivedBytes += BUFFERSIZE
		}
		fmt.Println("Received file completely from peer!")
	}
	return nil
}

func sendFile(file *os.File) {
	sendBuffer := make([]byte, BUFFERSIZE)
	var err error

	laddr, _ := net.ResolveUDPAddr("udp", myPeerInfo.PrivIP)
	listener, _ := udt.ListenUDT("udp", laddr)
	time.Sleep(time.Millisecond * 500)
	fmt.Println("sending file!")
	conn, _ := listener.Accept()

	for {
		_, err = file.Read(sendBuffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error: " + err.Error())
		}
		_, err = conn.Write(sendBuffer)
		if err != nil {
			fmt.Println("Error: " + err.Error())
		}
	}
	fmt.Println("File has been sent, closing connection with peer!")
}

func receiveFile(addr *net.UDPAddr) {
	laddr, _ := net.ResolveUDPAddr("udp", myPeerInfo.PrivIP)

	newFile, err := os.Create(friend.FileName)
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	defer newFile.Close()

	time.Sleep(time.Millisecond * 500)
	connection, err := udt.DialUDT("udp", laddr, addr)
	var receivedBytes int64

	fmt.Println("receiving file!")
	for {
		if (friend.FileSize - receivedBytes) < BUFFERSIZE {
			io.CopyN(newFile, connection, (friend.FileSize - receivedBytes))
			connection.Read(make([]byte, (receivedBytes+BUFFERSIZE)-friend.FileSize))
			break
		}
		io.CopyN(newFile, connection, BUFFERSIZE)
		receivedBytes += BUFFERSIZE
	}

	fmt.Println("Received file completely from peer!")
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
	addr, _ := net.ResolveUDPAddr("udp", friend.PrivIP)
	if myPeerInfo.FileName != "" {
		sendFile(file)
	} else {
		receiveFile(addr)
	}
	addr, _ = net.ResolveUDPAddr("udp", friend.PubIP)
	holePunch(server, addr)
	if myPeerInfo.FileName != "" {
		sendFile(file)
	} else {
		receiveFile(addr)
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
