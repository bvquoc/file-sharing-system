package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync/atomic"
)

var requestCount int32

type Response struct {
	Message      string `json:"message"`
	ServerName   string `json:"server_name"`
	RequestCount int32  `json:"request_count"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt32(&requestCount, 1)
	serverName := os.Getenv("SERVER_NAME")
	response := Response{
		Message:      "Hello from server",
		ServerName:   serverName,
		RequestCount: atomic.LoadInt32(&requestCount),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":5678", nil))
}
