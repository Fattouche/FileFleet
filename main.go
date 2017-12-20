package main

import (
	"fmt"
	"net/http"
	"os"
)

/*func getIP(conn net.Conn, ipAddresses chan string) {
	buff := make([]byte, 1)
	conn.Read(buff)
	fmt.Println("PACKET RECIEVED: ")
	fmt.Println(conn.RemoteAddr().String())
	if string(buff) == "1" {
		fmt.Println("got a connection from peer1!")
		peer1IP := conn.RemoteAddr()
		fmt.Println(peer1IP.String())
		ipAddresses <- peer1IP.String()
	} else if string(buff) == "2" {
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
}*/

func getIP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.RemoteAddr)
}

func sendIP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.RemoteAddr)
}

func main() {
	http.HandleFunc("/1", getIP)
	http.HandleFunc("/2", sendIP)
	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
