package ping

import (
	"fmt"
	"testing"
)

func TestSSHPing(t *testing.T) {
	tests := []struct {
		username string
		password string
		ip       string
		port     int
		exp      bool
	}{
		{"root", "xxxxx000", "172.16.244.7", 22, true},
		{"root", "xxxxx111", "172.16.244.7", 22, false},
	}

	for _, test := range tests {
		ok, _ := SSH(test.username, test.password, test.ip, test.port)
		if test.exp != ok {
			t.Errorf("SSH ping not valid: %t != %t", test.exp, ok)
		}
	}

}

func TestPing(t *testing.T) {
	tests := []struct {
		ip  string
		exp bool
	}{
		{"172.16.244.7", true},
		{"127.0.0.1", true},
		{"11.111.11.111", false},
	}

	for _, test := range tests {
		ok, err := Ping(test.ip)
		if err != nil {
			fmt.Printf("Ping err: %v\n", err)
		}
		if test.exp != ok {
			t.Errorf("Ping not valid: %t != %t", test.exp, ok)
		}
	}

}
