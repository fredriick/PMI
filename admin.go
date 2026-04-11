package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"proxymesh/gateway"
	"proxymesh/internal/models"
	"proxymesh/internal/subnet"
	"proxymesh/matchmaker"
)

func setupAdminRoutes(r *gin.Engine, mm *matchmaker.Matchmaker, sa *subnet.SubnetAllocator, apiKeySvc *gateway.APIKeyService, auditLog *gateway.AuditLogger) {
	admin := r.Group("/api/admin")
	admin.Use(adminAuthMiddleware())
	if auditLog != nil {
		admin.Use(auditLog.AuditMiddleware())
	}
	{
		admin.POST("/nodes", registerNodeHandler(mm))
		admin.POST("/nodes/:id/heartbeat", heartbeatHandler(mm))
		admin.DELETE("/nodes/:id", deregisterNodeHandler(mm))
		admin.GET("/nodes/:id", getNodeHandler(mm))
		admin.GET("/nodes", listNodesHandler(mm))
		admin.GET("/cooldowns", listCooldownsHandler(mm))
		admin.GET("/sessions", listSessionsHandler(mm))
		admin.DELETE("/sessions/:id", deleteSessionHandler(mm))
		admin.GET("/capacity", capacityReportHandler(mm))
		admin.GET("/scaling", scalingRecommendationsHandler(mm))
		admin.GET("/circuitbreakers", circuitBreakersHandler(mm))
		admin.POST("/circuitbreakers/:nodeId/reset", resetCircuitBreakerHandler(mm))
		if auditLog != nil {
			admin.GET("/audit", listAuditEntriesHandler(auditLog))
		}
		admin.GET("/users", listUsersHandler)
		admin.POST("/users", createUserHandler)
		admin.DELETE("/users/:username", deleteUserHandler)
	}

	if apiKeySvc != nil {
		keys := r.Group("/api/keys")
		keys.Use(adminAuthMiddleware())
		{
			keys.POST("", createAPIKeyHandler(apiKeySvc))
			keys.GET("", listAPIKeysHandler(apiKeySvc))
			keys.DELETE("", revokeAPIKeyHandler(apiKeySvc))
			keys.POST("/ratelimit", setKeyRateLimitHandler(apiKeySvc))
		}
	}

	if sa != nil {
		subnets := r.Group("/api/subnets")
		subnets.Use(adminAuthMiddleware())
		{
			subnets.POST("/pools", createPoolHandler(sa))
			subnets.POST("/allocate", allocateSubnetHandler(sa))
			subnets.DELETE("/allocate/:nodeID", deallocateSubnetHandler(sa))
			subnets.GET("/pools", listPoolsHandler(sa))
			subnets.GET("/nodes/:nodeID", getNodeSubnetHandler(sa))
		}
	}
}

func adminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("X-Admin-Key")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing admin key",
			})
			return
		}
		c.Next()
	}
}

func registerNodeHandler(mm *matchmaker.Matchmaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.NodeRegistrationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request: " + err.Error(),
			})
			return
		}

		if err := mm.RegisterNode(&req); err != nil {
			c.JSON(http.StatusInternalServerError, models.NodeRegistrationResponse{
				Status:  "error",
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, models.NodeRegistrationResponse{
			Status:  "success",
			NodeID:  req.NodeID,
			Message: "Node registered successfully",
		})
	}
}

func heartbeatHandler(mm *matchmaker.Matchmaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.Param("id")
		var req models.NodeHeartbeatRequest
		req.NodeID = nodeID

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request: " + err.Error(),
			})
			return
		}

		req.NodeID = nodeID

		if err := mm.Heartbeat(&req); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"node_id": nodeID,
		})
	}
}

func deregisterNodeHandler(mm *matchmaker.Matchmaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.Param("id")

		if err := mm.DeregisterNode(nodeID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"node_id": nodeID,
			"message": "Node deregistered",
		})
	}
}

func getNodeHandler(mm *matchmaker.Matchmaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.Param("id")

		node, err := mm.GetNodeStatus(nodeID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Node not found",
			})
			return
		}

		c.JSON(http.StatusOK, node)
	}
}

func listNodesHandler(mm *matchmaker.Matchmaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodes, err := mm.GetAllNodes()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to list nodes: " + err.Error(),
			})
			return
		}

		type nodeSummary struct {
			ID   string       `json:"id"`
			Node *models.Node `json:"node,omitempty"`
			Err  string       `json:"error,omitempty"`
		}

		var results []nodeSummary
		for _, nodeID := range nodes {
			node, err := mm.GetNodeStatus(nodeID)
			ns := nodeSummary{ID: nodeID}
			if err != nil {
				ns.Err = err.Error()
			} else {
				ns.Node = node
			}
			results = append(results, ns)
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"count":  len(results),
			"nodes":  results,
		})
	}
}

func listCooldownsHandler(mm *matchmaker.Matchmaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		entries, err := mm.GetCooldownEntries()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to list cooldowns: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "success",
			"cooldowns": entries,
		})
	}
}

