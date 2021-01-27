package auth

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	secrets := map[string]string{
		"IDC": "P00WpJ6glL-F8Fiq-eS-VA",
		"ALY": "T2P-4BWeiscDr49C9egusQ",
		"BDY": "DxNMVoqv3-SiNDApnaDJiQ",
		"BSY": "4cRKS3iyjHzi21W3O_nX-Q",
	}

	for name, secret := range secrets {
		tm := NewJwtTokenManager(secret, 100*time.Minute)
		js := `{
    "id": "admin",
    "fullname": "管理员",
    "email": "admin@xxxxx.com",
    "type": "special",
    "status": "active",
    "dn": "special user",
    "cacheTime": 0,
    "organization": "xxxxx"
  }`
		user := User{}
		err := json.Unmarshal([]byte(js), &user)
		if err != nil {
			t.Errorf("gen user failed: %v", err)
			return
		}
		token, _ := tm.Generate(user)
		fmt.Printf("%s: %s\n", name, token)
	}
}
