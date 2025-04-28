package main

import (
	"discovery-service/discovery/client"
	"discovery-service/models"
	"discovery-service/proto"
	"discovery-service/web/web"
	"flag"
	"google.golang.org/grpc"
	"log"
	"net"
	"strings"
)

func main() {
	port := flag.String("port", "8080", "port to listen on")
	peers := flag.String("peers", "", "comma-separated list of initial peers")
	flag.Parse()

	nodeID := "localhost:" + *port
	initialPeers := strings.Split(*peers, ",")

	s := &models.Server{Id: nodeID, Peers: []string{nodeID}}
	client.StartClient(s, initialPeers)

	lis, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterDiscoveryServer(grpcServer, s)

	log.Printf("Node %s is running...", nodeID)
	web.StartHTTPServer(s, *port)
	grpcServer.Serve(lis)
}
