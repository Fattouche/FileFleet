package main

import (
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

func main() {
	server, err := net.Listen("tcp", ":"+os.Getenv("PORT"))
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
	}
}
