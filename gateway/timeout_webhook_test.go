package gateway

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"proxymesh/internal/config"
)

func TestNodeTimeout_SetGetTimeout(t *testing.T) {
	et := NewEndpointTimeout()

	et.SetTimeout("/api/admin", 30*time.Second)

	got := et.GetTimeout("/api/admin")
	if got != 30*time.Second {
		t.Errorf("expected 30s, got %v", got)
	}

	got = et.GetTimeout("/nonexistent")
	if got != 0 {
		t.Errorf("expected 0 for nonexistent endpoint, got %v", got)
	}
}

func TestNodeTimeout_Middleware_NoTimeout(t *testing.T) {
	et := NewEndpointTimeout()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(et.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestNodeTimeout_Middleware_WithTimeout(t *testing.T) {
	et := NewEndpointTimeout()
	et.SetTimeout("/slow", 10*time.Millisecond)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(et.Middleware())
	router.GET("/slow", func(c *gin.Context) {
		time.Sleep(50 * time.Millisecond)
		c.String(http.StatusOK, "done")
	})

	req := httptest.NewRequest("GET", "/slow", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("expected 504 (timeout), got %d", w.Code)
	}
}

func TestNodeTimeout_SetTimeoutHandler(t *testing.T) {
	et := NewEndpointTimeout()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/admin/timeouts", nil)

	et.setTimeoutHandler(c)
}

func TestActivityDashboard_RecordAndGet(t *testing.T) {
	ad := NewActivityDashboard(100)

	ad.Record(Activity{
		Type:   "api_request",
		User:   "user1",
		Action: "GET /api/data",
	})

	activities := ad.GetActivities(10)
	if len(activities) != 1 {
		t.Errorf("expected 1 activity, got %d", len(activities))
	}
	if activities[0].User != "user1" {
		t.Errorf("expected user1, got %s", activities[0].User)
	}
	if activities[0].ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestActivityDashboard_GetActivities_Limit(t *testing.T) {
	ad := NewActivityDashboard(100)

	for i := 0; i < 10; i++ {
		ad.Record(Activity{User: "user1"})
	}

	activities := ad.GetActivities(3)
	if len(activities) != 3 {
		t.Errorf("expected 3 activities with limit, got %d", len(activities))
	}
}

func TestActivityDashboard_GetActivities_ExceedsEntries(t *testing.T) {
	ad := NewActivityDashboard(100)

	ad.Record(Activity{User: "user1"})
	ad.Record(Activity{User: "user2"})

	activities := ad.GetActivities(100)
	if len(activities) != 2 {
		t.Errorf("expected 2 activities, got %d", len(activities))
	}
}

func TestActivityDashboard_GetByUser(t *testing.T) {
	ad := NewActivityDashboard(100)

	ad.Record(Activity{User: "user1"})
	ad.Record(Activity{User: "user2"})
	ad.Record(Activity{User: "user1"})

	activities := ad.GetByUser("user1", 10)
	if len(activities) != 2 {
		t.Errorf("expected 2 activities for user1, got %d", len(activities))
	}
}

func TestActivityDashboard_Middleware_GET(t *testing.T) {
	ad := NewActivityDashboard(100)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ad.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	activities := ad.GetActivities(10)
	if len(activities) != 0 {
		t.Errorf("expected 0 activities for GET request, got %d", len(activities))
	}
}

func TestActivityDashboard_Middleware_NonGET(t *testing.T) {
	ad := NewActivityDashboard(100)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ad.Middleware())
	router.POST("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.Header.Set("X-User", "testuser")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	activities := ad.GetActivities(10)
	if len(activities) != 1 {
		t.Errorf("expected 1 activity for POST, got %d", len(activities))
	}
	if activities[0].User != "testuser" {
		t.Errorf("expected user 'testuser', got '%s'", activities[0].User)
	}
	if len(activities[0].Details) > 0 {
		t.Error("expected empty details for successful request")
	}
}

func TestActivityDashboard_Middleware_FailedRequest(t *testing.T) {
	ad := NewActivityDashboard(100)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ad.Middleware())
	router.POST("/api/test", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "error")
	})

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.Header.Set("X-User", "testuser")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	activities := ad.GetActivities(10)
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity for failed POST, got %d", len(activities))
	}
	if len(activities[0].Details) == 0 {
		t.Error("expected non-empty details for failed request")
	}
}

func TestActivityDashboard_Middleware_AnonymousUser(t *testing.T) {
	ad := NewActivityDashboard(100)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ad.Middleware())
	router.POST("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("POST", "/api/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	activities := ad.GetActivities(10)
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}
	if activities[0].User != "anonymous" {
		t.Errorf("expected 'anonymous' user, got '%s'", activities[0].User)
	}
}

