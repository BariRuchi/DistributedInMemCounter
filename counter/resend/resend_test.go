package resend_test

import (
	resend2 "discovery-service/counter/resend"
	"discovery-service/discovery/client"
	"discovery-service/models"
	"discovery-service/proto"
	"discovery-service/web"
	"google.golang.org/grpc"
	"log"
	"net"
	"testing"
	"time"
)

func startTestNode(t *testing.T, port string, initialPeers []string) {
	t.Helper()

	nodeID := "localhost:" + port
	s := models.NewServer(nodeID)
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

func TestMissedOperationsResentSuccessfully(t *testing.T) {
	go startTestNode(t, "8082", []string{})
	go startTestNode(t, "8083", []string{"localhost:8082"})

	// Wait for servers to come up
	time.Sleep(5 * time.Second)

	// Simulate missed operations
	server := &models.Server{
		Id:        "localhost:8084",
		Peers:     []string{"localhost:8083"},
		MissedOps: make(map[string][]string),
	}

	// Simulate a few missed operation IDs
	server.MissedOps["localhost:8083"] = []string{"op1", "op2", "op3"}

	// Create connection to peer manually
	conn, err := grpc.Dial("localhost:8083", grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(2*time.Second))
	if err != nil {
		t.Fatalf("Failed to connect to peer: %v", err)
	}
	defer conn.Close()

	server.Mu.Lock()
	server.ConnPool = map[string]*grpc.ClientConn{
		"localhost:8083": conn,
	}
	server.Mu.Unlock()

	// Now execute resend
	resend := resend2.Resend{}
	resend.Execute(server, "localhost:8083")

	// Check that MissedOps is now empty
	server.Mu.Lock()
	defer server.Mu.Unlock()

	if len(server.MissedOps["localhost:8083"]) != 0 {
		t.Fatalf("Expected missed operations to be empty after successful resend, got: %v", server.MissedOps["localhost:8083"])
	}
}
