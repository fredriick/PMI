package main

import (
	"log"

	"proxymesh/gateway"
	"proxymesh/internal/config"
	"proxymesh/internal/grpc"
	"proxymesh/internal/subnet"
	"proxymesh/matchmaker"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	redisClient, err := matchmaker.NewRedisClient(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	mm := matchmaker.NewMatchmaker(redisClient, cfg.Gateway.CircuitBreakerThreshold, cfg.Matchmaker.CooldownTTLMinutes)

	var subnetAllocator *subnet.SubnetAllocator
	if cfg.Subnet.Enabled {
		subnetAllocator = subnet.NewSubnetAllocator(redisClient.Client())
		if err := subnetAllocator.RegisterPool(cfg.Subnet.Prefix, cfg.Subnet.PrefixLen); err != nil {
			log.Printf("Warning: failed to register subnet pool: %v", err)
		} else {
			log.Printf("Subnet pool registered: %s/%d", cfg.Subnet.Prefix, cfg.Subnet.PrefixLen)
		}
	}

	compliance := gateway.NewComplianceService(&cfg.Compliance)

	tracer, err := gateway.InitTracing("gateway", cfg.Gateway.TracingEnabled)
	if err != nil {
		log.Printf("Warning: failed to initialize tracing: %v", err)
	}

	gw := gateway.NewGateway(cfg, mm, compliance, tracer)

	setupAdminRoutes(gw.Router(), mm, subnetAllocator)

	go func() {
		peerServer := grpc.NewPeerServer(cfg, mm)
		if err := peerServer.Start(9000); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	log.Printf("Starting Gateway on %s:%d", cfg.Gateway.Host, cfg.Gateway.Port)
	if err := gw.Start(); err != nil {
		log.Fatalf("Failed to start gateway: %v", err)
	}
}