func listSessionsHandler(mm *matchmaker.Matchmaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessions, err := mm.ListSessions()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to list sessions: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "success",
			"count":    len(sessions),
			"sessions": sessions,
		})
	}
}

func deleteSessionHandler(mm *matchmaker.Matchmaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")

		if err := mm.DeleteSession(sessionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":     "success",
			"session_id": sessionID,
			"message":    "Session deleted",
		})
	}
}

func capacityReportHandler(mm *matchmaker.Matchmaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		cp := matchmaker.NewCapacityPlanner(mm.GetRedis())
		report, err := cp.GenerateReport()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, report)
	}
}

func scalingRecommendationsHandler(mm *matchmaker.Matchmaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		cp := matchmaker.NewCapacityPlanner(mm.GetRedis())
		report, err := cp.GetScalingRecommendations()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, report)
	}
}

func circuitBreakersHandler(mm *matchmaker.Matchmaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		breakers := mm.GetCircuitBreakers()
		c.JSON(http.StatusOK, gin.H{
			"circuit_breakers": breakers,
			"count":            len(breakers),
		})
	}
}

func resetCircuitBreakerHandler(mm *matchmaker.Matchmaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.Param("nodeId")
		if nodeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "node_id required"})
			return
		}

		mm.ResetCircuitBreaker(nodeID)
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"node_id": nodeID,
			"message": "Circuit breaker reset",
		})
	}
}

// Subnet handlers

func createPoolHandler(sa *subnet.SubnetAllocator) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Prefix    string `json:"prefix" binding:"required"`
			PrefixLen int    `json:"prefix_len" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := sa.RegisterPool(req.Prefix, req.PrefixLen); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":     "success",
			"prefix":     req.Prefix,
			"prefix_len": req.PrefixLen,
		})
	}
}

func allocateSubnetHandler(sa *subnet.SubnetAllocator) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			NodeID    string `json:"node_id" binding:"required"`
			Prefix    string `json:"prefix" binding:"required"`
			PrefixLen int    `json:"prefix_len" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		subnetAddr, err := sa.Allocate(req.NodeID, req.Prefix, req.PrefixLen)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"node_id": req.NodeID,
			"subnet":  subnetAddr,
		})
	}
}

func deallocateSubnetHandler(sa *subnet.SubnetAllocator) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.Param("nodeID")

		if err := sa.Deallocate(nodeID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"node_id": nodeID,
			"message": "Subnet deallocated",
		})
	}
}

func listPoolsHandler(sa *subnet.SubnetAllocator) gin.HandlerFunc {
	return func(c *gin.Context) {
		pools, err := sa.ListPools()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"pools":  pools,
		})
	}
}

func getNodeSubnetHandler(sa *subnet.SubnetAllocator) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.Param("nodeID")

		subnetAddr, err := sa.GetNodeSubnet(nodeID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"node_id": nodeID,
			"subnet":  subnetAddr,
		})
	}
}

// API Key handlers

func createAPIKeyHandler(svc *gateway.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name    string `json:"name" binding:"required"`
			TTLDays int    `json:"ttl_days"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		key, err := svc.CreateKey(req.Name, req.TTLDays)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status": "success",
			"key":    key.Key,
			"name":   key.Name,
			"note":   "Store this key securely. It cannot be retrieved again.",
		})
	}
}

func listAPIKeysHandler(svc *gateway.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		keys, err := svc.ListKeys()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"keys":   keys,
		})
	}
}

func revokeAPIKeyHandler(svc *gateway.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Key string `json:"key" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := svc.RevokeKey(req.Key); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "API key revoked",
		})
	}
}

func setKeyRateLimitHandler(svc *gateway.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Key           string `json:"key" binding:"required"`
			Requests      int    `json:"requests" binding:"required"`
			WindowSeconds int    `json:"window_seconds" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := svc.SetKeyRateLimit(req.Key, req.Requests, req.WindowSeconds); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": fmt.Sprintf("Rate limit set: %d requests per %d seconds", req.Requests, req.WindowSeconds),
		})
	}
}

func listAuditEntriesHandler(al *gateway.AuditLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		date := c.Query("date")
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		action := c.Query("action")
		limit := int64(100)
		if l := c.Query("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
		}

		entries, err := al.GetEntries(date, action, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"date":    date,
			"count":   len(entries),
			"entries": entries,
		})
	}
}

func listUsersHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"users": []gin.H{
			{"username": "admin", "email": "admin@proxymesh.local", "roles": []string{"superadmin"}, "active": true, "last_login": time.Now().Format(time.RFC3339)},
			{"username": "operator", "email": "ops@proxymesh.local", "roles": []string{"operator"}, "active": true},
		},
	})
}

func createUserHandler(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username and password required"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"id":      req.Username,
		"message": "User created successfully",
	})
}

func deleteUserHandler(c *gin.Context) {
	username := c.Param("username")
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User " + username + " deleted",
	})
}
