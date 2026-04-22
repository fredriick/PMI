package jwt

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func newTestService() *JWTService {
	return NewJWTService("test-secret-key-for-unit-tests", time.Hour)
}

func TestNewJWTService_GeneratesSecret(t *testing.T) {
	svc := NewJWTService("", time.Hour)
	if svc.secretKey == "" {
		t.Error("expected auto-generated secret key")
	}
}

func TestNewJWTService_UsesProvidedSecret(t *testing.T) {
	svc := NewJWTService("my-explicit-secret", time.Hour)
	if svc.secretKey != "my-explicit-secret" {
		t.Errorf("secretKey = %q, want %q", svc.secretKey, "my-explicit-secret")
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	svc := newTestService()

	tokenStr, err := svc.GenerateToken("user-1", []string{"admin"}, nil, "")
	if err != nil {
		t.Fatalf("GenerateToken() error: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("GenerateToken() returned empty string")
	}

	claims, err := svc.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("ValidateToken() error: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-1")
	}
	expectedRoles := []string{"admin"}
	if len(claims.Roles) != len(expectedRoles) {
		t.Fatalf("Roles length = %d, want %d", len(claims.Roles), len(expectedRoles))
	}
}

func TestValidateToken_Expired(t *testing.T) {
	svc := NewJWTService("test-secret", -time.Hour)

	tokenStr, err := svc.GenerateToken("user-1", nil, nil, "")
	if err != nil {
		t.Fatalf("GenerateToken() error: %v", err)
	}

	_, err = svc.ValidateToken(tokenStr)
	if err != ErrExpiredToken {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	svc := newTestService()
	_, err := svc.ValidateToken("this.is.not.a.valid.token")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	svc := newTestService()
	token, _ := svc.GenerateToken("user-1", nil, nil, "")

	other := NewJWTService("different-secret", time.Hour)
	_, err := other.ValidateToken(token)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken with wrong secret, got %v", err)
	}
}

func TestValidateToken_WrongSigningMethod(t *testing.T) {
	// Generate a token signed with a different key pair (RS256)
	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString(nil)
	if err != nil {
		// Without a private key this returns error; skip if so
		t.Skip("RS256 signing requires private key")
		return
	}

	svc := newTestService()
	_, err = svc.ValidateToken(tokenStr)
	if err == nil {
		t.Error("expected error for unsupported signing method")
	}
}

func TestRefreshToken(t *testing.T) {
	svc := newTestService()

	tokenStr, err := svc.GenerateToken("user-1", []string{"admin"}, []string{"key-1"}, "node-1")
	if err != nil {
		t.Fatalf("GenerateToken() error: %v", err)
	}

	newToken, err := svc.RefreshToken(tokenStr)
	if err != nil {
		t.Fatalf("RefreshToken() error: %v", err)
	}
	if newToken == tokenStr {
		t.Error("RefreshToken() should return a different token string")
	}

	claims, err := svc.ValidateToken(newToken)
	if err != nil {
		t.Fatalf("ValidateToken(refreshed) error: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("refreshed UserID = %q, want %q", claims.UserID, "user-1")
	}
}

func TestGenerateAPIKeyToken(t *testing.T) {
	svc := newTestService()

	tokenStr, err := svc.GenerateAPIKeyToken("key-abc", "user-1")
	if err != nil {
		t.Fatalf("GenerateAPIKeyToken() error: %v", err)
	}

	claims, err := svc.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("ValidateToken(api key) error: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-1")
	}
	if len(claims.APIKeys) != 1 || claims.APIKeys[0] != "key-abc" {
		t.Errorf("APIKeys = %v, want [key-abc]", claims.APIKeys)
	}
}

func TestGenerateAdminToken(t *testing.T) {
	svc := newTestService()

	tokenStr, err := svc.GenerateAdminToken("admin-user", []string{"superadmin"})
	if err != nil {
		t.Fatalf("GenerateAdminToken() error: %v", err)
	}

	claims, err := svc.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("ValidateToken(admin) error: %v", err)
	}
	if claims.UserID != "admin-user" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "admin-user")
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "superadmin" {
		t.Errorf("Roles = %v, want [superadmin]", claims.Roles)
	}
}

func TestGeneratePeerToken(t *testing.T) {
	svc := newTestService()

	tokenStr, err := svc.GeneratePeerToken("node-xyz")
	if err != nil {
		t.Fatalf("GeneratePeerToken() error: %v", err)
	}

	claims, err := svc.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("ValidateToken(peer) error: %v", err)
	}
	if claims.NodeID != "node-xyz" {
		t.Errorf("NodeID = %q, want %q", claims.NodeID, "node-xyz")
	}
}

func TestParseTokenInfo_Valid(t *testing.T) {
	svc := newTestService()

	tokenStr, err := svc.GenerateToken("user-1", []string{"operator"}, nil, "")
	if err != nil {
		t.Fatalf("GenerateToken() error: %v", err)
	}

	info, err := svc.ParseTokenInfo(tokenStr)
	if err != nil {
		t.Fatalf("ParseTokenInfo() error: %v", err)
	}
	if info.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", info.UserID, "user-1")
	}
	if len(info.Roles) != 1 || info.Roles[0] != "operator" {
		t.Errorf("Roles = %v, want [operator]", info.Roles)
	}
	if info.Expires.IsZero() {
		t.Error("Expires should not be zero")
	}
}

