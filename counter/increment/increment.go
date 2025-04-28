package increment

import (
	"context"
	"discovery-service/models"
	pb "discovery-service/proto"
	"log"
	"time"
)

func PropagateIncrement(s *models.Server, opID string) {
	peers := append([]string{}, s.Peers...)

	for _, peer := range peers {
		if peer == s.Id {
			continue
		}

		func(p string) {
			conn := s.GetOrCreateConnection(peer)
			client := pb.NewDiscoveryClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			_, err := client.PropagateIncrement(ctx, &pb.IncrementRequest{Id: opID})
			if err != nil {
				log.Printf("Failed to propagate increment to %s: %v", p, err)
				queueMissedOp(s, p, opID) // <<< ADD THIS
			}
		}(peer)
	}
}

func queueMissedOp(s *models.Server, peer string, opID string) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if s.MissedOps == nil {
		s.MissedOps = make(map[string][]string)
	}
	s.MissedOps[peer] = append(s.MissedOps[peer], opID)
}
