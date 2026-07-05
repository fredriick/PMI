package peersdk

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"sync"
	"time"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	pbgrpc "proxymesh/internal/grpc"
)

type PeerClient struct {
	config    *PeerConfig
	conn      *grpc.ClientConn
	client    pbgrpc.PeerServiceClient
	ctx       context.Context
	cancel    context.CancelFunc
	sessionID string
	mu        sync.RWMutex
	stopped   bool
}

func NewPeerClient(cfg *PeerConfig) (*PeerClient, error) {
	ctx, cancel := context.WithCancel(context.Background())

	var opts []grpc.DialOption
	if !cfg.MTLSEnabled {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}

	conn, err := grpc.Dial(cfg.GatewayEndpoint, opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to dial gateway: %w", err)
	}

	return &PeerClient{
		config: cfg,
		conn:   conn,
		client: pbgrpc.NewPeerServiceClient(conn),
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (c *PeerClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return fmt.Errorf("client is stopped")
	}

	resp, err := c.client.Connect(c.ctx, &pbgrpc.ConnectRequest{
		NodeId:     c.config.NodeID,
		Ip:         "127.0.0.1",
		Country:    "US",
		City:       "Unknown",
		Isp:        "Residential",
		Os:         "unknown",
		Ipv6Subnet: "",
	})
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("connect rejected: %s", resp.Message)
	}

	c.sessionID = resp.SessionId
	log.Printf("gRPC connected: node_id=%s session=%s", c.config.NodeID, c.sessionID)
	return nil
}

func (c *PeerClient) Heartbeat(battery int, cpuUsage float64, isCharging bool, bytesSent, bytesReceived, durationSeconds int64) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.stopped {
		return fmt.Errorf("client is stopped")
	}

	_, err := c.client.Heartbeat(c.ctx, &pbgrpc.HeartbeatRequest{
		NodeId:          c.config.NodeID,
		Battery:         int32(battery),
		CpuUsage:        float32(cpuUsage),
		IsCharging:      isCharging,
		BandwidthSent:    bytesSent,
		BandwidthReceived: bytesReceived,
	})
	if err != nil {
		return fmt.Errorf("heartbeat failed: %w", err)
	}
	return nil
}

func (c *PeerClient) ReportBandwidth(bytesSent, bytesReceived, durationSeconds int64) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.stopped {
		return fmt.Errorf("client is stopped")
	}

	_, err := c.client.ReportBandwidth(c.ctx, &pbgrpc.BandwidthReport{
		NodeId:          c.config.NodeID,
		BytesSent:       bytesSent,
		BytesReceived:   bytesReceived,
		DurationSeconds: durationSeconds,
	})
	if err != nil {
		return fmt.Errorf("bandwidth report failed: %w", err)
	}
	return nil
}

func (c *PeerClient) Disconnect(reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return nil
	}
	c.stopped = true

	_, err := c.client.Disconnect(c.ctx, &pbgrpc.DisconnectRequest{
		NodeId:  c.config.NodeID,
		Reason:  reason,
	})
	if err != nil {
		log.Printf("gRPC disconnect error: %v", err)
	}
	return err
}

func (c *PeerClient) Stop() {
	c.mu.Lock()
	c.stopped = true
	c.mu.Unlock()

	c.cancel()
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *PeerClient) StartTelemetryStream(telemetryCh <-chan TelemetryUpdate) {
	go func() {
		stream, err := c.client.StreamTelemetry(c.ctx)
		if err != nil {
			log.Printf("Failed to open telemetry stream: %v", err)
			return
		}

		if err := stream.Send(&pbgrpc.TelemetryStreamRequest{
			NodeId: c.config.NodeID,
		}); err != nil {
			log.Printf("Failed to send initial telemetry request: %v", err)
			return
		}

		go func() {
			for {
				_, err := stream.Recv()
				if err != nil {
					log.Printf("Telemetry stream recv error: %v", err)
					return
				}
			}
		}()

		for update := range telemetryCh {
			if err := stream.Send(&pbgrpc.TelemetryStreamRequest{
				NodeId:          c.config.NodeID,
				BatteryLevel:    int32(update.Battery),
				CpuUsage:        float32(update.CPUUsage),
				BandwidthSent:    update.BytesSent,
				BandwidthReceived: update.BytesReceived,
				IsCharging:      update.IsCharging,
				IpAddress:       update.IP,
				Timestamp:       time.Now().Unix(),
			}); err != nil {
				log.Printf("Failed to send telemetry update: %v", err)
				return
			}
		}
	}()
}

type TelemetryUpdate struct {
	Battery       int
	CPUUsage      float64
	IsCharging    bool
	BytesSent     int64
	BytesReceived int64
	IP            string
}