func TestActivityDashboard_MaxEntries(t *testing.T) {
	ad := NewActivityDashboard(5)

	for i := 0; i < 10; i++ {
		ad.Record(Activity{User: "user1"})
	}

	activities := ad.GetActivities(100)
	if len(activities) != 5 {
		t.Errorf("expected 5 activities (maxEntries), got %d", len(activities))
	}
}

func TestActivityDashboard_DefaultMaxEntries(t *testing.T) {
	ad := NewActivityDashboard(0)

	if ad.maxEntries != 500 {
		t.Errorf("expected default maxEntries 500, got %d", ad.maxEntries)
	}
}

func TestActivityDashboard_listHandler(t *testing.T) {
	ad := NewActivityDashboard(100)
	ad.Record(Activity{User: "user1"})

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/admin/activity?limit=5", nil)

	ad.listHandler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestActivityDashboard_userHandler(t *testing.T) {
	ad := NewActivityDashboard(100)
	ad.Record(Activity{User: "user1"})

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/admin/users/user1/activity?limit=5", nil)
	c.Params = gin.Params{{Key: "user", Value: "user1"}}

	ad.userHandler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestNodeWebhook_TriggerEvent(t *testing.T) {
	nw := NewNodeWebhook(3)

	nw.RegisterWebhook("node1", WebhookConfig{
		URL:    "http://nonexistent.invalid/webhook",
		Events: []string{"node.registered"},
	})

	nw.TriggerEvent("node1", "node.registered", "online")
	nw.TriggerEvent("node1", "other.event", "state")
}

func TestNodeWebhook_GetAllWebhooks_Empty(t *testing.T) {
	nw := NewNodeWebhook(3)

	all := nw.GetAllWebhooks()
	if len(all) != 0 {
		t.Errorf("expected empty webhooks map, got %d", len(all))
	}
}

func TestNodeWebhook_GetAllWebhooks_AfterRegister(t *testing.T) {
	nw := NewNodeWebhook(3)

	nw.RegisterWebhook("node1", WebhookConfig{URL: "http://example.com", Events: []string{"test"}})
	nw.RegisterWebhook("node2", WebhookConfig{URL: "http://example.com", Events: []string{"test"}})

	all := nw.GetAllWebhooks()
	if len(all) != 2 {
		t.Errorf("expected 2 webhook entries, got %d", len(all))
	}
}

func TestNodeWebhook_ClearWebhooks(t *testing.T) {
	nw := NewNodeWebhook(3)

	nw.RegisterWebhook("node1", WebhookConfig{URL: "http://example.com"})
	nw.ClearWebhooks("node1")

	if len(nw.GetWebhooks("node1")) != 0 {
		t.Error("expected empty webhooks after ClearWebhooks")
	}
}

func TestNodeWebhook_RegisterWebhook_DuplicateURLs(t *testing.T) {
	nw := NewNodeWebhook(3)

	nw.RegisterWebhook("node1", WebhookConfig{URL: "http://example.com", Events: []string{"a"}})
	nw.RegisterWebhook("node1", WebhookConfig{URL: "http://example.com", Events: []string{"b"}})

	webhooks := nw.GetWebhooks("node1")
	if len(webhooks) != 2 {
		t.Errorf("expected 2 webhooks, got %d", len(webhooks))
	}
}

func TestScalingPolicyManager_AddPolicy(t *testing.T) {
	spm := NewScalingPolicyManager()

	policy := &ScalingPolicy{
		Name:      "custom-policy",
		Enabled:   true,
		Metric:    "critical_nodes",
		Threshold: 5,
		Comparison: ">",
		Action:    ActionScaleUp,
		Cooldown:  time.Minute,
	}
	spm.AddPolicy(policy)

	if spm.GetPolicy(policy.ID) == nil {
		t.Error("expected to retrieve added policy by ID")
	}
}

func TestScalingPolicyManager_ListPolicies(t *testing.T) {
	spm := NewScalingPolicyManager()

	policies := spm.ListPolicies()
	if len(policies) < 2 {
		t.Errorf("expected at least 2 default policies, got %d", len(policies))
	}
}

func TestScalingPolicyManager_AddPolicy_NoDuplicateID(t *testing.T) {
	spm := NewScalingPolicyManager()

	policy := &ScalingPolicy{
		Name:      "custom",
		Enabled:   true,
		Metric:    "unknown_metric",
		Threshold: 0,
		Cooldown:  time.Minute,
	}
	spm.AddPolicy(policy)
	spm.AddPolicy(policy)

	if spm.GetPolicy(policy.ID) == nil {
		t.Error("expected to retrieve policy")
	}
}

func TestScalingPolicyManager_MarshalPolicies(t *testing.T) {
	spm := NewScalingPolicyManager()

	data, err := spm.MarshalPolicies()
	if err != nil {
		t.Fatalf("MarshalPolicies failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON output")
	}
}

func TestNewTracer_Disabled(t *testing.T) {
	tracer, err := InitTracing("test", false)
	if err != nil {
		t.Fatalf("InitTracing failed: %v", err)
	}
	if tracer != nil {
		t.Error("expected nil tracer when disabled")
	}
}

func TestNewPrometheusPusher_Disabled(t *testing.T) {
	cfg := &config.PrometheusConfig{
		Enabled: false,
	}
	pusher := NewPrometheusPusher(cfg)
	if pusher == nil {
		t.Fatal("expected non-nil PrometheusPusher")
	}
	if pusher.enabled {
		t.Error("expected pusher to be disabled")
	}

	pusher.Start()
	pusher.Stop()
}

func TestNewAuditLogger_NilRedis(t *testing.T) {
	logger, err := NewAuditLogger("", nil)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer logger.Close()

	if logger.redis != nil {
		t.Error("expected nil redis for no redis config")
	}
}

func TestAuditLogger_LogEntry(t *testing.T) {
	logger, _ := NewAuditLogger("", nil)
	defer logger.Close()

	logger.Log(AuditEntry{
		Action:   "test_action",
		Resource: "test_resource",
		UserIP:   "127.0.0.1",
		Method:   "GET",
		Path:     "/test",
		Success:  true,
	})
}

func TestAuditLogger_GetEntries_NilRedis(t *testing.T) {
	logger, _ := NewAuditLogger("", nil)
	defer logger.Close()

	_, err := logger.GetEntries("2024-01-01", "test", 10)
	if err == nil {
		t.Error("expected error for GetEntries with nil redis")
	}
}

func TestAuditLogger_AuditMiddleware_GetRequest(t *testing.T) {
	logger, _ := NewAuditLogger("", nil)
	defer logger.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(logger.AuditMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuditLogger_AuditMiddleware_PostRequest(t *testing.T) {
	logger, _ := NewAuditLogger("", nil)
	defer logger.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(logger.AuditMiddleware())
	router.POST("/api/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("POST", "/api/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuditLogger_AuditMiddleware_FailedRequest(t *testing.T) {
	logger, _ := NewAuditLogger("", nil)
	defer logger.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(logger.AuditMiddleware())
	router.POST("/api/test", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "error")
	})

	req := httptest.NewRequest("POST", "/api/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestAuditLogger_FileCreation(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath, nil)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}

	logger.Log(AuditEntry{Action: "test"})

	logger.Close()

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("expected audit log file to be created")
	}
}

func TestLogger_Levels(t *testing.T) {
	l := &Logger{
		output:   os.Stdout,
		minLevel: LevelDebug,
	}

	l.Info("info message", nil)
	l.Warn("warn message", nil)
	l.Error("error message", nil)
	l.Debug("debug message", nil)
}

func TestLogger_LevelFiltering(t *testing.T) {
	l := &Logger{
		output:   os.Stdout,
		minLevel: LevelWarn,
	}

	l.Debug("debug message", nil)
	l.Info("info message", nil)
}

func TestNewResponseCompressor_Disabled(t *testing.T) {
	rc := NewResponseCompressor(false, 0)
	if rc == nil {
		t.Fatal("expected non-nil ResponseCompressor")
	}
	if rc.enabled {
		t.Error("expected compressor to be disabled")
	}
}

func TestNewResponseCompressor_Enabled(t *testing.T) {
	rc := NewResponseCompressor(true, 6)
	if rc == nil {
		t.Fatal("expected non-nil ResponseCompressor when enabled")
	}
}

func TestNewRequestIDGenerator(t *testing.T) {
	rig := NewRequestIDGenerator("REQ", "uuid", nil)
	if rig == nil {
		t.Fatal("expected non-nil RequestIDGenerator")
	}

	id := rig.Generate()
	if id == "" {
		t.Error("expected non-empty request ID")
	}

	uuidID := rig.GenerateUUID()
	if uuidID == "" {
		t.Error("expected non-empty UUID")
	}
}

func TestRequestIDConfig_Snapshot(t *testing.T) {
	cfg := NewRequestIDConfig("test", "counter")
	snap := cfg.Snapshot()

	if snap["prefix"] != "test" {
		t.Errorf("expected prefix 'test', got '%s'", snap["prefix"])
	}
}

func TestRequestIDConfig_SetFormat_Invalid(t *testing.T) {
	cfg := NewRequestIDConfig("test", "counter")

	if cfg.SetFormat("invalid-format") {
		t.Error("expected false for invalid format")
	}
}

func TestRequestIDGenerator_GetTraceID(t *testing.T) {
	rig := NewRequestIDGenerator("REQ", "uuid", nil)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	traceID := rig.GetTraceID(c)
	if traceID == "" {
		t.Error("expected non-empty trace ID")
	}
}
