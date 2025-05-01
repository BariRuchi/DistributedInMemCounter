package models

import (
	"context"
	"discovery-service/lib/arrays"
	pb "discovery-service/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"sync"
	"time"
)

type Server struct {
	pb.UnimplementedDiscoveryServer
	Id            string
	Peers         []string
	DeadPeers     []string
	Mu            sync.Mutex
	Counter       int64
	MissedOps     map[string][]string // new field
	Partitioned   bool
	SeenOps       map[string]bool             // For deduplication
	ConnPool      map[string]*grpc.ClientConn // Pool for active peer connections
	IncrementChan chan string
}

func (s *Server) GetOrCreateConnection(peer string) *grpc.ClientConn {
	existingConn, exists := s.ConnPool[peer]

	// Return the existing connection if it's available
	if exists && existingConn != nil {
		return existingConn
	}
	// If no connection exists, try to create a new one
	var conn *grpc.ClientConn
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		conn, err = grpc.NewClient(peer, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			s.Mu.Lock()
			s.ConnPool[peer] = conn
			s.Mu.Unlock()
			return conn
		}

		log.Printf("Failed to connect to %s (attempt %d): %v", peer, attempt+1, err)
		time.Sleep(time.Second * time.Duration(attempt+1))
	}

	log.Printf("Unable to establish connection to %s after %d retries", peer, 5)
	return nil
}

// Register handles peer registration and returns the updated peer list.
func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	log.Printf("Registering peer: %s", req.Id)

	// Add peer if not already present
	if !arrays.Contains(s.Peers, req.Id) {
		s.Peers = append(s.Peers, req.Id)
	}

	// Return the updated list of peers
	return &pb.RegisterResponse{Peers: append(s.Peers, s.DeadPeers...)}, nil
}

// GetPeers returns the list of peers.
func (s *Server) GetPeers(ctx context.Context, _ *pb.Empty) (*pb.PeersResponse, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	return &pb.PeersResponse{Peers: append(s.Peers, s.DeadPeers...)}, nil
}

// Heartbeat checks if the peer is alive.
func (s *Server) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	log.Printf("Received heartbeat from %s", req.Id)
	return &pb.HeartbeatResponse{Alive: true}, nil
}

func (s *Server) PropagateIncrement(ctx context.Context, req *pb.IncrementRequest) (*pb.IncrementResponse, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	if s.SeenOps == nil {
		s.SeenOps = make(map[string]bool)
	}

	if s.SeenOps[req.Id] {
		return &pb.IncrementResponse{Success: true}, nil
	}

	s.IncrementChan <- req.Id
	//s.Counter++
	//s.SeenOps[req.Id] = true

	log.Printf("Counter incremented via propagation: %d", s.Counter)
	return &pb.IncrementResponse{Success: true}, nil
}

func (s *Server) GetCounter(ctx context.Context, _ *pb.Empty) (*pb.CounterResponse, error) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	return &pb.CounterResponse{Counter: s.Counter}, nil
}

func NewServer(nodeId string) *Server {
	s := new(Server)
	s.Id = nodeId
	s.Peers = []string{nodeId}
	s.SeenOps = make(map[string]bool)
	s.IncrementChan = make(chan string)
	return s
}
