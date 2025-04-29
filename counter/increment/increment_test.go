package increment_test

import (
	"discovery-service/discovery/client"
	"discovery-service/models"
	"discovery-service/proto"
	"discovery-service/web"
	"encoding/json"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
	"testing"
	"time"
)

func startTestNode(t *testing.T, port string, initialPeers []string) {
	t.Helper()

	nodeID := "localhost:" + port
	s := &models.Server{Id: nodeID, Peers: []string{nodeID}}
	client.StartClient(s, initialPeers)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterDiscoveryServer(grpcServer, s)

	log.Printf("Node %s is running...", nodeID)
	web.StartHTTPServer(s, port)
	grpcServer.Serve(lis)
}

func TestRealIncrementPropagation(t *testing.T) {
	go startTestNode(t, "8080", []string{})
	go startTestNode(t, "8081", []string{"localhost:8080"})

	// Give time for discovery and setup
	time.Sleep(5 * time.Second)

	// Make an increment request on node1
	resp, err := http.Get("http://localhost:9080/increment")
	if err != nil {
		t.Fatalf("Failed to call increment API on node1: %v", err)
	}
	resp.Body.Close()

	resp, err = http.Get("http://localhost:9080/increment")
	if err != nil {
		t.Fatalf("Failed to call increment API on node1: %v", err)
	}
	resp.Body.Close()

	resp, err = http.Get("http://localhost:9080/increment")
	if err != nil {
		t.Fatalf("Failed to call increment API on node1: %v", err)
	}
	resp.Body.Close()

	resp, err = http.Get("http://localhost:9081/count")
	if err != nil {
		t.Fatalf("Failed to call count API on node2: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Count int64 `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode count response: %v", err)
	}

	if result.Count != 3 {
		t.Fatalf("Expected node2 count to be 3, got %d", result.Count)
	}
}

func TestParallelIncrementPropagation(t *testing.T) {
	go startTestNode(t, "8080", []string{})
	go startTestNode(t, "8081", []string{"localhost:8080"})

	// Give time for discovery and setup
	time.Sleep(5 * time.Second)

	// Number of parallel increments you want
	const numIncrements = 50

	// Fire parallel increment requests to node1
	done := make(chan struct{})
	for i := 0; i < numIncrements; i++ {
		go func() {
			resp, err := http.Get("http://localhost:9080/increment")
			if err != nil {
				t.Errorf("Failed to call increment API: %v", err)
				done <- struct{}{}
				return
			}
			resp.Body.Close()
			done <- struct{}{}
		}()
	}

	// Wait for all increments to finish
	for i := 0; i < numIncrements; i++ {
		<-done
	}

	// Give a little extra time for propagation
	time.Sleep(500 * time.Millisecond)

	// Now check the count on node2
	resp, err := http.Get("http://localhost:9081/count")
	if err != nil {
		t.Fatalf("Failed to call count API on node2: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Count int64 `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode count response: %v", err)
	}

	if result.Count != numIncrements {
		t.Fatalf("Expected node2 count to be %d, got %d", numIncrements, result.Count)
	}
}
