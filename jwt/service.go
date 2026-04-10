package jwt

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type Claims struct {
	jwt.RegisteredClaims
	UserID  string   `json:"user_id"`
	Roles   []string `json:"roles"`
	APIKeys []string `json:"api_keys,omitempty"`
	NodeID  string   `json:"node_id,omitempty"`
	Issuer  string   `json:"iss"`
}

type JWTService struct {
	secretKey  string
	expiration time.Duration
	issuer     string
}

func NewJWTService(secret string, expiration time.Duration) *JWTService {
	if secret == "" {
		gen := make([]byte, 32)
		rand.Read(gen)
		secret = hex.EncodeToString(gen)
	}

	return &JWTService{
		secretKey:  secret,
		expiration: expiration,
		issuer:     "proxymesh",
	}
}

func (s *JWTService) GenerateToken(userID string, roles []string, apiKeys []string, nodeID string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    s.issuer,
			ID:        generateTokenID(),
		},
		UserID:  userID,
		Roles:   roles,
		APIKeys: apiKeys,
		NodeID:  nodeID,
		Issuer:  s.issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		if strings.Contains(err.Error(), "token is expired") {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *JWTService) RefreshToken(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	return s.GenerateToken(claims.UserID, claims.Roles, claims.APIKeys, claims.NodeID)
}

func (s *JWTService) GenerateAPIKeyToken(apiKey string, userID string) (string, error) {
	return s.GenerateToken(userID, []string{"api"}, []string{apiKey}, "")
}

func (s *JWTService) GenerateAdminToken(userID string, roles []string) (string, error) {
	return s.GenerateToken(userID, roles, nil, "")
}

func (s *JWTService) GeneratePeerToken(nodeID string) (string, error) {
	return s.GenerateToken(nodeID, []string{"peer"}, nil, nodeID)
}

func (s *JWTService) GetSecretKey() string {
	return s.secretKey
}

func (s *JWTService) GetExpiration() time.Duration {
	return s.expiration
}

func generateTokenID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type TokenInfo struct {
	UserID   string    `json:"user_id"`
	Roles    []string  `json:"roles"`
	IssuedAt time.Time `json:"issued_at"`
	Expires  time.Time `json:"expires"`
	NodeID   string    `json:"node_id,omitempty"`
}

func (s *JWTService) ParseTokenInfo(tokenString string) (*TokenInfo, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	return &TokenInfo{
		UserID:   claims.UserID,
		Roles:    claims.Roles,
		IssuedAt: claims.IssuedAt.Time,
		Expires:  claims.ExpiresAt.Time,
		NodeID:   claims.NodeID,
	}, nil
}

func (s *JWTService) IsAdmin(claims *Claims) bool {
	for _, role := range claims.Roles {
		if role == "superadmin" || role == "admin" || role == "operator" {
			return true
		}
	}
	return false
}

func (s *JWTService) HasRole(claims *Claims, role string) bool {
	for _, r := range claims.Roles {
		if r == role {
			return true
		}
	}
	return false
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

func (s *JWTService) GenerateTokenResponse(userID string, roles []string, apiKeys []string, nodeID string) (*TokenResponse, error) {
	accessToken, err := s.GenerateToken(userID, roles, apiKeys, nodeID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.GenerateToken(userID, roles, apiKeys, nodeID)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.expiration.Seconds()),
	}, nil
}

func (s *JWTService) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"expiration": s.expiration.String(),
		"issuer":     s.issuer,
	})
}
