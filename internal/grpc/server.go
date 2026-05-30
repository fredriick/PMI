package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"proxymesh/internal/config"
	"proxymesh/internal/models"
	"proxymesh/matchmaker"
	
)

type PeerServer struct {
	UnimplementedPeerServiceServer
	matchmaker *matchmaker.Matchmaker
	config     *config.PeerConfig
	sessions   map[string]*PeerSession
	mu         sync.RWMutex
}

type PeerSession struct {
	NodeID    string
	SessionID string
	Connected time.Time
	LastSeen  time.Time
}

func NewPeerServer(cfg *config.Config, mm *matchmaker.Matchmaker) *PeerServer {
	return &PeerServer{
		matchmaker: mm,
		config:     &cfg.Peer,
		sessions:   make(map[string]*PeerSession),
	}
}

func (s *PeerServer) Connect(ctx context.Context, req *ConnectRequest) (*ConnectResponse, error) {
	log.Printf("Peer connecting: node_id=%s, ip=%s, country=%s", req.NodeId, req.Ip, req.Country)

	if req.NodeId == "" {
		return &ConnectResponse{Success: false, Message: "node_id is required"}, nil
	}

// Eligibility validation (battery, CPU, WiFi) is performed via Heartbeat after connection.
// For now, assume eligible if node_id is present (validated above).

	registrationReq := &models.NodeRegistrationRequest{
		NodeID:     req.NodeId,
		NodeType:   "residential",
		Country:    req.Country,
		City:       req.City,
		ISP:        req.Isp,
		IP:         req.Ip,
		IPv6Subnet: req.Ipv6Subnet,
		OS:         req.Os,
	}

	if err := s.matchmaker.RegisterNode(registrationReq); err != nil {
		return &ConnectResponse{Success: false, Message: err.Error()}, nil
	}

	sessionID := fmt.Sprintf("%s-%d", req.NodeId, time.Now().Unix())
	s.mu.Lock()
	s.sessions[sessionID] = &PeerSession{
		NodeID:    req.NodeId,
		SessionID: sessionID,
		Connected: time.Now(),
		LastSeen:  time.Now(),
	}
	s.mu.Unlock()

	return &ConnectResponse{
		Success:   true,
		Message:   "Connected successfully",
		SessionId: sessionID,
	}, nil
}

func (s *PeerServer) Heartbeat(ctx context.Context, req *HeartbeatRequest) (*HeartbeatResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, session := range s.sessions {
		if session.NodeID == req.NodeId {
			session.LastSeen = time.Now()
			break
		}
	}

	heartbeatReq := &models.NodeHeartbeatRequest{
		NodeID:     req.NodeId,
		Battery:    int(req.Battery),
		CPUUsage:   float64(req.CpuUsage),
		IsCharging: req.IsCharging,
	}

	if err := s.matchmaker.Heartbeat(heartbeatReq); err != nil {
		return &HeartbeatResponse{Success: false, Message: err.Error()}, nil
	}

	return &HeartbeatResponse{Success: true, Message: "Heartbeat received"}, nil
}

func (s *PeerServer) ReportBandwidth(ctx context.Context, req *BandwidthReport) (*BandwidthResponse, error) {
	log.Printf("Bandwidth report from %s: sent=%d, received=%d, duration=%ds",
		req.NodeId, req.BytesSent, req.BytesReceived, req.DurationSeconds)

	s.matchmaker.RecordBandwidth(req.NodeId, req.BytesSent, req.BytesReceived, req.DurationSeconds)

	return &BandwidthResponse{Success: true, Message: "Bandwidth recorded"}, nil
}

func (s *PeerServer) Disconnect(ctx context.Context, req *DisconnectRequest) (*DisconnectResponse, error) {
	log.Printf("Peer disconnecting: node_id=%s, reason=%s", req.NodeId, req.Reason)

	s.mu.Lock()
	defer s.mu.Unlock()

	for sessionID, session := range s.sessions {
		if session.NodeID == req.NodeId {
			delete(s.sessions, sessionID)
			break
		}
	}

	s.matchmaker.DeregisterNode(req.NodeId)

	return &DisconnectResponse{Success: true}, nil
}

func (s *PeerServer) Start(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	RegisterPeerServiceServer(grpcServer, s)

	log.Printf("Starting gRPC server on port %d", port)
	return grpcServer.Serve(lis)
}

func (s *PeerServer) GetSession(nodeID string) *PeerSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, session := range s.sessions {
		if session.NodeID == nodeID {
			return session
		}
	}
	return nil
}

func (s *PeerServer) StreamTelemetry(stream PeerService_StreamTelemetryServer) error {
	// Receive the initial request from the client
	in, err := stream.Recv()
	if err != nil {
		return err
	}
	nodeID := in.NodeId
	log.Printf("Starting telemetry stream for node_id=%s", nodeID)

	// Send initial acknowledgment
	if err := stream.Send(&TelemetryStreamResponse{
		ServerTimestamp: time.Now().Format(time.RFC3339),
		StatusMessage:   "Telemetry stream started",
		ConnectionQuality: 100,
	}); err != nil {
		return err
	}

	// Stream telemetry data periodically
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			log.Printf("Telemetry stream stopped for node_id=%s: %v", nodeID, stream.Context().Err())
			return nil
		case <-ticker.C:
			// Simulate collecting telemetry data
			connectionQuality := 85 + (time.Now().Unix() % 15) // Vary between 85-100

			if err := stream.Send(&TelemetryStreamResponse{
				ServerTimestamp: time.Now().Format(time.RFC3339),
				StatusMessage:   fmt.Sprintf("Telemetry update for %s", nodeID),
				ConnectionQuality: int32(connectionQuality),
			}); err != nil {
				log.Printf("Failed to send telemetry to %s: %v", nodeID, err)
				return err
			}
		}
	}
}
