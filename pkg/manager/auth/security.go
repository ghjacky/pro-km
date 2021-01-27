package auth

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"code.xxxxx.cn/platform/galaxy/pkg/util/alog"
	"github.com/dgrijalva/jwt-go"
)

const (
	// JwtExpireSeconds default jwt expire time
	JwtExpireSeconds = 500
)

// GenerateJWTToken gen a jwt token
func GenerateJWTToken(id int64, secret string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"time":  fmt.Sprintf("%d", time.Now().Unix()),
		"id":    fmt.Sprintf("%d", id),
		"nonce": fmt.Sprintf("%d", rand.Int31()),
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		alog.Errorf("generate jwt token failed: %v", err)
		return ""
	}
	return tokenString
}

// ParseJWTToken decode jwt token and validate jwt token
func ParseJWTToken(token, secret string) string {
	ans := ""
	_, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		claims, ok := t.Claims.(jwt.MapClaims)
		if !ok {
			return nil, fmt.Errorf("unexpected claims type")
		}

		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", claims["id"])
		}

		timeStampString, ok := claims["time"].(string)
		if !ok {
			return nil, fmt.Errorf("unexpected timeStamp format, it should be string")
		}
		timeStamp, err := strconv.Atoi(timeStampString)
		if err != nil {
			return nil, fmt.Errorf("unexpected timeStamp length, it should be int64")
		}

		if time.Now().Unix()-int64(timeStamp) > JwtExpireSeconds {
			return nil, fmt.Errorf("unexpected timeStamp in claims: %d", timeStamp)
		}
		ans = ans + fmt.Sprintf("time: %d\n", timeStamp)

		clientIDString, ok := claims["id"].(string)
		if !ok {
			return nil, fmt.Errorf("unexpected client id: %v", clientIDString)
		}
		clientID, err := strconv.Atoi(clientIDString)
		if err != nil {
			return nil, fmt.Errorf("unexpected timeStamp length, it should be int64")
		}

		ans = ans + fmt.Sprintf("id: %d\n", clientID)

		return []byte(secret), nil
	})
	if err != nil {
		return err.Error()
	}

	return ans
}
