package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

var ipAddresses chan (string)

func getIP(w http.ResponseWriter, r *http.Request) {
	ipAddresses <- r.RemoteAddr
	fmt.Println(r.RemoteAddr)
	io.WriteString(w, r.RemoteAddr)
}

func sendIP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.RemoteAddr)
	ip := <-ipAddresses
	io.WriteString(w, ip)
}

func main() {
	ipAddresses = make(chan (string))
	http.HandleFunc("/1", getIP)
	http.HandleFunc("/2", sendIP)
	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
