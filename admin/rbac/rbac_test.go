package admin

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// freshSvc mirrors newTestSvc but always returns a brand-new instance.
func freshSvc() *RBACService {
	return NewRBACService(&Config{
		DefaultRole:      RoleViewer,
		MinPasswordLen:   3,
		MaxLoginAttempts: 10,
		LockoutDuration:  15,
	})
}

// ── Constructor / init ──────────────────────────────────────────────────────

func TestNewRBACService_NilConfig(t *testing.T) {
	svc := NewRBACService(nil)
	if svc == nil {
		t.Fatal("returned nil service")
	}
	if svc.cfg.DefaultRole != RoleViewer {
		t.Errorf("DefaultRole = %q, want %q", svc.cfg.DefaultRole, RoleViewer)
	}
	if svc.cfg.MinPasswordLen != 8 {
		t.Errorf("MinPasswordLen = %d, want 8", svc.cfg.MinPasswordLen)
	}
}

func TestNewRBACService_CustomConfig(t *testing.T) {
	cfg := &Config{DefaultRole: RoleAdmin, MinPasswordLen: 4}
	svc := NewRBACService(cfg)
	if svc.cfg.DefaultRole != RoleAdmin {
		t.Errorf("DefaultRole = %q, want %q", svc.cfg.DefaultRole, RoleAdmin)
	}
	if svc.cfg.MinPasswordLen != 4 {
		t.Errorf("MinPasswordLen = %d, want 4", svc.cfg.MinPasswordLen)
	}
}

// ── Password helpers ─────────────────────────────────────────────────────────

func TestPasswordHashAndVerify(t *testing.T) {
	pw := "s3cret!"
	h := hashPassword(pw)
	if h == pw {
		t.Error("hashPassword should transform the string")
	}
	if !verifyPassword(pw, h) {
		t.Error("verifyPassword should match correct password")
	}
	if verifyPassword("wrong", h) {
		t.Error("verifyPassword should not match wrong password")
	}
}

// ── CreateUser ──────────────────────────────────────────────────────────────

func TestCreateUser_Success(t *testing.T) {
	svc := freshSvc()
	u, err := svc.CreateUser("alice", "pw123", "alice@example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Username != "alice" {
		t.Errorf("Username = %q", u.Username)
	}
	if u.Email != "alice@example.com" {
		t.Errorf("Email = %q", u.Email)
	}
	if u.ID == "" {
		t.Error("ID must be set")
	}
	if !u.Active {
		t.Error("new user must be active")
	}
	if len(u.Roles) != 1 || u.Roles[0] != RoleViewer {
		t.Errorf("Roles = %v, want [viewer]", u.Roles)
	}
}

func TestCreateUser_MissingUsername(t *testing.T) {
	svc := freshSvc()
	_, err := svc.CreateUser("", "pw", "", nil)
	if err == nil {
		t.Error("expected error for empty username")
	}
}

func TestCreateUser_MissingPassword(t *testing.T) {
	svc := freshSvc()
	_, err := svc.CreateUser("bob", "", "", nil)
	if err == nil {
		t.Error("expected error for empty password")
	}
}

