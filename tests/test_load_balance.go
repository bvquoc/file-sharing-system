package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	requestCount = 10000
	concurrency  = 10
	url          = "http://localhost:5010"
	maxRetries   = 3 // Maximum number of retries for a failed request
)

type Response struct {
	Message      string `json:"message"`
	ServerName   string `json:"server_name"`
	RequestCount int32  `json:"request_count"`
}

func main() {
	// Open a log file
	logFile, err := os.OpenFile("request_log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Set log output to the file
	log.SetOutput(logFile)

	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			client := &http.Client{}
			for j := 0; j < requestCount/concurrency; j++ {
				var response Response
				var err error
				for k := 0; k < maxRetries; k++ {
					resp, err := client.Get(url)
					if err == nil && resp.StatusCode == http.StatusOK {
						defer resp.Body.Close()
						err = json.NewDecoder(resp.Body).Decode(&response)
						if err == nil {
							break
						}
					}
					if k < maxRetries-1 {
						time.Sleep(100 * time.Millisecond) // Wait before retrying
					}
				}
				if err != nil {
					log.Printf("Request failed after %d retries: %v", maxRetries, err)
				} else {
					log.Printf("Response: %+v\n", response)
				}
			}
		}()
	}

	wg.Wait()
	log.Println("All requests completed")
}

//10 cuc file , 5 server => moi tgang 2 cuc
// server 2 chua cuc 2 cuc 3
// out, vo
// authen, authorization

// replicate

// client => upload anh, thi t up 10 anh, server => api gate way
// localhost:8000/upload

// 1 request => 1 file => 1 server
// 10 request => 10 file => 10 server

// user + auth xu ly ben gateway
//
