package resend

import (
	"context"
	"discovery-service/lib/arrays"
	"discovery-service/models"
	"discovery-service/proto"
	"log"
	"time"
)

type Resend struct {
}

func (r Resend) Execute(s *models.Server, peer string) {
	s.Mu.Lock()
	opIDs := append([]string{}, s.MissedOps[peer]...)
	s.Mu.Unlock()

	if len(opIDs) == 0 {
		return
	}

	log.Printf("Resending %d missed ops to %s", len(opIDs), peer)

	conn := s.GetOrCreateConnection(peer)
	client := proto.NewDiscoveryClient(conn)

	for _, opID := range opIDs {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := client.PropagateIncrement(ctx, &proto.IncrementRequest{Id: opID})
		cancel()

		if err != nil {
			log.Printf("Failed to resend opID %s to %s: %v", opID, peer, err)
			continue
		}

		// On success, remove opID
		s.Mu.Lock()
		s.MissedOps[peer] = arrays.Remove(s.MissedOps[peer], opID)
		s.Mu.Unlock()
	}
}
