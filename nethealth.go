package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type endpoint struct {
	address string
	ok      bool
	failed  int
}

var (
	l         *log.Logger
	endpoints []endpoint
	peername  string
	debug     bool          = true
	port      int           = 3333
	exiting   bool          = false
	timeout   time.Duration = 100 * time.Millisecond
	sleepTime time.Duration = 1 * time.Second
	maxFailed int           = 3 // Number of times for a endpoint to fail to consider it down
)

// Read a received connection. The protocol is just read a peername
// from the connection and write our own peername (useful for debugging)
func handleRequest(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 256)

	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	} else {
		//		fmt.Println("Received message from", string(buf))
		conn.Write([]byte(peername))
	}
}

func checkEndpoints() {
	buf := make([]byte, 256)
	for ; !exiting; time.Sleep(sleepTime) {
		for _, e := range endpoints {
			start := time.Now()
			conn, err := net.DialTimeout("tcp", e.address, timeout)
			if err != nil {
				fmt.Println("Error connecting", e.address, err.Error())
				continue
			}
			elapsed := time.Since(start)
			//conn.SetDeadline(timeout)
			conn.Write([]byte(peername))
			conn.Read(buf)
			fmt.Printf("Correctly connected to %s(%v) in %v\n", string(buf), e.address, elapsed)
			conn.Close()
		}
	}
}

func abort(a ...interface{}) {
	l.Println(a...)
	os.Exit(1)
}

func readConfig() {
	readValue := func(name string, defaultValue string) string {
		if os.Getenv(name) != "" {
			return os.Getenv(name)
		} else {
			return defaultValue
		}
	}

	hostname, err := os.Hostname()
	peername = readValue("PEERNAME", hostname)
	debug = readValue("DEBUG", "0") == "1"
	port, err = strconv.Atoi(readValue("PORT", "3333"))
	if err != nil {
		abort("Missing port config. Error:", err.Error())
	}

	// Read a comma separated list of endpoints
	ep := readValue("ENDPOINTS", "")
	if ep != "" {
		for _, e := range strings.Split(ep, ",") {
			if ! strings.Contains(e, ":") {
				// Add default port
				e = net.JoinHostPort(e, "3333")
			}
			l.Println("Adding endpoint", e)
			endpoints = append(endpoints, endpoint{e, true, 0})
		}
	}
}

func main() {
	l = log.New(os.Stderr, "", 0)
	readConfig()

	// Listen for incoming connections
	laddr := fmt.Sprintf(":%d", port)
	server, err := net.Listen("tcp", laddr)
	if err != nil {
		abort("Error starting server:", err.Error())
	}

	defer server.Close()
	fmt.Println("Listening on", port)

	go checkEndpoints()

	for {
		conn, err := server.Accept()
		if err != nil {
			l.Println("Error accepting", err.Error())
		}

		go handleRequest(conn)
	}
}
