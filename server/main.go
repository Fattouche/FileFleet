package main

import (
<<<<<<< HEAD
	"fmt"
	"net"
	"os"
)

func getIP(conn net.Conn, ipAddresses chan string) {
	buff := make([]byte, 1)
	conn.Read(buff)
	if string(buff) == "1" {
		fmt.Println("got a connection from peer1!")
		peer1IP := conn.RemoteAddr()
		fmt.Println(peer1IP.String())
		ipAddresses <- peer1IP.String()
	} else {
		fmt.Println("got a connection from peer2!")
		peer1IP := <-ipAddresses
		conn.Write([]byte(peer1IP))
	}
}

func determineListenAddress() (string, error) {
	port := os.Getenv("PORT")
	if port == "" {
		return "", fmt.Errorf("$PORT not set")
	}
	return ":" + port, nil
}

func main() {
	addr, err := determineListenAddress()
	if err != nil {
		panic(err)
	}

	server, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer server.Close()
	ipAddresses := make(chan string)
	fmt.Println("Waiting for connections from peers")
	for {
		//Blocks waiting for a connection
		connection, err := server.Accept()
		if err != nil {
			fmt.Println("Error: ", err)
			os.Exit(1)
		}
		//get IP address of peers
		go getIP(connection, ipAddresses)
=======
	"encoding/json"
	"fmt"
	"net"
	"os"
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

var peerMap map[string]*Peer

func createPeer(len int, buff []byte, publicIP string) (*Peer, error) {
	peer := new(Peer)
	err := json.Unmarshal(buff[:len], &peer)
	if err != nil {
		fmt.Println("Error in createPeer: " + err.Error())
		return nil, err
	}
	peer.PubIP = publicIP
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
>>>>>>> 72f11adb811ee407b4647f058e6caaba22a46d4d
	}
}
