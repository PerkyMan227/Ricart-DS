package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	node "ricart/Node" // <--- This line
	pro "ricart/proto" // <--- And this line

	"google.golang.org/grpc"
)

func main() {

	nodeID := flag.Int("id", 0, "Node ID")
	port := flag.Int("port", 0, "Port to listen on")
	flag.Parse()

	allNodes := map[int32]string{
		1: "localhost:50051",
		2: "localhost:50052",
		3: "localhost:50053",
	}

	address := allNodes[int32(*nodeID)]

	peers := make(map[int32]string)
	for id, addr := range allNodes {
		if id != int32(*nodeID) {
			peers[id] = addr
		}
	}

	n := node.NewNode(int32(*nodeID), address, peers)

	go startServer(n, *port)

	// Give time for all nodes to start
	log.Printf("[Node %d] - Waiting for other nodes to start...", *nodeID)
	time.Sleep(3 * time.Second)

	// Simulate random requests to critical section
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 3; i++ {
		// Random delay before requesting CS
		delay := time.Duration(rand.Intn(5)+1) * time.Second
		log.Printf("[Node %d] - Waiting %v before requesting CS", *nodeID, delay)
		time.Sleep(delay)

		// Request access to critical section
		n.RequestCS()

		// Enter and work in critical section
		n.EnterCriticalSection()

		// Release critical section
		n.ReleaseCS()
	}

	log.Printf("[Node %d] - All CS requests complete. Keeping server alive...", *nodeID)
	select {} // Keep the program running

}

func startServer(n *node.Node, port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Printf("!!! FATAL: [Node %d] - Failed to listen on port %d: %v", n.ID(), port, err)
		return
	}

	grpcServer := grpc.NewServer()
	pro.RegisterMutualExclusionServer(grpcServer, n)

	log.Printf("[Node %d] - gRPC server listening on port %d", n.ID(), port)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("[Node %d] - Failed to serve: %v", n.ID(), err)
	}

}
