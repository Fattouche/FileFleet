package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

type Peer struct {
	privIP   string
	pubIP    string
	name     string
	friend   string
	isSender string
}

var peerMap map[string]*Peer

func createPeer(len int, buff []byte, publicIP string) (*Peer, error) {
	peer := new(Peer)
	fmt.Println(string(buff[:len]))
	err := json.Unmarshal(buff[:len], &peer)
	fmt.Println(json.Marshal(peer))
	if err != nil {
		fmt.Println("Error in createPeer: " + err.Error())
		return nil, err
	}
	peer.pubIP = publicIP
	peerMap[peer.name] = peer
	return peer, nil
}

func checkPeer(peer *Peer, server *net.UDPConn) {
	addr, err := net.ResolveUDPAddr("udp4", peer.pubIP)
	if err != nil {
		fmt.Println("Error in checkPeer: " + err.Error())
	}
	for {
		if _, ok := peerMap[peer.friend]; ok {
			if peer.isSender != "" || peerMap[peer.friend].isSender != "" {
				fmt.Println("Error: Both peers trying to send a file")
				server.WriteToUDP([]byte("0"), addr)
				return
			}
			msg, err := json.Marshal(peer)
			if err != nil {
				fmt.Println("Error marshalling in checkpeer: " + err.Error())
			}
			server.WriteToUDP([]byte("1"), addr)
			server.WriteToUDP(msg, addr)
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
			fmt.Println("Connecting " + peer.name + " and " + peer.friend)
			server.WriteToUDP([]byte("1"), addr)
		}
		go checkPeer(peer, server)
	}
}
