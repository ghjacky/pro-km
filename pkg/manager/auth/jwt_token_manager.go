package auth

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// jwtTokenManager manage jwt token
type jwtTokenManager struct {
	secret   string
	tokenTTL time.Duration
}

// NewJwtTokenManager build a jwt token manager
func NewJwtTokenManager(secret string, tokenTTL time.Duration) TokenManager {
	return &jwtTokenManager{
		secret:   secret,
		tokenTTL: tokenTTL,
	}
}

// Generate create a jwt token encoded user info
func (jtm *jwtTokenManager) Generate(user User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"time":  fmt.Sprintf("%d", time.Now().Unix()),
		"nonce": fmt.Sprintf("%d", rand.Int31()),
		"user":  user,
	})
	return token.SignedString([]byte(jtm.secret))
}

// Decrypt parse a jwtToken to get user info
func (jtm *jwtTokenManager) Decrypt(jwtToken string) (*User, error) {
	user := &User{}
	_, err := jwt.Parse(jwtToken, func(t *jwt.Token) (interface{}, error) {
		var err error

		claims, ok := t.Claims.(jwt.MapClaims)
		if !ok {
			return nil, fmt.Errorf("unexpected claims type")
		}

		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}

		timeStampString, ok := claims["time"].(string)
		if !ok {
			return nil, fmt.Errorf("unexpected timeStamp format, it should be string")
		}
		timeStamp, err := strconv.Atoi(timeStampString)
		if err != nil {
			return nil, fmt.Errorf("unexpected timeStamp length, it should be int64")
		}

		if time.Now().Unix()-int64(timeStamp) > int64(jtm.tokenTTL.Seconds()) {
			return nil, fmt.Errorf("token expired")
		}

		bytes, err := json.Marshal(claims["user"])
		if err != nil {
			return nil, fmt.Errorf("marshal failed")
		}

		err = json.Unmarshal(bytes, user)
		return []byte(jtm.secret), err
	})

	if err != nil {
		return nil, err
	}

	return user, err
}

// SetTokenTTL implements token manager interface.
func (jtm *jwtTokenManager) SetTokenTTL(ttl time.Duration) {
	if ttl < 0 {
		ttl = 0
	}
	jtm.tokenTTL = ttl * time.Second
}
