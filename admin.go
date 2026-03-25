package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"proxymesh/internal/models"
	"proxymesh/matchmaker"
)

func setupAdminRoutes(r *gin.Engine, mm *matchmaker.Matchmaker) {
	admin := r.Group("/api/admin")
	admin.Use(adminAuthMiddleware())
	{
		admin.POST("/nodes", registerNodeHandler(mm))
		admin.POST("/nodes/:id/heartbeat", heartbeatHandler(mm))
		admin.DELETE("/nodes/:id", deregisterNodeHandler(mm))
		admin.GET("/nodes/:id", getNodeHandler(mm))
		admin.GET("/nodes", listNodesHandler(mm))
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
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Use Redis KEYS nodes:* to list all nodes",
		})
	}
}
