package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

var ip string

func getIP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Got connection form peer1")
	ip = r.Header.Get("Host")
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
