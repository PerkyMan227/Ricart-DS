package Node

import (
	"context"
	"log"
	"sync"
	"time"

	pro "ricart/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type deferredRequest struct {
	nodeId  int32
	replyCh chan *pro.Reply
}

type Node struct {
	id            int32
	address       string
	clock         int64                   // Lamport logical clock
	state         NodeState               // RELEASED, WANTED, HELD
	requestTime   int64                   // Timestamp when CS was requested
	peers         map[int32]string        // Other nodes' addresses
	replies       []int32                 // Nodes that sent OK
	deferredQueue []*deferredRequest      // Requests to reply to later
	mu            sync.Mutex              // Protects shared state
	pro.UnimplementedMutualExclusionServer
}

func NewNode(id int32, address string, peers map[int32]string) *Node {
	return &Node{
		id:            id,
		address:       address,
		clock:         0,
		state:         RELEASED,
		peers:         peers,
		replies:       []int32{},
		deferredQueue: []*deferredRequest{},
	}
}

// ID returns the node's ID (exported getter)
func (n *Node) ID() int32 {
	return n.id
}

// Address returns the node's address (exported getter)
func (n *Node) Address() string {
	return n.address
}

type NodeState int

const (
	RELEASED NodeState = iota
	WANTED
	HELD
)

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

	needToReply := false

	if n.state == RELEASED {
		needToReply = true
	} else if n.state == WANTED {
		// Check priority: lower timestamp wins and ties broken by lower node ID
		if req.Timestamp < n.requestTime {
			needToReply = true
		} else if (req.Timestamp == n.requestTime) && (req.NodeId < n.id) {
			needToReply = true
		} else {
			// We have higher priority. Defer the request by blocking until we reply later
			needToReply = false
		}
	} else { // State = HELD. Defer the request
		needToReply = false
	}

	if needToReply {
		n.clock++
		reply := &pro.Reply{NodeId: n.id, Timestamp: n.clock}
		n.mu.Unlock()
		return reply, nil
	}

	// Defer the request - create a channel and block until we're ready to reply
	replyCh := make(chan *pro.Reply, 1)
	n.deferredQueue = append(n.deferredQueue, &deferredRequest{
		nodeId:  req.NodeId,
		replyCh: replyCh,
	})
	n.mu.Unlock()

	// Block and wait for the reply to be sent when we exit CS
	reply := <-replyCh
	return reply, nil
}
func (n *Node) RequestCS() {

	//Change state to wanted
	n.mu.Lock()
	n.state = WANTED
	n.clock++
	n.requestTime = n.clock
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

	//Wait for replies
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

	log.Printf("[Node %d] - ENTERING Critical section", n.id)

}

func (n *Node) ReleaseCS() {
	n.mu.Lock()
	log.Printf("[Node %d] - LEAVING Critical section", n.id)
	n.state = RELEASED

	// Send replies to all deferred requests
	for _, req := range n.deferredQueue {
		n.clock++
		reply := &pro.Reply{NodeId: n.id, Timestamp: n.clock}
		req.replyCh <- reply
		log.Printf("[Node %d] - Sent deferred OK to Node %d", n.id, req.nodeId)
	}

	n.deferredQueue = []*deferredRequest{}
	n.mu.Unlock()
}

func (n *Node) sendRequest(peerID int32, peerAddr string) {

	log.Printf("[Node %d] - Attempting to send request to Node %d at %s", n.id, peerID, peerAddr)

	creds := insecure.NewCredentials()
	conn, err := grpc.NewClient(peerAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Printf("[Node %d] - Failed to connect to Node %d: %v", n.id, peerID, err)
		return
	}
	defer conn.Close()

	client := pro.NewMutualExclusionClient(conn)

	n.mu.Lock()
	timestamp := n.requestTime
	n.mu.Unlock()

	// Send request once and wait for reply (peer will reply later if deferred)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reply, err := client.RequestAccess(ctx, &pro.Request{NodeId: n.id, Timestamp: timestamp})

	if err == nil && reply != nil {
		n.mu.Lock()
		n.replies = append(n.replies, reply.NodeId)
		n.mu.Unlock()
		log.Printf("[Node %d] - Received OK from Node %d", n.id, peerID)
	} else {
		log.Printf("[Node %d] - Error receiving reply from Node %d: %v", n.id, peerID, err)
	}
}

// EnterCriticalSection simulates work in the critical section, whatever that may be...
func (n *Node) EnterCriticalSection() {
	log.Printf("========================================================")
	log.Printf("[Node %d] ====== PERFORMING CRITICAL SECTION WORK ======", n.id)
	time.Sleep(2 * time.Second) // Simulate some work
	log.Printf("[Node %d] ======  CRITICAL SECTION WORK COMPLETE  ======", n.id)
	log.Printf("========================================================")
}
