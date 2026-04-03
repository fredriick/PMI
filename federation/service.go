package federation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"proxymesh/internal/config"
	"proxymesh/internal/models"
	"proxymesh/matchmaker"
)

type FederationService struct {
	cfg       config.FederationConfig
	mm        *matchmaker.Matchmaker
	peers     map[string]*PeerGateway
	mu        sync.RWMutex
	nodeCache map[string][]string
	stop      chan bool
}

type PeerGateway struct {
	Region   string
	Address  string
	LastSeen time.Time
	Healthy  bool
}

type FederationStats struct {
	Region    string    `json:"region"`
	PeerCount int       `json:"peer_count"`
	NodeCount int       `json:"node_count"`
	LastSync  time.Time `json:"last_sync"`
	Status    string    `json:"status"`
}

func NewFederationService(cfg *config.Config, mm *matchmaker.Matchmaker) *FederationService {
	return &FederationService{
		cfg:       cfg.Federation,
		mm:        mm,
		peers:     make(map[string]*PeerGateway),
		nodeCache: make(map[string][]string),
		stop:      make(chan bool),
	}
}

func (f *FederationService) Start() error {
	if !f.cfg.Enabled {
		log.Printf("Federation disabled")
		return nil
	}

	log.Printf("Starting federation service with regions: %v", f.cfg.Regions)

	go f.heartbeatLoop()
	go f.syncLoop()

	return nil
}

func (f *FederationService) Stop() {
	close(f.stop)
}

func (f *FederationService) heartbeatLoop() {
	ticker := time.NewTicker(time.Duration(f.cfg.HeartbeatInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			f.checkPeerHealth()
		case <-f.stop:
			return
		}
	}
}

func (f *FederationService) syncLoop() {
	ticker := time.NewTicker(time.Duration(f.cfg.SyncInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			f.syncNodes()
		case <-f.stop:
			return
		}
	}
}

func (f *FederationService) checkPeerHealth() {
	f.mu.RLock()
	defer f.mu.RUnlock()

	now := time.Now()
	for region, peer := range f.peers {
		if now.Sub(peer.LastSeen) > time.Duration(f.cfg.HeartbeatInterval*3)*time.Second {
			peer.Healthy = false
			log.Printf("Federation: peer %s marked unhealthy", region)
		}
	}
}

func (f *FederationService) syncNodes() {
	nodes, err := f.mm.GetAllNodes()
	if err != nil {
		log.Printf("Federation: failed to get nodes: %v", err)
		return
	}

	f.mu.Lock()
	f.nodeCache["local"] = nodes
	f.mu.Unlock()

	log.Printf("Federation: synced %d nodes", len(nodes))
}

func (f *FederationService) RegisterPeer(region, address string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.peers[region] = &PeerGateway{
		Region:   region,
		Address:  address,
		LastSeen: time.Now(),
		Healthy:  true,
	}

	log.Printf("Federation: registered peer %s at %s", region, address)
	return nil
}

func (f *FederationService) GetStats() []FederationStats {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var stats []FederationStats

	for region, peer := range f.peers {
		nodeCount := 0
		if cached, ok := f.nodeCache[region]; ok {
			nodeCount = len(cached)
		}

		status := "healthy"
		if !peer.Healthy {
			status = "unhealthy"
		}

		stats = append(stats, FederationStats{
			Region:    region,
			PeerCount: 1,
			NodeCount: nodeCount,
			LastSync:  peer.LastSeen,
			Status:    status,
		})
	}

	return stats
}

func (f *FederationService) GetNodesForRegion(region string) ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	nodes, ok := f.nodeCache[region]
	if !ok {
		return nil, fmt.Errorf("no nodes found for region: %s", region)
	}

	return nodes, nil
}

func (f *FederationService) SelectNodeByRegion(preferredRegion string) (*models.Node, error) {
	if preferredRegion != "" {
		nodes, err := f.GetNodesForRegion(preferredRegion)
		if err == nil && len(nodes) > 0 {
			return f.mm.GetNodeStatus(nodes[0])
		}
	}

	nodes, err := f.mm.GetAllNodes()
	if err != nil || len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}

	return f.mm.GetNodeStatus(nodes[0])
}

type FederationMessage struct {
	Type      string          `json:"type"`
	Region    string          `json:"region"`
	Nodes     []string        `json:"nodes,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data,omitempty"`
}

func (f *FederationService) HandleMessage(ctx context.Context, msg *FederationMessage) error {
	switch msg.Type {
	case "heartbeat":
		return f.handleHeartbeat(msg)
	case "node_sync":
		return f.handleNodeSync(msg)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

func (f *FederationService) handleHeartbeat(msg *FederationMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.peers[msg.Region]; !ok {
		f.peers[msg.Region] = &PeerGateway{Region: msg.Region}
	}
	f.peers[msg.Region].LastSeen = time.Now()
	f.peers[msg.Region].Healthy = true

	return nil
}

func (f *FederationService) handleNodeSync(msg *FederationMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.nodeCache[msg.Region] = msg.Nodes
	log.Printf("Federation: received %d nodes from %s", len(msg.Nodes), msg.Region)

	return nil
}
