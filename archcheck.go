package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

const configfile string = "archcheck.json"

type ReferenceArchitectures []struct {
	Architecture string `json:"architecture,omitempty"`
	Servers      []struct {
		Type          string `json:"type"`
		InboundPorts  []int  `json:"inbound ports"`
		OutboundPorts []int  `json:"outbound ports"`
	} `json:"servers"`
	Validation []struct {
		Type  string `json:"type"`
		Tests []struct {
			From  string `json:"from,omitempty"`
			Ports []int  `json:"ports"`
			To    string `json:"to,omitempty"`
		} `json:"tests"`
	} `json:"validation"`
}

func readConfig() ReferenceArchitectures {
	// read config file
	file, err := os.ReadFile(configfile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't open config file %q: %s", configfile, err)
		os.Exit(1)
	}

	data := ReferenceArchitectures{}

	err = json.Unmarshal([]byte(file), &data)
	if err != nil {
		fmt.Println("Can not unmarshal JSON")
		os.Exit(1)
	}

	return data
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: archcheck\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	config := readConfig()

	flag.Usage = usage
	flag.Parse()

	var host_ports = make(map[string]int, 0)

	fmt.Println("Available Architectures:")
	for _, arch := range config {
		fmt.Println(" - " + arch.Architecture)
		// get all inbound ports from all servers and architectures
		for _, server := range arch.Servers {
			for _, port := range server.InboundPorts {
				if host_ports[strconv.Itoa(port)] != port {
					host_ports[strconv.Itoa(port)] = port
				}
			}
		}
	}
	fmt.Println("Host Ports: ", host_ports)

	for port := range host_ports {
		go checkPort(port)
	}

	// args := flag.Args()
	// if len(args) < 1 {
	// 	fmt.Fprintf(os.Stderr, "Input ports are missing.")
	// 	os.Exit(1)
	// }

	// for port := range args {
	// 	go checkPort(args[port])
	// }

	for {
		time.Sleep(1000)
	}
}

func checkPort(port string) {
	_, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid port %q: %s", port, err)
		return
	}

	ln, err := net.Listen("tcp", ":"+port)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't listen on port %q: %s", port, err)
		return
	}

	fmt.Printf("TCP Port %q is available\n", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			fmt.Printf("Error accepting connection: %s\n", err)
		}
		fmt.Printf("Connection accepted from %s on Port %q\n", conn.RemoteAddr(), port)

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		// Read data from the client
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		// Process and use the data (here, we'll just print it)
		fmt.Printf("Received: %s\n", buffer[:n])
	}
}
