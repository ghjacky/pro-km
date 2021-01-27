package pmp

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const (
	// TokenHeader is header  key name
	TokenHeader = "Authorization"
)

// GenerateJWTToken generate jwt token for pmp authorise
func GenerateJWTToken(secret string, userid string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"time":  fmt.Sprintf("%d", time.Now().Unix()),
		"nonce": fmt.Sprintf("%d", rand.Int31()),
		"user":  map[string]string{"id": userid},
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		panic(err)
	}
	return tokenString
}

// ParseJWTToken parse jwt token for pmp
func ParseJWTToken(token, secret string, expireDuring int64) (string, error) {
	userid := ""
	_, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		var err error

		claims, ok := t.Claims.(jwt.MapClaims)
		if !ok {
			return nil, fmt.Errorf("Unexpected claims type")
		}

		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method")
		}

		timeStampString, ok := claims["time"].(string)
		if !ok {
			return nil, fmt.Errorf("Unexpected timeStamp format, it should be string")
		}
		timeStamp, err := strconv.Atoi(timeStampString)
		if err != nil {
			return nil, fmt.Errorf("Unexpected timeStamp length, it should be int64")
		}

		if time.Now().Unix()-expireDuring > int64(timeStamp) {
			return nil, fmt.Errorf("Token Expired")
		}

		bytes, err := json.Marshal(claims["user"])
		if err != nil {
			return nil, fmt.Errorf("Marshal Failed")
		}

		data := map[string]string{}
		err = json.Unmarshal(bytes, &data)
		userid = data["id"]
		return []byte(secret), err
	})

	if err != nil {
		return "", err
	}

	return userid, err
}
