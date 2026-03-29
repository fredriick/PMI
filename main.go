package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	geoip := gateway.NewGeoIPService()

	mm := matchmaker.NewMatchmaker(redisClient, cfg.Gateway.CircuitBreakerThreshold, cfg.Matchmaker.CooldownTTLMinutes, geoip.Lookup)

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

	var rateLimiter gateway.RateLimiter
	if cfg.Gateway.RateLimitDistributed {
		rateLimiter = gateway.NewDistributedRateLimiter(redisClient.Client(), cfg.Gateway.RateLimitRequests, cfg.Gateway.RateLimitWindowSeconds)
		log.Printf("Using distributed rate limiter (Redis)")
	} else {
		rateLimiter = gateway.NewLocalRateLimiter(cfg.Gateway.RateLimitRequests, cfg.Gateway.RateLimitWindowSeconds)
		log.Printf("Using local rate limiter (in-memory)")
	}

	gw := gateway.NewGateway(cfg, mm, compliance, tracer, rateLimiter)

	apiKeyService := gateway.NewAPIKeyService(redisClient.Client())
	gw.SetAPIKeyService(apiKeyService)

	auditLogger, err := gateway.NewAuditLogger("audit.log", redisClient.Client())
	if err != nil {
		log.Printf("Warning: failed to create audit logger: %v", err)
	} else {
		defer auditLogger.Close()
		log.Printf("Audit logging enabled")
	}

	setupAdminRoutes(gw.Router(), mm, subnetAllocator, apiKeyService, auditLogger)

	config.OnChange(func(newCfg *config.Config) {
		gw.ReloadCompliance(&newCfg.Compliance)
		log.Printf("Compliance config reloaded: %d blocked domains", len(newCfg.Compliance.BlockedDomains))
	})

	config.Watch()

	server, err := gw.StartServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		log.Printf("Starting Gateway on %s:%d", cfg.Gateway.Host, cfg.Gateway.Port)
		if cfg.Gateway.MTLSEnabled {
			err = server.ListenAndServeTLS(cfg.Gateway.ServerCertPath, cfg.Gateway.ServerKeyPath)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start gateway: %v", err)
		}
	}()

	go func() {
		peerServer := grpc.NewPeerServer(cfg, mm)
		if err := peerServer.Start(9000); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	mm.StopHealthCheck()
	mm.StopCooldownCleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
