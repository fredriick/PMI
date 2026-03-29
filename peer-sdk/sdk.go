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

type PeerSDK struct {
	config      *PeerConfig
	node        *models.Node
	ctx         context.Context
	cancel      context.CancelFunc
	isConnected bool
	mu          sync.RWMutex
	statusCh    chan NodeStatus
	reconnCh    chan struct{}
}

type NodeStatus struct {
	NodeID     string  `json:"node_id"`
	Online     bool    `json:"online"`
	Battery    int     `json:"battery"`
	IsCharging bool    `json:"is_charging"`
	CPUUsage   float64 `json:"cpu_usage"`
	IP         string  `json:"ip"`
}

func NewPeerSDK(config *PeerConfig) *PeerSDK {
	ctx, cancel := context.WithCancel(context.Background())
	return &PeerSDK{
		config:   config,
		ctx:      ctx,
		cancel:   cancel,
		statusCh: make(chan NodeStatus, 10),
		reconnCh: make(chan struct{}, 1),
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
		City:       "New York",
		ISP:        "Residential ISP",
		IP:         p.getLocalIP(),
		OS:         "linux",
		LastSeen:   time.Now(),
		Reputation: 100.0,
	}

	p.isConnected = true
	p.startHealthMonitor()
	p.startReconnector()

	return nil
}

func (p *PeerSDK) Disconnect() {
	p.mu.Lock()
	p.isConnected = false
	p.mu.Unlock()
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
					p.mu.Lock()
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
					log.Printf("Reconnection failed, next backoff: %s", backoff)
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
	status := NodeStatus{
		NodeID:     p.node.ID,
		Online:     p.isConnected,
		Battery:    p.getBatteryLevel(),
		IsCharging: p.isCharging(),
		CPUUsage:   p.getCPUUsage(),
		IP:         p.node.IP,
	}

	select {
	case p.statusCh <- status:
	default:
	}
}

func (p *PeerSDK) Stop() {
	p.cancel()
	p.isConnected = false
}

func (p *PeerSDK) GetStatus() NodeStatus {
	return NodeStatus{
		NodeID:     p.node.ID,
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
