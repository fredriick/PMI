package peersdk

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"proxymesh/internal/models"
)

type PeerConfig struct {
	NodeID          string
	GatewayEndpoint string
	MTLSEnabled     bool
	MinBattery      int
	MaxCPU          float64
}

type NodeStatus struct {
	NodeID     string  `json:"node_id"`
	Online     bool    `json:"online"`
	Battery    int     `json:"battery"`
	IsCharging bool    `json:"is_charging"`
	CPUUsage   float64 `json:"cpu_usage"`
	IP         string  `json:"ip"`
}

type PeerSDK struct {
	config      *PeerConfig
	node        *models.Node
	ctx         context.Context
	cancel      context.CancelFunc
	isConnected bool
	mu          sync.RWMutex
	statusCh    chan NodeStatus
	reconnCh    chan struct{}
	client      *PeerClient
	lastBW      BandwidthSnapshot
	bwCh        chan BandwidthSnapshot
}

type BandwidthSnapshot struct {
	BytesSent       int64
	BytesReceived   int64
	DurationSeconds int64
}

func NewPeerSDK(config *PeerConfig) *PeerSDK {
	ctx, cancel := context.WithCancel(context.Background())
	return &PeerSDK{
		config:   config,
		ctx:      ctx,
		cancel:   cancel,
		statusCh: make(chan NodeStatus, 10),
		reconnCh: make(chan struct{}, 1),
		bwCh:     make(chan BandwidthSnapshot, 10),
	}
}

func (p *PeerSDK) Start() error {
	if !p.checkEligibility() {
		return fmt.Errorf("node does not meet eligibility requirements")
	}

	p.node = &models.Node{
		ID:         p.config.NodeID,
		NodeType:   models.NodeTypeResidential,
		Country:    "US",
		City:       "Unknown",
		ISP:        "Residential",
		IP:         p.getLocalIP(),
		OS:         "linux",
		LastSeen:   time.Now(),
		Reputation: 100.0,
	}

	if p.config.GatewayEndpoint != "" {
		client, err := NewPeerClient(p.config)
		if err != nil {
			return fmt.Errorf("failed to create peer client: %w", err)
		}
		p.client = client

		if err := client.Connect(); err != nil {
			client.Stop()
			p.client = nil
			return fmt.Errorf("failed to connect to gateway: %w", err)
		}

		go p.startTelemetryLoop()
	}

	p.mu.Lock()
	p.isConnected = true
	p.mu.Unlock()

	p.startHealthMonitor()
	p.startReconnector()

	return nil
}

func (p *PeerSDK) startTelemetryLoop() {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-p.ctx.Done():
				return
			case bw := <-p.bwCh:
				if p.client != nil {
					if err := p.client.ReportBandwidth(bw.BytesSent, bw.BytesReceived, bw.DurationSeconds); err != nil {
						log.Printf("Bandwidth report error: %v", err)
					}
				}
			case <-ticker.C:
				if p.client != nil {
					battery := p.getBatteryLevel()
					cpu := p.getCPUUsage()
					charging := p.isCharging()

					if err := p.client.Heartbeat(battery, cpu, charging, 0, 0, 0); err != nil {
						log.Printf("Heartbeat error: %v", err)
						p.TriggerReconnect()
					}
				}
			}
		}
	}()
}

func (p *PeerSDK) ReportBandwidth(bytesSent, bytesReceived, durationSeconds int64) {
	select {
	case p.bwCh <- BandwidthSnapshot{
		BytesSent:       bytesSent,
		BytesReceived:   bytesReceived,
		DurationSeconds: durationSeconds,
	}:
	default:
	}
}

func (p *PeerSDK) Stop() {
	if p.client != nil {
		p.client.Disconnect("user_disconnect")
		p.client.Stop()
	}
	p.cancel()
	p.mu.Lock()
	p.isConnected = false
	p.mu.Unlock()
}

func (p *PeerSDK) Disconnect() {
	p.Stop()
}

