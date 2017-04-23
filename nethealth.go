package main

import (
	"flag"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	endpoints []string
	peername  string
	port      int  = 3333
	exiting   bool = false
)

func abort(a ...interface{}) {
	glog.Fatalln(a...)
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

	port, err = strconv.Atoi(readValue("PORT", "3333"))
	if err != nil {
		abort("Error reading port config:", err.Error())
	}

	// Read a comma separated list of endpoints
	// e.g. peer1:port1,peer2:port2 ...
	ep := readValue("ENDPOINTS", "")
	if ep != "" {
		for _, e := range strings.Split(ep, ",") {
			if !strings.Contains(e, ":") {
				// Add default port if missing from the endpoint
				e = net.JoinHostPort(e, "3333")
			}
			glog.Infoln("Adding endpoint", e)
			endpoints = append(endpoints, e)
		}
	}
}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func main() {
	flag.Parse()
	readConfig()

	if len(endpoints) < 1 {
		// If we don't have endpoints we try to read them from K8s API

		kubeconfig := flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")

		// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
		config, err := buildConfig(*kubeconfig)
		if err != nil {
			panic(err)
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}

		// TODO Automatically get the list of endpoints
		// Right now this just list all the pods. We should filter them somehow to get
		// nethealth entpoints
		for {
			pods, err := clientset.CoreV1().Pods("").List(v1.ListOptions{})
			if err != nil {
				panic(err.Error())
			}
			glog.Infof("There are %d pods in the cluster\n", len(pods.Items))
			for _, pod := range pods.Items {
				glog.Infof("Pod %s @ %s\n", pod.Name, pod.Status.PodIP)
			}
			time.Sleep(10 * time.Second)
		}

	}

	endpoint := Endpoint{name: peername, port: port}
	endpoint.Start()
	defer endpoint.Stop()

	for _, e := range endpoints {
		endpoint.AddPeer(e)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    <-c
	glog.Warningln("Received SIGTERM, exiting")
    os.Exit(1)
}
