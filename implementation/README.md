# Distributed Mutual Exclusion (Ricart-Agrawala)
Ricart-agawala implementation in gRPC for mandatory activity 4
The system tarts a cluster of three nodes that use gRPC to communicate and corrdinate access to a shared critical section

## Prerequisites 
Go: Version 1.20.x or later

## Running the system

To run the system you must open 3 sepperate terminal windows/tabs and run a command in each

The program start one node at a time using the command `go run main.go` with two flags:
`-id`: Hardcoded unique ID of the node (1, 2, or 3)
`-port`: the port that nodes gRPC ser will listen on

Terminal 1:

```bash
go run main.go -id=1 -port=50051
```


Terminal 2:

```bash
go run main.go -id=2 -port=50052
```

Terminal 3:

```bash
go run main.go -id=3 -port=50053
```


## How it works

1. You must start all three processes manually as shown above.
2. The program hardcodes the addresses and ports in the source code for `main.go` Each flag must be identical to whats shown above.
3. After 3 sec startup delay, each node ill independently start requesting access to the critical state
4. You can observe the interaction between the node in the log output in each terminal window/tab

