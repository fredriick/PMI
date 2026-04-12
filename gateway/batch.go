package gateway

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type BatchProcessor struct {
	mu           sync.RWMutex
	handlers     map[string]BatchHandler
	maxBatchSize int
	timeout      time.Duration
}

type BatchHandler func([]map[string]interface{}) ([]map[string]interface{}, error)

type BatchRequest struct {
	Operations []map[string]interface{} `json:"operations"`
}

type BatchResponse struct {
	Results []map[string]interface{} `json:"results"`
	Errors  []string                 `json:"errors,omitempty"`
}

func NewBatchProcessor(maxBatchSize int, timeout time.Duration) *BatchProcessor {
	return &BatchProcessor{
		handlers:     make(map[string]BatchHandler),
		maxBatchSize: maxBatchSize,
		timeout:      timeout,
	}
}

func (bp *BatchProcessor) RegisterHandler(operation string, handler BatchHandler) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.handlers[operation] = handler
}

func (bp *BatchProcessor) Process(ctx context.Context, batchReq BatchRequest) (*BatchResponse, error) {
	if len(batchReq.Operations) > bp.maxBatchSize {
		return nil, fmt.Errorf("batch size exceeds maximum of %d", bp.maxBatchSize)
	}

	ctx, cancel := context.WithTimeout(ctx, bp.timeout)
	defer cancel()

	results := make([]map[string]interface{}, 0, len(batchReq.Operations))
	errors := make([]string, 0)

	for _, op := range batchReq.Operations {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		opType, ok := op["type"].(string)
		if !ok {
			errors = append(errors, "operation type missing")
			results = append(results, map[string]interface{}{"error": "type required"})
			continue
		}

		bp.mu.RLock()
		handler, exists := bp.handlers[opType]
		bp.mu.RUnlock()

		if !exists {
			errors = append(errors, fmt.Sprintf("unknown operation: %s", opType))
			results = append(results, map[string]interface{}{"error": "unknown type"})
			continue
		}

		result, err := handler([]map[string]interface{}{op})
		if err != nil {
			errors = append(errors, err.Error())
			results = append(results, map[string]interface{}{"error": err.Error()})
		} else if len(result) > 0 {
			results = append(results, result[0])
		}
	}

	return &BatchResponse{
		Results: results,
		Errors:  errors,
	}, nil
}

func (bp *BatchProcessor) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/batch", bp.batchHandler)
}

