package main

import (
	"fmt"
	"net"
	"time"

	"github.com/golang/glog"
	"github.com/cactus/go-statsd-client/statsd"
)

type Endpoint struct {
	name    string
	port    int
	peers   []string
	exiting bool
}

var (
	timeout   time.Duration = 100 * time.Millisecond
	sleepTime time.Duration = 5 * time.Second
	maxFailed int           = 3 // Number of times for a Endpoint to fail to consider it down
)

// Starts the receiving server. The protocol is just read a peername
// from the connection and write our own peername (useful for debugging)
func (e *Endpoint) runServer() {
	// Listen for incoming connections
	laddr := fmt.Sprintf(":%d", e.port)
	server, err := net.Listen("tcp", laddr)
	if err != nil {
		glog.Errorln("Error starting server:", err.Error())
		return
	}
	glog.Warningln("Listening on", e.port)
	defer server.Close()

	for !e.exiting {
		conn, err := server.Accept()
		if err != nil {
			glog.Warningln("Error accepting conection", err.Error())
		}

		buf := make([]byte, 256)

		_, err = conn.Read(buf)
		if err != nil {
			glog.Warningln("Error reading:", err.Error())
		} else {
			glog.Infoln("Received message from", string(buf))
			conn.Write([]byte(e.name))
		}
		conn.Close()
	}
}

func (e *Endpoint) checkEndpoints() {
	buf := make([]byte, 256)
	client, err := statsd.NewClient("statsd:8125", "")

	if err != nil {
		glog.Errorln("Error creating stats client", err.Error())
	}

	for ; !e.exiting; time.Sleep(sleepTime) {
		for _, peer := range e.peers {
			start := time.Now()
			conn, err := net.DialTimeout("tcp", peer, timeout)
			if err != nil {
				glog.Warningln("Error connecting", peer, err.Error())
				continue
			}
			elapsed := time.Since(start)
			//conn.SetDeadline(timeout)
			conn.Write([]byte(e.name))
			conn.Read(buf)
			glog.Infof("Correctly connected to %s(%v) in %v\n", string(buf), peer, elapsed)
			stat_name := fmt.Sprintf("%s.%s", e.name, buf)
			client.TimingDuration(stat_name, elapsed, 1.0)
			conn.Close()
		}
	}
}

func (e *Endpoint) Start() {
	go e.runServer()
	go e.checkEndpoints()
}

func (e *Endpoint) AddPeer(address string) {
	glog.Infoln("Adding peer ", address)
	e.peers = append(e.peers, address)
}

func (e *Endpoint) Stop() {
	e.exiting = true
}
