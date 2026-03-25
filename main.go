package main

import (
	"log"

	"proxymesh/gateway"
	"proxymesh/internal/config"
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

	mm := matchmaker.NewMatchmaker(redisClient, cfg.Gateway.CircuitBreakerThreshold)

	compliance := gateway.NewComplianceService(&cfg.Compliance)

	gw := gateway.NewGateway(cfg, mm, compliance)

	setupAdminRoutes(gw.Router(), mm)

	log.Printf("Starting Gateway on %s:%d", cfg.Gateway.Host, cfg.Gateway.Port)
	if err := gw.Start(); err != nil {
		log.Fatalf("Failed to start gateway: %v", err)
	}
}
