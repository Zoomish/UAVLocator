package config

import "testing"

func TestListenAddr(t *testing.T) {
	tests := []struct {
		httpAddr string
		port     string
		want     string
	}{
		{port: "10000", httpAddr: ":3000", want: "0.0.0.0:10000"},
		{port: "10000", want: "0.0.0.0:10000"},
		{httpAddr: ":3000", want: "0.0.0.0:3000"},
		{want: "0.0.0.0:3000"},
		{httpAddr: "127.0.0.1:3000", want: "127.0.0.1:3000"},
	}
	for _, tt := range tests {
		if got := ListenAddr(tt.httpAddr, tt.port); got != tt.want {
			t.Fatalf("ListenAddr(%q, %q) = %s, want %s", tt.httpAddr, tt.port, got, tt.want)
		}
	}
}

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
