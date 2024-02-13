package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

const ConfigFile string = "archcheck.json"

var NodeGroups NodeGroup
var NodePorts NodePortGroup
var Nodes []Node

type NodeGroup struct {
	Nodes []Node
}
type NodePortGroup struct {
	Ports []NodePort `json:"Ports,omitempty"`
}

type NodePort struct {
	Port    int  `json:"port"`
	Success bool `json:"success"`
}

type Node struct {
	Hostname    string     `json:"hostname"`
	IP          string     `json:"ip"`
	Ports       []NodePort `json:"Ports,omitempty"`
	Connections []struct {
		ToHostname  string `json:"to hostname"`
		ToIPAddress string `json:"to ip"`
		Port        int    `json:"port"`
		Success     bool   `json:"success"`
	} `json:"Connections,omitempty"`
}

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
	file, err := os.ReadFile(ConfigFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't open config file %q: %s", ConfigFile, err)
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
	fmt.Fprintf(os.Stderr, "usage: archcheck <primary hostname/IP>\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	config := readConfig()

	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Target host is missing.")
		os.Exit(1)
	}

	go registerNode(args[0])
	go updateStatus(args[0])

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

	http.HandleFunc("/", userInterface)
	http.HandleFunc("/register", register)
	http.HandleFunc("/updateStatus", receiveStatus)
	http.HandleFunc("/data/NodeData.json", NodeGroupsToJSON)
	http.Handle("/graphs/", http.StripPrefix("/graphs/", http.FileServer(http.Dir("./graphs"))))

	for port := range host_ports {
		// go checkPort(port)
		fmt.Println("Checking Port: ", port)
		go servePort(port)
	}

	// for port := range args {
	// 	go checkPort(args[port])
	// }

	for {
		time.Sleep(10 * time.Second)
	}
}

// GetLocalIP returns the non loopback local IP of the host
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func registerNode(target string) {
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println("Unable to get hostname when registering node Error:", err)
		return
	}
	myNode := Node{
		Hostname: hostname,
		IP:       GetLocalIP(),
	}
	nodeJSON, err := json.Marshal(myNode)
	if err != nil {
		fmt.Println("Unable to marshal JSON when registering node Error:", err)
		return
	}

	//send request to target
	for {
		_, err := http.Post("http://"+target+":443/register", "application/json", bytes.NewBuffer(nodeJSON))
		if err != nil {
			fmt.Println("Unable to connect to target Error:", err)
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}
}

func updateStatus(target string) {
	for {
		nodeJSON, err := json.Marshal(NodePorts)
		if err != nil {
			fmt.Println("Unable to marshal JSON when updating status node Error:", err)
			return
		}

		fmt.Println("Node Ports: ", bytes.NewBuffer(nodeJSON))

		//send request to target
		for {
			_, err := http.Post("http://"+target+":443/updateStatus", "application/json", bytes.NewBuffer(nodeJSON))
			if err != nil {
				fmt.Println("Unable to connect to target Error:", err)
				time.Sleep(3 * time.Second)
			} else {
				break
			}
		}
		time.Sleep(10 * time.Second)
	}
}

func servePort(port string) {
	fmt.Println("Serving Port ", port)
	_, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid port %q: %s", port, err)
		return
	}

	portInt, _ := strconv.Atoi(port)

	var listenErr error
	go func() { listenErr = http.ListenAndServe(":"+port, nil) }()

	// need to correct for race condition as listenErr is not set yet
	if listenErr != nil {
		fmt.Fprintf(os.Stderr, "Can't listen on port %q: %s", port, err)
		addPortToNode(portInt, false)
		return
	}
	fmt.Printf("Listening on Port %q\n", port)

	addPortToNode(portInt, true)
}

func addPortToNode(port int, success bool) {
	fmt.Println("Adding Port ", port)
	for _, nodePort := range NodePorts.Ports {
		if nodePort.Port == port {
			nodePort.Success = success
			return
		}
	}
	NodePorts.Ports = append(NodePorts.Ports, NodePort{port, success})
	fmt.Println("Status: ", port, success)
	fmt.Println("Node Ports ", NodePorts)
}

func userInterface(w http.ResponseWriter, r *http.Request) {
	contents, err := loadPage("index")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't load page: %s", err)
		return
	}
	fmt.Printf("Connection accepted from %s\n", r.Host)

	fmt.Fprintf(w, "%s", string(contents))
}

// take the JSON data from the connection the register the node
func register(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Registration request from %s\n", r.Host)
	var myNode Node

	err := json.NewDecoder(r.Body).Decode(&myNode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	Nodes = append(Nodes, myNode)
	fmt.Println("Node Group ", Nodes)
	fmt.Fprintf(w, "OK")
}

// take the JSON data from the connection the register the node
func receiveStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("ReceiveStatus from %s\n", r.Host)
	var myNode NodePortGroup

	err := json.NewDecoder(r.Body).Decode(&myNode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println(myNode)

	updateNodeGroup(r.Host, myNode)

	fmt.Fprintf(w, "OK")
}

func updateNodeGroup(host string, nodePort NodePortGroup) {
	// for _, nodeGroup := range NodeGroups {
	// 	if nodeGroup.Hostname == node.Hostname {
	// 		nodeGroup.addNode(node)
	// 	}
	// }
	fmt.Println("Node Ports ", nodePort)
}

// func (nodeGroup *NodeGroup) addNode(node Node) {
// 	nodeGroup.Nodes = append(nodeGroup.Nodes, node)
// }

func loadPage(page string) ([]byte, error) {
	filename := "html/" + page + ".html"
	body, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func NodeGroupsToJSON(w http.ResponseWriter, r *http.Request) {
	json, err := json.Marshal(Nodes)
	if err != nil {
		fmt.Println("Unable to marshal JSON when registering node Error:", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Println("Node Groups JSON: ", string(json))
	fmt.Fprintf(w, "%s", string(json))
}
