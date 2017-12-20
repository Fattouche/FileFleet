package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

var ip string

func getIP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got connection form peer1")
	ws, _ := upgrader.Upgrade(w, r, nil)
	fmt.Println(ws.RemoteAddr().Network())
	ip = r.Header.Get("X-forwarded-for")
	w.Write([]byte(ip))
}

func sendIP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got connection form peer2")
	io.WriteString(w, ip)
}

func main() {
	http.HandleFunc("/1", getIP)
	http.HandleFunc("/2", sendIP)
	fmt.Println("starting server")
	fmt.Println(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}
