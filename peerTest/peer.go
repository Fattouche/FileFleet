package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

type Peer struct {
	privIP   string
	pubIP    string
	name     string
	friend   string
	fileName string
}

var friend Peer

const BUFFERSIZE = 5000

func transferFile() {

}

func getPeerInfo(selfPeer *Peer, server *net.UDPConn) {
	buff, err := json.Marshal(selfPeer)
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
			time.Sleep(1000 * time.Millisecond)
		}
	}()
	for {
		len, _, _ := server.ReadFromUDP(buf)
		fmt.Println(string(buf[:len]))
		if string(buf) == "1" {
			fmt.Println("Connected to server, waiting for information")
			connected = true
		} else if string(buf) == "0" {
			fmt.Println("Server error, please reconnect")
			return
		} else {
			fmt.Println("Recieved information from server: " + string(buf[:len]))
			err = json.Unmarshal(buf[:len], &friend)
			if err != nil {
				fmt.Println("Error: " + err.Error())
				server.WriteToUDP([]byte("0"), serverAddr)
				server.WriteToUDP([]byte("0"), serverAddr)
			} else {
				server.WriteToUDP([]byte("1"), serverAddr)
				server.WriteToUDP([]byte("1"), serverAddr)
			}
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Required myName friendName 'filename'(if you are the one sending) for input")
		return
	}
	selfPeer := new(Peer)
	selfPeer.name = os.Args[0]
	selfPeer.friend = os.Args[1]
	selfPeer.fileName = os.Args[2]

	addr, err := net.ResolveUDPAddr("udp4", ":8081")
	server, err := net.ListenUDP("udp4", addr)
	if err != nil {
		fmt.Println("Error: " + err.Error())
		server.Close()
		panic(err)
	}
	fmt.Println("Listening on :8081")
	defer server.Close()
	selfPeer.privIP = server.LocalAddr().String()
	getPeerInfo(selfPeer, server)
	a, _ := json.Marshal(friend)
	fmt.Println(string(a))
	//transferFile()
}
