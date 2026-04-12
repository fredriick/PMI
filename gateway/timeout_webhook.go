package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type EndpointTimeout struct {
	mu       sync.RWMutex
	timeouts map[string]time.Duration
}

func NewEndpointTimeout() *EndpointTimeout {
	return &EndpointTimeout{
		timeouts: make(map[string]time.Duration),
	}
}

func (et *EndpointTimeout) SetTimeout(endpoint string, duration time.Duration) {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.timeouts[endpoint] = duration
}

func (et *EndpointTimeout) GetTimeout(endpoint string) time.Duration {
	et.mu.RLock()
	defer et.mu.RUnlock()
	return et.timeouts[endpoint]
}

func (et *EndpointTimeout) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		endpoint := c.Request.URL.Path
		timeout := et.GetTimeout(endpoint)

		if timeout > 0 {
			ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
			defer cancel()

			c.Request = c.Request.WithContext(ctx)

			done := make(chan struct{})
			go func() {
				c.Next()
				close(done)
			}()

			select {
			case <-done:
			case <-ctx.Done():
				c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
					"error":    "Request timeout",
					"timeout":  timeout.String(),
					"endpoint": endpoint,
				})
			}
		} else {
			c.Next()
		}
	}
}

func (et *EndpointTimeout) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/admin/timeouts", et.setTimeoutHandler)
	r.GET("/api/admin/timeouts", et.listTimeoutsHandler)
}

func (et *EndpointTimeout) setTimeoutHandler(c *gin.Context) {
	var req struct {
		Endpoint string `json:"endpoint" binding:"required"`
		Timeout  int    `json:"timeout_seconds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	et.SetTimeout(req.Endpoint, time.Duration(req.Timeout)*time.Second)
	c.JSON(200, gin.H{"status": "success"})
}

func (et *EndpointTimeout) listTimeoutsHandler(c *gin.Context) {
	et.mu.RLock()
	defer et.mu.RUnlock()

	timeouts := make(map[string]string)
	for endpoint, duration := range et.timeouts {
		timeouts[endpoint] = duration.String()
	}

	c.JSON(200, gin.H{"timeouts": timeouts})
}

type NodeWebhook struct {
	mu         sync.RWMutex
	webhooks   map[string][]WebhookConfig
	failCount  map[string]int
	maxRetries int
}

type WebhookConfig struct {
	URL    string   `json:"url"`
	Secret string   `json:"secret"`
	Events []string `json:"events"`
}

type WebhookPayload struct {
	Event     string    `json:"event"`
	NodeID    string    `json:"node_id"`
	State     string    `json:"state"`
	Timestamp time.Time `json:"timestamp"`
}

func NewNodeWebhook(maxRetries int) *NodeWebhook {
	return &NodeWebhook{
		webhooks:   make(map[string][]WebhookConfig),
		failCount:  make(map[string]int),
		maxRetries: maxRetries,
	}
}

func (nw *NodeWebhook) RegisterWebhook(nodeID string, config WebhookConfig) {
	nw.mu.Lock()
	defer nw.mu.Unlock()

	nw.webhooks[nodeID] = append(nw.webhooks[nodeID], config)
}

func (nw *NodeWebhook) TriggerEvent(nodeID, event, state string) {
	nw.mu.RLock()
	webhooks := nw.webhooks[nodeID]
	nw.mu.RUnlock()

	payload := WebhookPayload{
		Event:     event,
		NodeID:    nodeID,
		State:     state,
		Timestamp: time.Now(),
	}

	for _, wh := range webhooks {
		for _, e := range wh.Events {
			if e == event || e == "*" {
				go nw.sendWebhook(wh.URL, payload)
				break
			}
		}
	}
}

func (nw *NodeWebhook) sendWebhook(url string, payload WebhookPayload) error {
	data, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

type ActivityDashboard struct {
	mu         sync.RWMutex
	activities []Activity
	maxEntries int
}

type Activity struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	User      string    `json:"user"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
	IP        string    `json:"ip"`
}

func NewActivityDashboard(maxEntries int) *ActivityDashboard {
	if maxEntries <= 0 {
		maxEntries = 500
	}
	return &ActivityDashboard{
		activities: make([]Activity, 0, maxEntries),
		maxEntries: maxEntries,
	}
}

func (ad *ActivityDashboard) Record(activity Activity) {
	activity.ID = fmt.Sprintf("act-%d", time.Now().UnixNano())
	activity.Timestamp = time.Now()

	ad.mu.Lock()
	ad.activities = append(ad.activities, activity)
	if len(ad.activities) > ad.maxEntries {
		ad.activities = ad.activities[1:]
	}
	ad.mu.Unlock()
}

func (ad *ActivityDashboard) GetActivities(limit int) []Activity {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	if limit <= 0 || limit > len(ad.activities) {
		limit = len(ad.activities)
	}

	result := make([]Activity, limit)
	copy(result, ad.activities[len(ad.activities)-limit:])
	return result
}

func (ad *ActivityDashboard) GetByUser(username string, limit int) []Activity {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	var result []Activity
	for _, a := range ad.activities {
		if a.User == username {
			result = append(result, a)
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}

func (ad *ActivityDashboard) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.GetHeader("X-User")
		if user == "" {
			user = "anonymous"
		}

		path := c.Request.URL.Path
		method := c.Request.Method

		if method == "GET" || path == "/health" {
			c.Next()
			return
		}

		c.Next()

		activity := Activity{
			Type:   "api_request",
			User:   user,
			Action: fmt.Sprintf("%s %s", method, path),
			IP:     c.ClientIP(),
		}

		if c.Writer.Status() >= 400 {
			activity.Details = fmt.Sprintf("failed with status %d", c.Writer.Status())
		}

		ad.Record(activity)
	}
}

func (ad *ActivityDashboard) listHandler(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	activities := ad.GetActivities(limit)
	c.JSON(200, gin.H{"activities": activities})
}

func (ad *ActivityDashboard) userHandler(c *gin.Context) {
	user := c.Param("user")
	limit := 20
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	activities := ad.GetByUser(user, limit)
	c.JSON(200, gin.H{"user": user, "activities": activities})
}
