package web

import (
	"discovery-service/counter/increment"
	"discovery-service/models"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func StartHTTPServer(s *models.Server, grpcPort string) {
	httpPort := computeHTTPPort(grpcPort)
	mux := http.NewServeMux()
	mux.HandleFunc("/peers", func(w http.ResponseWriter, r *http.Request) {
		s.Mu.Lock()
		defer s.Mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string][]string{
			"peers": s.Peers,
		})
	})

	mux.HandleFunc("/increment", func(w http.ResponseWriter, r *http.Request) {
		opID := uuid.New().String()

		s.Mu.Lock()
		s.Counter++
		if s.SeenOps == nil {
			s.SeenOps = make(map[string]bool)
		}
		s.SeenOps[opID] = true
		s.Mu.Unlock()

		increment.PropagateIncrement(s, opID)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Counter incremented"))
	})

	mux.HandleFunc("/count", func(w http.ResponseWriter, r *http.Request) {
		s.Mu.Lock()
		count := s.Counter
		s.Mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int64{
			"count": count,
		})
	})

	go func() {
		log.Printf("HTTP server listening on %s", httpPort)
		if err := http.ListenAndServe(httpPort, mux); err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()
}

func computeHTTPPort(grpcPort string) string {
	p := strings.TrimPrefix(grpcPort, ":")
	portNum, err := strconv.Atoi(p)
	if err != nil {
		log.Fatalf("Invalid gRPC port: %s", grpcPort)
	}
	return fmt.Sprintf(":%d", portNum+1000)
}
