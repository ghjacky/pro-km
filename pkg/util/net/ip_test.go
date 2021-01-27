package net

import "testing"

func TestIsIPv4(t *testing.T) {
	testCases := []struct {
		ip  string
		exp bool
	}{
		{"", false},
		{"172.16.244.106", true},
		{"b2:00:ac:d8:90:00", false},
		{"b::1", false},
	}
	for _, test := range testCases {
		if IsIPv4(test.ip) != test.exp {
			t.Errorf("Not Valid")
		}
	}
}
