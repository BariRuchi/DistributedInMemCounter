package heartbeat

import (
	"context"
	"discovery-service/lib/arrays"
	"discovery-service/models"
	"discovery-service/proto"
	"google.golang.org/grpc"
	"log"
	"sync"
	"time"
)

const (
	MaxRetries      = 5
	BaseBackoff     = time.Second
	heartbeatPeriod = 5 * time.Second
)

var (
	mu              sync.Mutex
	peersState      = make(map[string]*peerState)
	recoveryActions []RecoveryAction
)

type peerState struct {
	failures int
	dead     bool
}

type RecoveryAction interface {
	Execute(s *models.Server, peer string)
}

func RegisterRecoveryAction(action RecoveryAction) {
	mu.Lock()
	defer mu.Unlock()
	recoveryActions = append(recoveryActions, action)
}

func MonitorHeartbeats(s *models.Server) {
	go func() {
		for {
			time.Sleep(heartbeatPeriod)
			// 1. Get a snapshot of current live peers
			currentPeers := append([]string{}, s.Peers...)
			currentPeers = append(currentPeers, s.DeadPeers...)

			// 2. Add dead peers to the list to be checked
			for peer, state := range peersState {
				if state.dead && !arrays.Contains(currentPeers, peer) {
					currentPeers = append(currentPeers, peer)
				}
			}
			// 3. Start heartbeat checks
			for _, peer := range currentPeers {
				if peer == s.Id {
					continue
				}

				_, ok := peersState[peer]
				if !ok {
					peersState[peer] = &peerState{}
				}
				checkHeartbeat(s, peer)
			}

		}
	}()
}

func checkHeartbeat(s *models.Server, peer string) {
	backoff := BaseBackoff
	success := false

	// Reuse existing connection if available
	conn := s.GetOrCreateConnection(peer)

	if conn == nil {
		log.Printf("Failed to establish connection to %s after %d retries", peer, MaxRetries)
		return
	}

	client := proto.NewDiscoveryClient(conn)

	// Retry heartbeat with exponential backoff
	for attempt := 0; attempt < MaxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := client.Heartbeat(ctx, &proto.HeartbeatRequest{Id: s.Id})
		cancel()

		if err == nil {
			success = true
			break
		}

		log.Printf("Heartbeat to %s failed (attempt %d): %v", peer, attempt+1, err)
		time.Sleep(backoff)
		backoff *= 2 // Exponential backoff
	}

	handleHeartbeatResult(s, peer, success, conn)
}

func handleHeartbeatResult(s *models.Server, peer string, success bool, conn *grpc.ClientConn) {
	state := peersState[peer]

	if success {
		log.Printf("Heartbeat to %s succeeded", peer)
		state.failures = 0
		if state.dead {
			log.Printf("Peer %s healed", peer)
			state.dead = false
			s.Mu.Lock()
			s.Peers = append(s.Peers, peer)
			s.DeadPeers = arrays.Remove(s.DeadPeers, peer)
			s.Mu.Unlock()
			// Execute all registered recovery actions
			for _, action := range recoveryActions {
				action.Execute(s, peer)
			}
		}
	} else {
		log.Printf("Peer %s failed heartbeat after %d attempts", peer, MaxRetries)
		state.failures++
		if !state.dead {
			log.Printf("Marking %s as dead", peer)
			state.dead = true
			s.Mu.Lock()
			s.Peers = arrays.Remove(s.Peers, peer)
			s.DeadPeers = append(s.DeadPeers, peer)
			s.Mu.Unlock()
			// Close the connection as the peer is dead
			closeConnection(s, peer, conn)
		}
	}
}

func closeConnection(s *models.Server, peer string, conn *grpc.ClientConn) {
	mu.Lock()
	if conn != nil {
		conn.Close()
		delete(s.ConnPool, peer)
	}
	mu.Unlock()
}
