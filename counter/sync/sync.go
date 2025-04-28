package sync

import (
	"context"
	"discovery-service/models"
	"discovery-service/proto"
	"log"
	"time"
)

func SyncCounterFromPeer(s *models.Server, client proto.DiscoveryClient, peer string) {
	log.Printf("Syncing counter from peer: %s", peer)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	resp, err := client.GetCounter(ctx, &proto.Empty{})
	cancel()

	if err != nil {
		log.Printf("Failed to get counter from %s: %v", peer, err)
		return
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()

	// Take the maximum of local counter and the counter from the peer
	if resp.Counter > s.Counter {
		s.Counter = resp.Counter
		log.Printf("Updated counter to %d after syncing with %s", s.Counter, peer)
	}
}
