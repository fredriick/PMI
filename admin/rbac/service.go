package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Role string

const (
	RoleSuperAdmin Role = "superadmin"
	RoleAdmin      Role = "admin"
	RoleOperator   Role = "operator"
	RoleViewer     Role = "viewer"
)

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Roles     []Role    `json:"roles"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LastLogin time.Time `json:"last_login,omitempty"`
	Active    bool      `json:"active"`
}

type Permission struct {
	Resource string   `json:"resource"`
	Actions  []string `json:"actions"`
}

type RBACService struct {
	users map[string]*User
	perms map[Role][]Permission
	mu    sync.RWMutex
	cfg   *Config
}

type Config struct {
	DefaultRole      Role `mapstructure:"default_role"`
	MinPasswordLen   int  `mapstructure:"min_password_len"`
	MaxLoginAttempts int  `mapstructure:"max_login_attempts"`
	LockoutDuration  int  `mapstructure:"lockout_duration_minutes"`
}

func NewRBACService(cfg *Config) *RBACService {
	if cfg == nil {
		cfg = &Config{
			DefaultRole:      RoleViewer,
			MinPasswordLen:   8,
			MaxLoginAttempts: 5,
			LockoutDuration:  15,
		}
	}

	svc := &RBACService{
		users: make(map[string]*User),
		perms: make(map[Role][]Permission),
		cfg:   cfg,
	}

	svc.initPermissions()
	return svc
}

func (s *RBACService) initPermissions() {
	s.perms[RoleSuperAdmin] = []Permission{
		{Resource: "*", Actions: []string{"*"}},
	}

	s.perms[RoleAdmin] = []Permission{
		{Resource: "nodes", Actions: []string{"*"}},
		{Resource: "keys", Actions: []string{"*"}},
		{Resource: "sessions", Actions: []string{"*"}},
		{Resource: "cooldowns", Actions: []string{"*"}},
		{Resource: "subnets", Actions: []string{"*"}},
		{Resource: "capacity", Actions: []string{"read", "write"}},
		{Resource: "audit", Actions: []string{"read"}},
	}

	s.perms[RoleOperator] = []Permission{
		{Resource: "nodes", Actions: []string{"read", "write"}},
		{Resource: "keys", Actions: []string{"read", "write"}},
		{Resource: "sessions", Actions: []string{"read", "delete"}},
		{Resource: "cooldowns", Actions: []string{"read"}},
		{Resource: "capacity", Actions: []string{"read"}},
	}

	s.perms[RoleViewer] = []Permission{
		{Resource: "nodes", Actions: []string{"read"}},
		{Resource: "keys", Actions: []string{"read"}},
		{Resource: "capacity", Actions: []string{"read"}},
	}
}

func (s *RBACService) CreateUser(username, password, email string, roles []Role) (*User, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password required")
	}

	if len(password) < s.cfg.MinPasswordLen {
		return nil, fmt.Errorf("password must be at least %d characters", s.cfg.MinPasswordLen)
	}

	if len(roles) == 0 {
		roles = []Role{s.cfg.DefaultRole}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; exists {
		return nil, fmt.Errorf("user already exists")
	}

	user := &User{
		ID:        generateID(),
		Username:  username,
		Password:  hashPassword(password),
		Roles:     roles,
		Email:     email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	s.users[username] = user
	return user, nil
}

func (s *RBACService) GetUser(username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[username]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (s *RBACService) ListUsers() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return users
}

func (s *RBACService) UpdateUserRoles(username string, roles []Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[username]
	if !ok {
		return fmt.Errorf("user not found")
	}

	user.Roles = roles
	user.UpdatedAt = time.Now()
	return nil
}

func (s *RBACService) DeactivateUser(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[username]
	if !ok {
		return fmt.Errorf("user not found")
	}

	user.Active = false
	user.UpdatedAt = time.Now()
	return nil
}

func (s *RBACService) Authenticate(username, password string) (*User, error) {
	s.mu.RLock()
	user, ok := s.users[username]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.Active {
		return nil, fmt.Errorf("account deactivated")
	}

	if !verifyPassword(password, user.Password) {
		return nil, fmt.Errorf("invalid credentials")
	}

	s.mu.Lock()
	user.LastLogin = time.Now()
	s.mu.Unlock()

	return user, nil
}

func (s *RBACService) HasPermission(user *User, resource, action string) bool {
	for _, role := range user.Roles {
		perms, ok := s.perms[role]
		if !ok {
			continue
		}

		for _, perm := range perms {
			if perm.Resource == "*" || perm.Resource == resource {
				for _, a := range perm.Actions {
					if a == "*" || a == action {
						return true
					}
				}
			}
		}
	}
	return false
}

func (s *RBACService) RequirePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userVal, exists := c.Get("rbac_user")
		if !exists {
			c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
			return
		}

		user, ok := userVal.(*User)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid user context"})
			return
		}

		if !s.HasPermission(user, resource, action) {
			c.AbortWithStatusJSON(403, gin.H{"error": "Insufficient permissions"})
			return
		}

		c.Next()
	}
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashPassword(password string) string {
	b := []byte(password)
	for i := range b {
		b[i] ^= 0x5A
	}
	return hex.EncodeToString(b)
}

func verifyPassword(password, hash string) bool {
	return hashPassword(password) == hash
}

func (s *RBACService) GetConfig() *Config {
	return s.cfg
}

func (s *RBACService) MarshalJSON() (json.RawMessage, error) {
	type Alias RBACService
	return json.Marshal(&struct {
		Alias
		UserCount int `json:"user_count"`
	}{
		Alias:     Alias(*s),
		UserCount: len(s.users),
	})
}

type RBACMiddleware struct {
	svc *RBACService
}

func NewRBACMiddleware(svc *RBACService) *RBACMiddleware {
	return &RBACMiddleware{svc: svc}
}

func (m *RBACMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.GetHeader("X-Admin-User")
		password := c.GetHeader("X-Admin-Password")

		if username == "" || password == "" {
			auth := c.GetHeader("Authorization")
			if auth != "" {
				username = "admin"
				password = auth
			}
		}

		if username == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "Missing credentials"})
			return
		}

		user, err := m.svc.Authenticate(username, password)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": err.Error()})
			return
		}

		c.Set("rbac_user", user)
		c.Set("rbac_username", user.Username)
		c.Set("rbac_roles", user.Roles)

		c.Next()
	}
}

func GetRBACUser(c *gin.Context) *User {
	if val, ok := c.Get("rbac_user"); ok {
		return val.(*User)
	}
	return nil
}

func GetRBACRoles(c *gin.Context) []Role {
	if val, ok := c.Get("rbac_roles"); ok {
		return val.([]Role)
	}
	return nil
}

type contextKey string

const ctxKey contextKey = "rbac_service"

func WithContext(ctx context.Context, svc *RBACService) context.Context {
	return context.WithValue(ctx, ctxKey, svc)
}

func FromContext(ctx context.Context) *RBACService {
	if val := ctx.Value(ctxKey); val != nil {
		return val.(*RBACService)
	}
	return nil
}
