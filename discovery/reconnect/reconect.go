package reconnect

import (
	"context"
	"discovery-service/models"
	"discovery-service/proto"
	"log"
)

type Reconnect struct {
}

func (r Reconnect) Execute(s *models.Server, peer string) {
	// Attempt to establish a connection to the peer
	conn := s.GetOrCreateConnection(peer)
	client := proto.NewDiscoveryClient(conn)

	// Send a heartbeat or any other message to verify the connection
	_, err := client.Heartbeat(context.Background(), &proto.HeartbeatRequest{Id: s.Id})
	if err != nil {
		log.Printf("failed to send heartbeat: %v", err)
	}

	// If successful, re-register with the peer and synchronize state
	_, err = client.Register(context.Background(), &proto.RegisterRequest{Id: s.Id})
	if err != nil {
		log.Printf("failed to register with peer: %v", err)
	}
}
