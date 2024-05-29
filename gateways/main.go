package main

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type Server struct {
	URL           *url.URL
	Alive         bool
	ReverseProxy  *httputil.ReverseProxy
	Mutex         sync.RWMutex
	RequestCount  int
	TotalResponse time.Duration
	ResponseCount int
}

type LoadBalancer struct {
	Servers      []*Server
	CurrentIndex int
	Mutex        sync.Mutex
}

func NewLoadBalancer(serverUrls []string) *LoadBalancer {
	var servers []*Server
	for _, serverUrl := range serverUrls {
		url, err := url.Parse(serverUrl)
		if err != nil {
			log.Fatal(err)
		}
		servers = append(servers, &Server{
			URL:          url,
			Alive:        true,
			ReverseProxy: httputil.NewSingleHostReverseProxy(url),
		})
	}
	return &LoadBalancer{Servers: servers}
}

func (lb *LoadBalancer) getNextServer() *Server {
	lb.Mutex.Lock()
	defer lb.Mutex.Unlock()

	for i := 0; i < len(lb.Servers); i++ {
		server := lb.Servers[lb.CurrentIndex]
		lb.CurrentIndex = (lb.CurrentIndex + 1) % len(lb.Servers)
		if server.Alive {
			return server
		}
	}
	return nil
}

func (lb *LoadBalancer) healthCheck() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for _, server := range lb.Servers {
			go func(s *Server) {
				alive := isAlive(s.URL)
				s.Mutex.Lock()
				s.Alive = alive
				s.Mutex.Unlock()
				log.Printf("Health check for %s, alive: %t", s.URL, alive)
			}(server)
		}
	}
}

func isAlive(u *url.URL) bool {
	conn, err := net.DialTimeout("tcp", u.Host, 5*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for retries := 0; retries < len(lb.Servers); retries++ {
		server := lb.getNextServer()
		if server == nil {
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
			return
		}

		server.Mutex.Lock()
		server.RequestCount++
		server.Mutex.Unlock()

		start := time.Now()

		proxyErrorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("Error proxying to %s: %v", server.URL, err)
			server.Mutex.Lock()
			server.Alive = false
			server.Mutex.Unlock()
			lb.ServeHTTP(w, r)
		}

		server.ReverseProxy.ErrorHandler = proxyErrorHandler
		server.ReverseProxy.ServeHTTP(w, r)

		duration := time.Since(start)

		server.Mutex.Lock()
		server.RequestCount--
		server.TotalResponse += duration
		server.ResponseCount++
		server.Mutex.Unlock()

		break
	}
}

func (lb *LoadBalancer) getLeastConnectionServer() *Server {
	lb.Mutex.Lock()
	defer lb.Mutex.Unlock()

	var leastConnServer *Server
	minConn := int(^uint(0) >> 1) // Initialize to max int value

	for _, server := range lb.Servers {
		if server.Alive && server.RequestCount < minConn {
			leastConnServer = server
			minConn = server.RequestCount
		}
	}
	return leastConnServer
}

func main() {
	serverUrls := []string{
		"http://app1:5678",
		"http://app2:5678",
		"http://app3:5678",
		"http://app4:5678",
	}

	lb := NewLoadBalancer(serverUrls) // Initialize the load balancer with the given server URLs
	go lb.healthCheck()               // Start the health check in a separate goroutine

	http.HandleFunc("/", lb.ServeHTTP)             // Set the HTTP handler function for the load balancer
	log.Println("Starting load balancer on :8080") // Log that the load balancer is starting
	log.Fatal(http.ListenAndServe(":8080", nil))   // Start the HTTP server on port 8080 and log any fatal errors
}