func (p *PeerSDK) TriggerReconnect() {
	select {
	case p.reconnCh <- struct{}{}:
	default:
	}
}

func (p *PeerSDK) startReconnector() {
	go func() {
		backoff := time.Second
		maxBackoff := 5 * time.Minute

		for {
			select {
			case <-p.ctx.Done():
				return
			case <-p.reconnCh:
				p.mu.RLock()
				connected := p.isConnected
				p.mu.RUnlock()

				if connected {
					continue
				}

				log.Printf("Attempting reconnection (backoff: %s)...", backoff)
				time.Sleep(backoff)

				if p.checkEligibility() {
					client, err := NewPeerClient(p.config)
					if err != nil {
						log.Printf("Reconnect failed to create client: %v", err)
						p.TriggerReconnect()
						continue
					}

					if err := client.Connect(); err != nil {
						client.Stop()
						log.Printf("Reconnect failed: %v", err)
						p.TriggerReconnect()
						continue
					}

					p.mu.Lock()
					p.client = client
					p.isConnected = true
					p.node.LastSeen = time.Now()
					p.mu.Unlock()

					log.Println("Reconnected successfully")
					backoff = time.Second
				} else {
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
					log.Printf("Reconnection failed (not eligible), next backoff: %s", backoff)
					p.TriggerReconnect()
				}
			}
		}
	}()
}

func (p *PeerSDK) checkEligibility() bool {
	if p.config.MinBattery > 0 {
		battery := p.getBatteryLevel()
		if battery < p.config.MinBattery && !p.isCharging() {
			return false
		}
	}

	if p.config.MaxCPU > 0 {
		cpu := p.getCPUUsage()
		if cpu > p.config.MaxCPU {
			return false
		}
	}

	if !p.isOnUnmeteredWiFi() {
		return false
	}

	return true
}

func (p *PeerSDK) startHealthMonitor() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-p.ctx.Done():
				return
			case <-ticker.C:
				p.reportStatus()
			}
		}
	}()
}

func (p *PeerSDK) reportStatus() {
	p.mu.RLock()
	connected := p.isConnected
	node := p.node
	p.mu.RUnlock()

	status := NodeStatus{
		NodeID:     node.ID,
		Online:     connected,
		Battery:    p.getBatteryLevel(),
		IsCharging: p.isCharging(),
		CPUUsage:   p.getCPUUsage(),
		IP:         node.IP,
	}

	select {
	case p.statusCh <- status:
	default:
	}
}

func (p *PeerSDK) GetStatus() NodeStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.node == nil {
		return NodeStatus{Online: false}
	}

	return NodeStatus{
		NodeID:     p.config.NodeID,
		Online:     p.isConnected,
		Battery:    p.getBatteryLevel(),
		IsCharging: p.isCharging(),
		CPUUsage:   p.getCPUUsage(),
		IP:         p.node.IP,
	}
}

func (p *PeerSDK) StatusChan() <-chan NodeStatus {
	return p.statusCh
}

func (p *PeerSDK) IsConnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isConnected
}

type ConsentManager struct {
	enabled   bool
	onEnable  func() error
	onDisable func() error
	mu        sync.RWMutex
}

func NewConsentManager(onEnable, onDisable func() error) *ConsentManager {
	return &ConsentManager{
		enabled:   false,
		onEnable:  onEnable,
		onDisable: onDisable,
	}
}

func (c *ConsentManager) Enable() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.enabled {
		return nil
	}

	if c.onEnable != nil {
		if err := c.onEnable(); err != nil {
			return err
		}
	}

	c.enabled = true
	return nil
}

func (c *ConsentManager) Disable() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.enabled {
		return nil
	}

	if c.onDisable != nil {
		if err := c.onDisable(); err != nil {
			return err
		}
	}

	c.enabled = false
	return nil
}

func (c *ConsentManager) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

func GenerateNodeID() string {
	return uuid.New().String()
}
