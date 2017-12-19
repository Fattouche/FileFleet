package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
)

const BUFFERSIZE = 5000

//Sends the file to the connection
func sendFile(connection net.Conn) {
	fmt.Println("A client has connected!")
	defer connection.Close()
	file, err := os.Open("islands.mp4")
	if err != nil {
		fmt.Println(err)
		return
	}
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println(err)
		return
	}
	fileSize := strconv.FormatInt(fileInfo.Size(), 10)
	fileName := fileInfo.Name()
	fmt.Println("Sending filename and filesize!")

	connection.Write([]byte(fileSize))
	connection.Write([]byte(fileName))

	sendBuffer := make([]byte, BUFFERSIZE)
	fmt.Println("Start sending file!")
	for {
		_, err = file.Read(sendBuffer)
		if err == io.EOF {
			break
		}
		connection.Write(sendBuffer)
	}
	fmt.Println("File has been sent, closing connection!")
	return
}

//Want to send the IP to the centralized server so that both peers can establish
//connection with eachother.
func sendIP() string {
	connection, err := net.Dial("tcp", "localhost:27000")
	defer connection.Close()
	if err != nil {
		panic(err)
	}
	connection.Write([]byte("1"))
	return connection.LocalAddr().String()
}

//Main catalyst for sending file to peer
func connectToPeer(myIP string) {
	//Listen for connections
	server, err := net.Listen("tcp", myIP)
	defer server.Close()
	if err != nil {
		fmt.Println("Error listetning: ", err)
		os.Exit(1)
	}
	fmt.Println("peer1 started! Waiting for connections...")
	for {
		//Blocks waiting for a connection
		connection, err := server.Accept()
		if err != nil {
			fmt.Println("Error: ", err)
			os.Exit(1)
		}
		fmt.Println("Connected to a peer")
		//Sends file to connected peer
		go sendFile(connection)
	}
}

func main() {
	myIP := sendIP()
	connectToPeer(myIP)
}
