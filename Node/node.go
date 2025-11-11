package Node

import (
	"context"
	"log"
	"sync"
	"time"

	pro "ricart/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Node struct {
	id            int32
	address       string
	clock         int64            // Lamport logical clock
	state         NodeState        // RELEASED, WANTED, HELD
	requestTime   int64            // Timestamp when CS was requested
	peers         map[int32]string // Other nodes' addresses
	replies       []int32          // Nodes that sent OK
	deferredQueue []int32          // Requests to reply to later
	mu            sync.Mutex       // Protects shared state
}

type NodeState int

const (
	RELEASED NodeState = iota
	WANTED
	HELD
)

// This for incrementing own clock
func (n *Node) incrementClock() int64 {
	n.mu.Lock()
	n.clock++
	n.mu.Unlock()
	return n.clock
}

// This for updating own clock with recieved clock from other proccess + 1
func (n *Node) updateClock(recievedTime int64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.clock = max(n.clock, recievedTime) + 1
}

// Always increment clock before sending messages, update clock to max(localClock, recievedClock) + 1

func (n *Node) RequestAccess(ctx context.Context, req *pro.Request) (*pro.Reply, error) {
	n.updateClock(req.Timestamp)

	n.mu.Lock()
	defer n.mu.Unlock()

	needToReply := false

	if n.state == RELEASED {
		needToReply = true
	} else if n.state == WANTED {
		// Check priority
		if req.Timestamp < n.requestTime {
			needToReply = true
		} else if (req.Timestamp == n.requestTime) && (req.NodeId < n.id) {
			needToReply = true
		} else {
			// We have higher priority. We defer the request by putting the request in the queue
			n.deferredQueue = append(n.deferredQueue, req.NodeId)
		}
	} else { // State = HELD. Same idea
		n.deferredQueue = append(n.deferredQueue, req.NodeId)
	}

	if needToReply {
		return &pro.Reply{NodeId: n.id, Timestamp: n.clock}, nil
	}

	return nil, status.Error(codes.Unavailable, "Request deffered")

}

func (n *Node) RequestCS() {

	//Change state to wanted
	n.mu.Lock()
	n.state = WANTED
	n.requestTime = n.incrementClock()
	n.replies = []int32{}
	n.mu.Unlock()

	log.Printf("[Node %d] - Requesting to access CS with timestamp: %d", n.id, n.requestTime)

	//Sending request to all nodes

	var wg sync.WaitGroup
	for peerID, peerAddress := range n.peers {
		wg.Add(1)
		go func(id int32, addr string) {
			defer wg.Done()
			n.sendRequest(id, addr)
		}(peerID, peerAddress)
	}

	for {
		n.mu.Lock()
		if len(n.replies) == len(n.peers) {
			n.state = HELD
			n.mu.Unlock()
			break
		}
		n.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}

	log.Printf("[Node %d] - ENTERING Critical State", n.id)

}

func (n *Node) sendRequest(peerID int32, peerAddr string) {
	creds := insecure.NewCredentials()
	conn, err := grpc.NewClient(peerAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return
	}

	client := pro.NewMutualExclusionClient(conn)

	n.mu.Lock()
	timestamp := n.requestTime
	n.mu.Unlock()

	reply, err := client.RequestAccess(context.Background(), &pro.Request{NodeId: n.id, Timestamp: timestamp})

	if err == nil {
		n.mu.Lock()
		n.replies = append(n.replies, reply.NodeId)
		n.mu.Unlock()
		log.Printf("[Node %d] - Recived OK from Node %d", n.id, peerID)
	}
}

func (n *Node) sendDeferredReply(nodeID int32) {
	// Change state to RELEASED
	// Send OK to all nodes in deferred queue
	// Clear deffered queeue
	// How you may ask. Fuck if i know...
}