func (bp *BatchProcessor) batchHandler(c *gin.Context) {
	var req BatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := bp.Process(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

type NodeHealthScore struct {
	mu      sync.RWMutex
	scores  map[string]*HealthScore
	redis   *redis.Client
	ctx     context.Context
	weights HealthWeights
}

type HealthWeights struct {
	LatencyWeight     float64 `json:"latency_weight"`
	LoadWeight        float64 `json:"load_weight"`
	SuccessRateWeight float64 `json:"success_rate_weight"`
	ReputationWeight  float64 `json:"reputation_weight"`
	UptimeWeight      float64 `json:"uptime_weight"`
}

type HealthScore struct {
	NodeID       string    `json:"node_id"`
	TotalScore   float64   `json:"total_score"`
	LatencyScore float64   `json:"latency_score"`
	LoadScore    float64   `json:"load_score"`
	SuccessScore float64   `json:"success_score"`
	Reputation   float64   `json:"reputation"`
	UptimeScore  float64   `json:"uptime_score"`
	LastUpdate   time.Time `json:"last_update"`
}

func NewNodeHealthScore(redisClient *redis.Client, weights HealthWeights) *NodeHealthScore {
	if weights.LatencyWeight == 0 {
		weights = HealthWeights{
			LatencyWeight:     30,
			LoadWeight:        20,
			SuccessRateWeight: 30,
			ReputationWeight:  15,
			UptimeWeight:      5,
		}
	}

	return &NodeHealthScore{
		scores:  make(map[string]*HealthScore),
		redis:   redisClient,
		ctx:     context.Background(),
		weights: weights,
	}
}

func (nhs *NodeHealthScore) CalculateScore(nodeID string, latencyMs int64, load int, successRate float64, reputation float64, uptimeHours float64) float64 {
	latencyScore := nhs.calculateLatencyScore(latencyMs)
	loadScore := nhs.calculateLoadScore(load)
	successScore := successRate * 100
	reputationScore := reputation
	uptimeScore := nhs.calculateUptimeScore(uptimeHours)

	totalScore := (latencyScore * nhs.weights.LatencyWeight / 100) +
		(loadScore * nhs.weights.LoadWeight / 100) +
		(successScore * nhs.weights.SuccessRateWeight / 100) +
		(reputationScore * nhs.weights.ReputationWeight / 100) +
		(uptimeScore * nhs.weights.UptimeWeight / 100)

	nhs.mu.Lock()
	nhs.scores[nodeID] = &HealthScore{
		NodeID:       nodeID,
		TotalScore:   totalScore,
		LatencyScore: latencyScore,
		LoadScore:    loadScore,
		SuccessScore: successScore,
		Reputation:   reputation,
		UptimeScore:  uptimeScore,
		LastUpdate:   time.Now(),
	}
	nhs.mu.Unlock()

	return totalScore
}

func (nhs *NodeHealthScore) calculateLatencyScore(latencyMs int64) float64 {
	if latencyMs <= 50 {
		return 100
	}
	if latencyMs >= 500 {
		return 0
	}
	return 100 - ((float64(latencyMs) - 50) * 100 / 450)
}

func (nhs *NodeHealthScore) calculateLoadScore(load int) float64 {
	if load <= 10 {
		return 100
	}
	if load >= 100 {
		return 0
	}
	return 100 - (float64(load) * 100 / 90)
}

func (nhs *NodeHealthScore) calculateUptimeScore(uptimeHours float64) float64 {
	if uptimeHours >= 720 {
		return 100
	}
	return (uptimeHours / 720) * 100
}

func (nhs *NodeHealthScore) GetScore(nodeID string) *HealthScore {
	nhs.mu.RLock()
	defer nhs.mu.RUnlock()
	return nhs.scores[nodeID]
}

func (nhs *NodeHealthScore) GetAllScores() []*HealthScore {
	nhs.mu.RLock()
	defer nhs.mu.RUnlock()

	scores := make([]*HealthScore, 0, len(nhs.scores))
	for _, score := range nhs.scores {
		scores = append(scores, score)
	}
	return scores
}

func (nhs *NodeHealthScore) GetTopNodes(count int) []string {
	scores := nhs.GetAllScores()

	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].TotalScore > scores[i].TotalScore {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	result := make([]string, 0, count)
	for i := 0; i < len(scores) && i < count; i++ {
		result = append(result, scores[i].NodeID)
	}
	return result
}

type KeyPriority struct {
	mu         sync.RWMutex
	priorities map[string]int
	redis      *redis.Client
	ctx        context.Context
}

func NewKeyPriority(redisClient *redis.Client) *KeyPriority {
	return &KeyPriority{
		priorities: make(map[string]int),
		redis:      redisClient,
		ctx:        context.Background(),
	}
}

func (kp *KeyPriority) SetPriority(apiKey string, priority int) {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	kp.priorities[apiKey] = priority

	if kp.redis != nil {
		key := fmt.Sprintf("key:priority:%s", apiKey)
		kp.redis.Set(kp.ctx, key, priority, 0)
	}
}

func (kp *KeyPriority) GetPriority(apiKey string) int {
	kp.mu.RLock()
	defer kp.mu.RUnlock()

	if priority, exists := kp.priorities[apiKey]; exists {
		return priority
	}
	return 0
}

func (kp *KeyPriority) GetWeightedPriority(apiKey string, baseWeight float64) float64 {
	priority := kp.GetPriority(apiKey)
	return baseWeight * float64(priority+1)
}

func (kp *KeyPriority) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			key := extractBearerToken(authHeader)
			if key == "" {
				key = authHeader
			}

			priority := kp.GetPriority(key)
			c.Set("request_priority", priority)
		}

		c.Next()
	}
}
