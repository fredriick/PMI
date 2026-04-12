package gateway

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type RequestReplay struct {
	mu       sync.RWMutex
	requests map[string]*ReplayRequest
	redis    *redis.Client
	ctx      context.Context
	maxAge   time.Duration
}

type ReplayRequest struct {
	ID        string            `json:"id"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
	ClientIP  string            `json:"client_ip"`
	CreatedAt time.Time         `json:"created_at"`
}

func NewRequestReplay(redisClient *redis.Client, maxAge time.Duration) *RequestReplay {
	return &RequestReplay{
		requests: make(map[string]*ReplayRequest),
		redis:    redisClient,
		ctx:      context.Background(),
		maxAge:   maxAge,
	}
}

func (rr *RequestReplay) RecordRequest(req *http.Request, body []byte, clientIP string) string {
	id := fmt.Sprintf("req-%d", time.Now().UnixNano())

	headers := make(map[string]string)
	for k, v := range req.Header {
		headers[k] = strings.Join(v, ", ")
	}

	replayReq := &ReplayRequest{
		ID:        id,
		Method:    req.Method,
		URL:       req.URL.String(),
		Headers:   headers,
		Body:      string(body),
		ClientIP:  clientIP,
		CreatedAt: time.Now(),
	}

	rr.mu.Lock()
	rr.requests[id] = replayReq
	rr.mu.Unlock()

	if rr.redis != nil {
		data := fmt.Sprintf("%s|%s|%s|%s", req.Method, req.URL.String(), string(body), clientIP)
		rr.redis.Set(rr.ctx, fmt.Sprintf("replay:%s", id), data, rr.maxAge)
	}

	return id
}

func (rr *RequestReplay) GetRequest(id string) *ReplayRequest {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	return rr.requests[id]
}

func (rr *RequestReplay) ReplayRequest(id string, targetURL string) error {
	req := rr.GetRequest(id)
	if req == nil {
		return fmt.Errorf("request not found: %s", id)
	}

	httpReq, err := http.NewRequest(req.Method, targetURL+req.URL, strings.NewReader(req.Body))
	if err != nil {
		return err
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (rr *RequestReplay) ListRequests(limit int) []*ReplayRequest {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	requests := make([]*ReplayRequest, 0, limit)
	for _, req := range rr.requests {
		requests = append(requests, req)
		if len(requests) >= limit {
			break
		}
	}
	return requests
}

func (rr *RequestReplay) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/admin/replay", rr.createReplayHandler)
	r.GET("/api/admin/replay", rr.listReplayHandler)
	r.POST("/api/admin/replay/:id", rr.replayHandler)
}

func (rr *RequestReplay) createReplayHandler(c *gin.Context) {
	var req ReplayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	id := fmt.Sprintf("req-%d", time.Now().UnixNano())
	req.ID = id
	req.CreatedAt = time.Now()

	rr.mu.Lock()
	rr.requests[id] = &req
	rr.mu.Unlock()

	c.JSON(201, gin.H{"id": id, "status": "created"})
}

func (rr *RequestReplay) listReplayHandler(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	requests := rr.ListRequests(limit)
	c.JSON(200, gin.H{"requests": requests})
}

func (rr *RequestReplay) replayHandler(c *gin.Context) {
	id := c.Param("id")
	targetURL := c.Query("target")

	if targetURL == "" {
		c.JSON(400, gin.H{"error": "target URL required"})
		return
	}

	if err := rr.ReplayRequest(id, targetURL); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "replayed"})
}

type EndpointSizeLimits struct {
	mu     sync.RWMutex
	limits map[string]SizeLimit
}

type SizeLimit struct {
	Endpoint string `json:"endpoint"`
	Method   string `json:"method"`
	MaxBytes int64  `json:"max_bytes"`
}

func NewEndpointSizeLimits() *EndpointSizeLimits {
	return &EndpointSizeLimits{
		limits: make(map[string]SizeLimit),
	}
}

func (esl *EndpointSizeLimits) SetLimit(endpoint, method string, maxBytes int64) {
	key := fmt.Sprintf("%s:%s", method, endpoint)
	esl.mu.Lock()
	esl.limits[key] = SizeLimit{
		Endpoint: endpoint,
		Method:   method,
		MaxBytes: maxBytes,
	}
	esl.mu.Unlock()
}

func (esl *EndpointSizeLimits) GetLimit(endpoint, method string) int64 {
	key := fmt.Sprintf("%s:%s", method, endpoint)
	esl.mu.RLock()
	defer esl.mu.RUnlock()

	if limit, exists := esl.limits[key]; exists {
		return limit.MaxBytes
	}
	return 10 * 1024 * 1024
}

func (esl *EndpointSizeLimits) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		endpoint := c.Request.URL.Path
		method := c.Request.Method
		maxBytes := esl.GetLimit(endpoint, method)

		if c.Request.ContentLength > maxBytes && maxBytes > 0 {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":     "Request body too large",
				"max_bytes": maxBytes,
				"actual":    c.Request.ContentLength,
			})
			return
		}

		c.Next()
	}
}

func (esl *EndpointSizeLimits) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/admin/limits", esl.setLimitHandler)
	r.GET("/api/admin/limits", esl.listLimitsHandler)
}

func (esl *EndpointSizeLimits) setLimitHandler(c *gin.Context) {
	var req SizeLimit
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	esl.SetLimit(req.Endpoint, req.Method, req.MaxBytes)
	c.JSON(200, gin.H{"status": "success"})
}

func (esl *EndpointSizeLimits) listLimitsHandler(c *gin.Context) {
	esl.mu.RLock()
	defer esl.mu.RUnlock()

	limits := make([]SizeLimit, 0, len(esl.limits))
	for _, limit := range esl.limits {
		limits = append(limits, limit)
	}

	c.JSON(200, gin.H{"limits": limits})
}
