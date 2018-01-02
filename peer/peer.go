package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"
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

func sendFile(server *net.UDPConn, file *os.File, addr *net.UDPAddr) {
	packet := new(Packet)
	packet.SeqNo = 0
	sendBuffer := make([]byte, BUFFERSIZE)
	recvBuffer := make([]byte, 10)
	var err error
	done := false
	fmt.Println("sending file!")
	go func() {
		for done == false {
			len, err := file.ReadAt(sendBuffer, int64(packet.SeqNo))
			packet.SeqNo += len
			packet.Info = sendBuffer[:len]
			msg, err := json.Marshal(&packet)
			if err != nil {
				fmt.Println("Error: " + err.Error())
			}
			_, err = server.WriteToUDP(msg, addr)
			if err != nil {
				fmt.Println("Error: " + err.Error())
			}
			time.Sleep(time.Microsecond * 5)
		}
		return
	}()
	prevBytesReceived := 0
	bytesRecieved := int64(0)
	for done == false {
		len, _, _ := server.ReadFromUDP(recvBuffer)
		prevBytesReceived = int(bytesRecieved)
		bytesRecieved, err = strconv.ParseInt(string(recvBuffer[:len]), 10, 64)
		if err != nil {
			fmt.Println("Error: " + err.Error())
		}
		if bytesRecieved == myPeerInfo.FileSize {
			done = true
		}
		if int(bytesRecieved) == prevBytesReceived {
			packet.SeqNo = int(bytesRecieved)
		}
	}
}

func receiveFile(server *net.UDPConn, addr *net.UDPAddr) {
	recvBuffer := make([]byte, BUFFERSIZE+1000)
	packet := new(Packet)

	newFile, err := os.Create(friend.FileName)
	if err != nil {
		fmt.Println("Error: " + err.Error())
	}
	defer newFile.Close()

	receivedBytes := int64(0)

	for receivedBytes < friend.FileSize {
		len, err := server.Read(recvBuffer)
		if err != nil {
			fmt.Println("Error: " + err.Error())
		}
		json.Unmarshal(recvBuffer[:len], &packet)
		if int64(packet.SeqNo) != receivedBytes+BUFFERSIZE {
			if int64(packet.SeqNo) == friend.FileSize {
				newFile.Write(packet.Info)
				server.WriteToUDP([]byte(strconv.Itoa(int(friend.FileSize))), addr)
				server.WriteToUDP([]byte(strconv.Itoa(int(friend.FileSize))), addr)
				fmt.Println("receieved entire file!")
				return
			}
			server.WriteToUDP([]byte(strconv.Itoa(int(receivedBytes))), addr)
		} else {
			newFile.Write(packet.Info)
			receivedBytes = int64(packet.SeqNo)
		}
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
	if transferInsideNetwork(file) == nil {
		return
	}

	addr, _ := net.ResolveUDPAddr("udp4", friend.PubIP)
	holePunch(server, addr)
	if myPeerInfo.FileName != "" {
		sendFile(server, file, addr)
	} else {
		receiveFile(server, addr)
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
