package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"proxymesh/internal/models"
	"proxymesh/internal/subnet"
	"proxymesh/matchmaker"
)

func setupAdminRoutes(r *gin.Engine, mm *matchmaker.Matchmaker, sa *subnet.SubnetAllocator) {
	admin := r.Group("/api/admin")
	admin.Use(adminAuthMiddleware())
	{
		admin.POST("/nodes", registerNodeHandler(mm))
		admin.POST("/nodes/:id/heartbeat", heartbeatHandler(mm))
		admin.DELETE("/nodes/:id", deregisterNodeHandler(mm))
		admin.GET("/nodes/:id", getNodeHandler(mm))
		admin.GET("/nodes", listNodesHandler(mm))
		admin.GET("/cooldowns", listCooldownsHandler(mm))
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
