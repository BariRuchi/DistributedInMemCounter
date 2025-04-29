# Distributed In Mem Counter

## Overview
This project implements a basic distributed counter system using gRPC for communication and custom service discovery. It ensures eventual consistency through peer-to-peer increment propagation, deduplication logic, retry mechanisms, and heartbeat monitoring.

---

## Design Decisions

- **Service Discovery**:
    - Each node maintains a dynamic list of peers (`Server.Peers`).
    - Heartbeat mechanism (`heartbeat.MonitorHeartbeats`) monitors peer liveness.
    - Nodes dynamically remove dead peers and re-add recovered nodes.

- **Eventual Consistency**:
    - Increments are propagated to peers.
    - Duplicate operations are ignored through deduplication.
    - Retries with exponential backoff ensure missed updates eventually succeed.

- **Heartbeat and Failures**:
    - Regular heartbeat checks mark nodes as dead/alive.
    - Connection pooling optimizes peer communication.

---

## How to Run Nodes

**GRPC file generate command**
- Install `protoc` (Protocol Buffers Compiler)

  **On MacOS**:
  ```bash
  brew install protobuf
  
- Install Go plugins for protoc
  ```bash
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

- To generate Go code from the .proto file, run:

  ```bash 
  protoc --go_out=. --go-grpc_out=. discovery.proto


1. **Start Multiple Nodes**

```bash
go run main.go --port=5001 --peers=localhost:5002,localhost:5003
go run main.go --port=5002 --peers=localhost:5001,localhost:5003
go run main.go --port=5003 --peers=localhost:5001,localhost:5002
```

2. **Send Increment Requests**

You can simulate increments via gRPC or CLI commands if implemented.

3. **Run Tests**

```bash
cd counter/increment

go test -v
```

Tests cover:
- Concurrent increments on one node
- Increment propagation
- Deduplication and retry handling
- Cluster rebalancing (join/leave)

---

## Questions

### How does your system handle network partitions?
- During a network partition, nodes failing heartbeats are marked dead and removed.
- Once the partition heals, heartbeats succeed again, and nodes are re-added.
- Missed increment operations can retry after reconnection.

### What are the limitations of your design?
- **Strong consistency is not guaranteed**: The system achieves *eventual* consistency but temporary divergence is possible.
- **Partition detection is heartbeat-based**: False positives (temporary lag or network jitter) can cause unnecessary peer removal.
- **Single point retries**: Retries are initiated by the sender only; missed operations may require additional sync.
- **No external distributed consensus** (like Raft/Paxos) is used — lightweight but weaker guarantees.

---

## Directory Structure

```
/discovery/heartbeat  # Heartbeat monitoring
/counter/increment    # Counter operations
/counter/sync         # Synchronization logic
/counter/resend       # Retry handling
/models/server.go     # Server and peer state
/proto                # gRPC definitions
```

---
