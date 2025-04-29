package client

import (
	"context"
	"discovery-service/counter/resend"
	"discovery-service/counter/sync"
	"discovery-service/discovery/heartbeat"
	"discovery-service/discovery/reconnect"
	"discovery-service/lib/arrays"
	"discovery-service/models"
	"discovery-service/proto"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
)

func StartClient(s *models.Server, initialPeers []string) {
	visited := map[string]bool{}
	s.ConnPool = map[string]*grpc.ClientConn{}
	var connectAndRegister func(addr string)
	connectAndRegister = func(addr string) {
		if addr == s.Id || visited[addr] || addr == "" {
			return
		}
		visited[addr] = true

		// Corrected the connection creation using grpc.Dial
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Println("Could not connect to peer:", addr)
			return
		}
		defer conn.Close()

		client := proto.NewDiscoveryClient(conn)

		// Register with the peer
		resp, err := client.Register(context.Background(), &proto.RegisterRequest{Id: s.Id})
		if err != nil {
			fmt.Println("Error registering:", err)
			return
		}

		// Sync counter with all known peers
		for _, p := range resp.Peers {
			if !arrays.Contains(s.Peers, p) {
				sync.SyncCounterFromPeer(s, client, p)
			}
		}

		// Lock and update the peer list
		s.Mu.Lock()
		s.Peers = arrays.AppendUnique(s.Peers, addr)
		s.Peers = arrays.AppendUnique(s.Peers, resp.Peers...)
		s.Mu.Unlock()

		// Recursively register with discovered peers
		for _, p := range resp.Peers {
			connectAndRegister(p)
		}
	}

	// Start connecting to initial peers
	for _, addr := range initialPeers {
		connectAndRegister(addr)
	}

	// Register recovery actions for heartbeat
	heartbeat.RegisterRecoveryAction(reconnect.Reconnect{})
	heartbeat.RegisterRecoveryAction(resend.Resend{})
	heartbeat.MonitorHeartbeats(s)
}
