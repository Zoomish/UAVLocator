package config

import "testing"

func TestSSLMode(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{host: "localhost", want: "disable"},
		{host: "127.0.0.1", want: "disable"},
		{host: "::1", want: "disable"},
		{host: "db.example.com", want: "require"},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := sslMode(tt.host); got != tt.want {
				t.Fatalf("sslMode(%q) = %s, want %s", tt.host, got, tt.want)
			}
		})
	}
}
