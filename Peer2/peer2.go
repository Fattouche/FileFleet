package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
)

const BUFFERSIZE = 5000

func readFromPeer(ip string) {
	connection, err := net.Dial("tcp", ip)
	defer connection.Close()
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected to server, start receiving the file name and file size")
	bufferFileName := make([]byte, 64)
	bufferFileSize := make([]byte, 10)

	connection.Read(bufferFileSize)
	fileSize, _ := strconv.ParseInt(string(bufferFileSize), 10, 64)

	fileNameLength, _ := connection.Read(bufferFileName)
	fileName := string(bufferFileName[:fileNameLength])

	recievedFile, err := os.Create(fileName)
	defer recievedFile.Close()
	if err != nil {
		panic(err)
	}

	var receivedBytes int64

	for {
		if (fileSize - receivedBytes) < BUFFERSIZE {
			io.CopyN(recievedFile, connection, (fileSize - receivedBytes))
			connection.Read(make([]byte, (receivedBytes+BUFFERSIZE)-fileSize))
			break
		}
		io.CopyN(recievedFile, connection, BUFFERSIZE)
		receivedBytes += BUFFERSIZE
	}
	fmt.Println("Received file completely!")
}

func getIPFromServer() string {
	connection, err := net.Dial("tcp", "localhost:27000")
	defer connection.Close()
	if err != nil {
		panic(err)
	}
	connection.Write([]byte("2"))
	buff := make([]byte, 15)
	connection.Read(buff)
	ipAddress := string(buff)
	return ipAddress
}

func main() {
	ip := getIPFromServer()
	fmt.Println(ip)
	readFromPeer(ip)
}
