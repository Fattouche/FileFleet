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
}

var friend Peer
var myPeerInfo *Peer

const BUFFERSIZE = 5000

func holePunch() {

}

//If in same network, dont need holepunching!
func transferInsideNetwork(file *os.File) error {
	if myPeerInfo.FileName != "" {
		fileInfo, err := file.Stat()
		if err != nil {
			fmt.Println("Error: " + err.Error())
			return err
		}
		addr, _ := net.ResolveTCPAddr("tcp", myPeerInfo.PrivIP)
		server, err := net.ListenTCP("tcp", addr)
		server.SetDeadline(time.Now().Add(time.Millisecond * 5000))
		if err != nil {
			fmt.Println("Error listetning: ", err)
			return err
		}
		defer server.Close()
		connection, err := server.AcceptTCP()
		if err != nil {
			return err
		}
		connection.Write([]byte(fileInfo.Name()))
		connection.Write([]byte(strconv.Itoa(int(fileInfo.Size()))))
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
		dialer := &net.Dialer{Timeout: 5 * time.Second, LocalAddr: localAddr}
		connection, err := dialer.Dial("tcp", friend.PrivIP)
		if err != nil {
			return err
		}
		defer connection.Close()
		bufferFileName := make([]byte, 100)
		bufferFileSize := make([]byte, 10)

		len, err := connection.Read(bufferFileName)
		fileName := string(bufferFileName[:len])
		len, err = connection.Read(bufferFileSize)
		fileSize, err := strconv.ParseInt(string(bufferFileSize), 10, 64)
		if err != nil {
			fmt.Println("Error: " + err.Error())
		}
		newFile, err := os.Create(fileName)
		defer newFile.Close()

		var receivedBytes int64

		for {
			if (fileSize - receivedBytes) < BUFFERSIZE {
				io.CopyN(newFile, connection, (fileSize - receivedBytes))
				connection.Read(make([]byte, (receivedBytes+BUFFERSIZE)-fileSize))
				break
			}
			io.CopyN(newFile, connection, BUFFERSIZE)
			receivedBytes += BUFFERSIZE
		}
		fmt.Println("Received file completely from peer!")

	}
	return nil
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
	err = transferInsideNetwork(file)
	//succesful tcp transfer inside the network
	if err == nil {
		return
	}
}

func getPeerInfo(server *net.UDPConn) {
	buff, err := json.Marshal(myPeerInfo)
	if err != nil {
		fmt.Println("Error:" + err.Error())
	}
	serverAddr, err := net.ResolveUDPAddr("udp4", "18.217.212.81:8080")
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