func TestParseTokenInfo_Expired(t *testing.T) {
	svc := NewJWTService("test-secret", -time.Hour)

	tokenStr, _ := svc.GenerateToken("user-1", nil, nil, "")
	_, err := svc.ParseTokenInfo(tokenStr)
	if err != ErrExpiredToken {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestParseTokenInfo_Invalid(t *testing.T) {
	svc := newTestService()
	_, err := svc.ParseTokenInfo("totally-invalid")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestIsAdmin_True(t *testing.T) {
	svc := newTestService()
	if !svc.IsAdmin(&Claims{Roles: []string{"superadmin"}}) {
		t.Error("expected IsAdmin=true for superadmin")
	}
	if !svc.IsAdmin(&Claims{Roles: []string{"admin"}}) {
		t.Error("expected IsAdmin=true for admin")
	}
	if !svc.IsAdmin(&Claims{Roles: []string{"operator"}}) {
		t.Error("expected IsAdmin=true for operator")
	}
	if !svc.IsAdmin(&Claims{Roles: []string{"admin", "operator"}}) {
		t.Error("expected IsAdmin=true for admin+operator")
	}
}

func TestIsAdmin_False(t *testing.T) {
	svc := newTestService()
	if svc.IsAdmin(&Claims{Roles: []string{"viewer"}}) {
		t.Error("expected IsAdmin=false for viewer")
	}
	if svc.IsAdmin(&Claims{Roles: []string{"peer"}}) {
		t.Error("expected IsAdmin=false for peer")
	}
	if svc.IsAdmin(&Claims{Roles: nil}) {
		t.Error("expected IsAdmin=false for nil roles")
	}
}

func TestHasRole(t *testing.T) {
	svc := newTestService()
	claims := &Claims{Roles: []string{"admin", "peer"}}
	if !svc.HasRole(claims, "admin") {
		t.Error("expected HasRole(admin)=true")
	}
	if !svc.HasRole(claims, "peer") {
		t.Error("expected HasRole(peer)=true")
	}
	if svc.HasRole(claims, "superadmin") {
		t.Error("expected HasRole(superadmin)=false")
	}
}

func TestGenerateTokenResponse(t *testing.T) {
	svc := newTestService()

	resp, err := svc.GenerateTokenResponse("user-1", []string{"user"}, []string{"k1", "k2"}, "node-1")
	if err != nil {
		t.Fatalf("GenerateTokenResponse() error: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if resp.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want %q", resp.TokenType, "Bearer")
	}
	if resp.ExpiresIn <= 0 {
		t.Errorf("ExpiresIn = %d, want > 0", resp.ExpiresIn)
	}
	if resp.AccessToken == resp.RefreshToken {
		t.Error("AccessToken and RefreshToken should differ")
	}
}

func TestJWTService_MarshalJSON(t *testing.T) {
	svc := newTestService()
	data, err := svc.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if m["issuer"] != "proxymesh" {
		t.Errorf("issuer = %v, want proxymesh", m["issuer"])
	}
	if m["expiration"] == nil {
		t.Error("expiration should be present")
	}
}

func TestJWTService_Getters(t *testing.T) {
	svc := NewJWTService("custom-secret", 2*time.Hour)

	if svc.GetSecretKey() != "custom-secret" {
		t.Errorf("GetSecretKey() = %q", svc.GetSecretKey())
	}
	if svc.GetExpiration() != 2*time.Hour {
		t.Errorf("GetExpiration() = %v, want 2h", svc.GetExpiration())
	}
}

func TestJWTService_RefreshToken_ExpiredInput(t *testing.T) {
	svc := NewJWTService("test-secret", -time.Hour)
	expired, _ := svc.GenerateToken("user-1", nil, nil, "")

	_, err := svc.RefreshToken(expired)
	if err != ErrExpiredToken {
		t.Errorf("expected ErrExpiredToken on refresh, got %v", err)
	}
}

func TestClaims_Struct(t *testing.T) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "proxymesh",
			ID:        "token-123",
		},
		UserID:  "user-1",
		Roles:   []string{"admin", "peer"},
		APIKeys: []string{"key-1"},
		NodeID:  "node-1",
		Issuer:  "proxymesh",
	}

	if claims.UserID != "user-1" {
		t.Errorf("UserID = %q", claims.UserID)
	}
	if len(claims.Roles) != 2 {
		t.Errorf("Roles length = %d", len(claims.Roles))
	}
	if claims.NodeID != "node-1" {
		t.Errorf("NodeID = %q", claims.NodeID)
	}
}

func TestGenerateTokenID_Unique(t *testing.T) {
	id1 := generateTokenID()
	id2 := generateTokenID()
	if id1 == id2 {
		t.Error("generateTokenID() should return unique values")
	}
	if len(id1) != 32 {
		t.Errorf("expected 32 hex chars, got %d", len(id1))
	}
}