func TestCreateUser_PasswordTooShort(t *testing.T) {
	svc := NewRBACService(&Config{MinPasswordLen: 6})
	_, err := svc.CreateUser("carol", "abc", "", nil)
	if err == nil {
		t.Error("expected error for short password")
	}
	if !strings.Contains(err.Error(), "at least 6") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCreateUser_DefaultRoleApplied(t *testing.T) {
	svc := freshSvc()
	u, err := svc.CreateUser("dave", "mypass", "", nil) // no roles provided → default role
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(u.Roles) != 1 || u.Roles[0] != RoleViewer {
		t.Errorf("Roles = %v, want [%s]", u.Roles, RoleViewer)
	}
}

func TestCreateUser_CustomRoles(t *testing.T) {
	svc := freshSvc()
	u, err := svc.CreateUser("eve", "mypass", "", []Role{RoleOperator})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(u.Roles) != 1 || u.Roles[0] != RoleOperator {
		t.Errorf("Roles = %v, want [operator]", u.Roles)
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	svc := freshSvc()
	svc.CreateUser("frank", "mypass", "", nil)
	_, err := svc.CreateUser("frank", "mypass2", "", nil)
	if err == nil {
		t.Fatal("expected 'user already exists' error on duplicate username")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ── GetUser / ListUsers ──────────────────────────────────────────────────────

func TestGetUser_Found(t *testing.T) {
	svc := freshSvc()
	svc.CreateUser("grace", "mypass", "g@e.com", nil)
	u, err := svc.GetUser("grace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Email != "g@e.com" {
		t.Errorf("Email = %q", u.Email)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	svc := freshSvc()
	_, err := svc.GetUser("ghost")
	if err == nil {
		t.Error("expected 'user not found' error")
	}
}

func TestListUsers(t *testing.T) {
	svc := freshSvc()
	svc.CreateUser("u1", "mypass", "", nil)
	svc.CreateUser("u2", "mypass", "", nil)
	list := svc.ListUsers()
	if len(list) != 2 {
		t.Fatalf("ListUsers() = %d, want 2", len(list))
	}
}

// ── UpdateUserRoles ─────────────────────────────────────────────────────────

func TestUpdateUserRoles(t *testing.T) {
	svc := freshSvc()
	svc.CreateUser("henry", "mypass", "", []Role{RoleViewer})
	err := svc.UpdateUserRoles("henry", []Role{RoleAdmin, RoleOperator})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	u, _ := svc.GetUser("henry")
	if len(u.Roles) != 2 || u.Roles[0] != RoleAdmin || u.Roles[1] != RoleOperator {
		t.Errorf("Roles = %v", u.Roles)
	}
}

func TestUpdateUserRoles_NotFound(t *testing.T) {
	svc := freshSvc()
	err := svc.UpdateUserRoles("nobody", []Role{RoleAdmin})
	if err == nil {
		t.Error("expected 'user not found' error")
	}
}

// ── DeactivateUser ──────────────────────────────────────────────────────────

func TestDeactivateUser(t *testing.T) {
	svc := freshSvc()
	svc.CreateUser("ian", "mypass", "", nil)
	err := svc.DeactivateUser("ian")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	u, _ := svc.GetUser("ian")
	if u.Active {
		t.Error("user must be inactive after DeactivateUser")
	}
}

func TestDeactivateUser_NotFound(t *testing.T) {
	svc := freshSvc()
	err := svc.DeactivateUser("nobody")
	if err == nil {
		t.Error("expected 'user not found' error")
	}
}

// ── Authenticate ────────────────────────────────────────────────────────────

func TestAuthenticate_Success(t *testing.T) {
	svc := freshSvc()
	svc.CreateUser("jane", "mypassword", "j@e.com", nil)
	u, err := svc.Authenticate("jane", "mypassword")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Username != "jane" {
		t.Errorf("Username = %q", u.Username)
	}
	if u.LastLogin.IsZero() {
		t.Error("LastLogin must be set after auth")
	}
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	svc := freshSvc()
	svc.CreateUser("kate", "secret", "", nil)
	_, err := svc.Authenticate("kate", "wrong")
	if err == nil {
		t.Error("expected 'invalid credentials' error")
	}
}

func TestAuthenticate_UserNotFound(t *testing.T) {
	svc := freshSvc()
	_, err := svc.Authenticate("nobody", "x")
	if err == nil {
		t.Error("expected 'invalid credentials' error")
	}
}

func TestAuthenticate_InactiveAccount(t *testing.T) {
	svc := freshSvc()
	svc.CreateUser("liam", "mypass", "", nil)
	svc.DeactivateUser("liam")

	_, err := svc.Authenticate("liam", "mypass")
	// Inactive check is before password verification, so error must mention deactivation
	if err == nil || !strings.Contains(err.Error(), "deactivated") {
		t.Errorf("expected deactivated error, got: %v", err)
	}
}

// ── HasPermission ────────────────────────────────────────────────────────────

func TestHasPermission_SuperAdminWildcard(t *testing.T) {
	svc := freshSvc()
	u := &User{Username: "sadmin", Roles: []Role{RoleSuperAdmin}}
	if !svc.HasPermission(u, "anything", "any_action") {
		t.Error("superadmin must pass any resource/action check")
	}
}

func TestHasPermission_AdminNodesWrite(t *testing.T) {
	svc := freshSvc()
	u := &User{Username: "admin", Roles: []Role{RoleAdmin}}
	if !svc.HasPermission(u, "nodes", "write") {
		t.Error("admin must allow nodes/write")
	}
	if !svc.HasPermission(u, "keys", "*") {
		t.Error("admin must allow keys/*")
	}
	if svc.HasPermission(u, "audit", "write") {
		t.Error("admin must not allow audit/write")
	}
}

func TestHasPermission_OperatorCooldownsRead(t *testing.T) {
	svc := freshSvc()
	u := &User{Username: "op", Roles: []Role{RoleOperator}}
	if !svc.HasPermission(u, "cooldowns", "read") {
		t.Error("operator must allow cooldowns/read")
	}
	if svc.HasPermission(u, "cooldowns", "write") {
		t.Error("operator must not allow cooldowns/write")
	}
	if svc.HasPermission(u, "capacity", "write") {
		t.Error("operator must not allow capacity/write")
	}
}

func TestHasPermission_ViewerNodesRead(t *testing.T) {
	svc := freshSvc()
	u := &User{Username: "viewer", Roles: []Role{RoleViewer}}
	if !svc.HasPermission(u, "nodes", "read") {
		t.Error("viewer must allow nodes/read")
	}
	if svc.HasPermission(u, "nodes", "write") {
		t.Error("viewer must not allow nodes/write")
	}
}

func TestHasPermission_NoMatch(t *testing.T) {
	svc := freshSvc()
	u := &User{Username: "nobody", Roles: []Role{RoleViewer}}
	if svc.HasPermission(u, "subnets", "admin") {
		t.Error("viewer must not allow subnets/admin")
	}
}

// ── RequirePermission middleware ─────────────────────────────────────────────

func TestRequirePermission_MissingUser(t *testing.T) {
	svc := freshSvc()
	r := gin.New()
	r.Use(svc.RequirePermission("nodes", "read"))
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Errorf("expect 401 when no rbac_user in context, got %d", rr.Code)
	}
}

func TestRequirePermission_Authorized(t *testing.T) {
	svc := freshSvc()
	u := &User{Username: "op", Roles: []Role{RoleOperator}}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("rbac_user", u); c.Next() })
	r.Use(svc.RequirePermission("sessions", "read"))
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("expect 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestRequirePermission_Forbidden(t *testing.T) {
	svc := freshSvc()
	u := &User{Username: "view", Roles: []Role{RoleViewer}}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("rbac_user", u); c.Next() })
	r.Use(svc.RequirePermission("nodes", "write"))
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != 403 {
		t.Errorf("expect 403, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ── MarshalJSON (RBAC fix regression test) ──────────────────────────────────

func TestMarshalJSON_SafeStructNoPanic(t *testing.T) {
	svc := freshSvc()
	svc.CreateUser("admin", "mypass", "", []Role{RoleSuperAdmin})

	data, err := json.Marshal(svc)
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}
	if len(data) == 0 {
		t.Error("MarshalJSON returned empty output")
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("roundtrip unmarshal error: %v", err)
	}
	if _, hasMutex := m["mu"]; hasMutex {
		t.Error("MarshalJSON output must not expose internal mutex field")
	}
}

// ── GetConfig ───────────────────────────────────────────────────────────────

func TestGetConfig(t *testing.T) {
	cfg := &Config{DefaultRole: RoleAdmin, MinPasswordLen: 4}
	svc := NewRBACService(cfg)
	got := svc.GetConfig()
	if got.DefaultRole != RoleAdmin {
		t.Errorf("DefaultRole = %q", got.DefaultRole)
	}
	if got.MinPasswordLen != 4 {
		t.Errorf("MinPasswordLen = %d", got.MinPasswordLen)
	}
}

// ── Context helpers ─────────────────────────────────────────────────────────

func TestWithContextAndFromContext(t *testing.T) {
	svc := freshSvc()
	ctx := WithContext(context.Background(), svc)
	if FromContext(ctx) != svc {
		t.Error("FromContext must return the injected service")
	}
}

// ── Role constants ──────────────────────────────────────────────────────────

func TestRoleConstants(t *testing.T) {
	if RoleSuperAdmin != "superadmin" {
		t.Errorf("RoleSuperAdmin = %q", RoleSuperAdmin)
	}
	if RoleAdmin != "admin" {
		t.Errorf("RoleAdmin = %q", RoleAdmin)
	}
	if RoleOperator != "operator" {
		t.Errorf("RoleOperator = %q", RoleOperator)
	}
	if RoleViewer != "viewer" {
		t.Errorf("RoleViewer = %q", RoleViewer)
	}
}

// ── GetRBACUser / GetRBACRoles helpers ──────────────────────────────────────

func TestGetRBACUser(t *testing.T) {
	u := &User{Username: "alice"}
	c := &gin.Context{}
	// unset key returns nil
	if got := GetRBACUser(c); got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
	c.Set("rbac_user", u)
	if got := GetRBACUser(c); got != u {
		t.Errorf("GetRBACUser = %+v, want %+v", got, u)
	}
}

func TestGetRBACRoles(t *testing.T) {
	roles := []Role{RoleOperator, RoleViewer}
	c := &gin.Context{}
	if got := GetRBACRoles(c); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
	c.Set("rbac_roles", roles)
	if got := GetRBACRoles(c); !slicesEqual(got, roles) {
		t.Errorf("GetRBACRoles = %v, want %v", got, roles)
	}
}

func TestRBACMiddleware_Authenticate(t *testing.T) {
	tcs := []struct {
		name     string
		create   string // username to pre-create, "SETUP" means create generic user
		user     string // X-Admin-User header
		password string // X-Admin-Password / Authorization value
		useAuth  bool   // send Authorization header instead of X-Admin-*
		wantCode int
	}{
		{
			name:     "missing_credentials",
			wantCode: 401,
		},
		{
			name:     "valid_headers",
			create:   "tester",
			user:     "tester",
			password: "mypass",
			wantCode: 200,
		},
		{
			name:     "wrong_password",
			create:   "tester",
			user:     "tester",
			password: "badpass",
			wantCode: 401,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			svc := freshSvc()
			if tc.create == "tester" || tc.create == "admin" {
				svc.CreateUser(tc.create, "mypass", "", []Role{RoleOperator})
			}

			r := gin.New()
			r.Use(NewRBACMiddleware(svc).Authenticate())
			r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

			req := httptest.NewRequest("GET", "/test", nil)
			if tc.useAuth {
				req.Header.Set("Authorization", tc.password)
			} else {
				if tc.user != "" {
					req.Header.Set("X-Admin-User", tc.user)
				}
				if tc.password != "" {
					req.Header.Set("X-Admin-Password", tc.password)
				}
			}

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			if rr.Code != tc.wantCode {
				t.Errorf("[%s] want %d, got %d: %s", tc.name, tc.wantCode, rr.Code, rr.Body.String())
			}
		})
	}
}

func slicesEqual(a, b []Role) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
