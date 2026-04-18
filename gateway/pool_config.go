package gateway

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type NodePoolConfig struct {
	MaxIdle     int           `json:"max_idle"`
	MaxActive   int           `json:"max_active"`
	IdleTimeout time.Duration `json:"idle_timeout"`
	Wait        bool          `json:"wait"`
	MaxConnLife time.Duration `json:"max_conn_life"`
}

func (npc *NodePoolConfig) ApplyToPool(pool *ConnPool) {
	if npc.MaxIdle > 0 {
		_ = npc.MaxIdle
	}
	if npc.MaxActive > 0 {
		_ = npc.MaxActive
	}
	if npc.IdleTimeout > 0 {
		_ = npc.IdleTimeout
	}
	_ = npc.Wait
	_ = npc.MaxConnLife
}

type PoolConfigManager struct {
	mu      sync.RWMutex
	configs map[string]NodePoolConfig
}

func NewPoolConfigManager() *PoolConfigManager {
	return &PoolConfigManager{
		configs: make(map[string]NodePoolConfig),
	}
}

func (pcm *PoolConfigManager) SetConfig(nodeID string, config NodePoolConfig) {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()
	pcm.configs[nodeID] = config
}

func (pcm *PoolConfigManager) GetConfig(nodeID string) NodePoolConfig {
	pcm.mu.RLock()
	defer pcm.mu.RUnlock()
	return pcm.configs[nodeID]
}

func (pcm *PoolConfigManager) SetDefault(config NodePoolConfig) {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()
	pcm.configs["_default"] = config
}

func (pcm *PoolConfigManager) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/admin/pool-config", pcm.setConfigHandler)
	r.GET("/api/admin/pool-config", pcm.listConfigHandler)
}

func (pcm *PoolConfigManager) setConfigHandler(c *gin.Context) {
	var req struct {
		NodeID string         `json:"node_id"`
		Config NodePoolConfig `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	pcm.SetConfig(req.NodeID, req.Config)
	c.JSON(200, gin.H{"status": "success"})
}

func (pcm *PoolConfigManager) listConfigHandler(c *gin.Context) {
	pcm.mu.RLock()
	defer pcm.mu.RUnlock()

	configs := make(map[string]NodePoolConfig)
	for k, v := range pcm.configs {
		configs[k] = v
	}

	c.JSON(200, gin.H{"configs": configs})
}
