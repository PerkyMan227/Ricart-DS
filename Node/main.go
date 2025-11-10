package node

import (
	"os"
	"fmt"
	"sync"
	"time"
	"your_project/mutexpb"
)

type NodeState int
const (
    RELEASED NodeState = iota
    WANTED                  
    HELD                    
)
type Config struct {
    ID   int32  `json:"id"`
    Addr string `json:"addr"`
}

type Node struct {
    mutexpb.UnimplementedDistributedMutexServer

    id      int32
    state   NodeState
    clock   int64
    peers   map[int32]mutexpb.DistributedMutexClient
    
    requestTimestamp int64
    replyCount       int
    
    deferredQueue []*mutexpb.Request 

    mu sync.Mutex 
}

func main() {
	id := os.Args[0]
	
	jsonFile, err := os.Open("nodes.json")
	if err != nil {
		return
	}
	fmt.Println("Successfully Opened nodes.json")
	defer jsonFile.Close()
} 
