package notify

import (
	"context"
	"testing"
)

func TestDialPublicRejectsLoopback(t *testing.T) {
	_, err := dialPublic(context.Background(), "tcp", "127.0.0.1:443")
	if err == nil {
		t.Fatal("loopback dial allowed")
	}
}

func TestPublicHTTPS(t *testing.T) {
	ctx := context.Background()
	ok := []string{
		"https://8.8.8.8/hook",
		"https://1.1.1.1/a?x=1",
		"https://[2002:808:808::]/hook",
		"https://[64:ff9b:1:808:0:808::]/hook",
	}
	bad := []string{
		"",
		"-",
		"http://8.8.8.8/hook",
		"https://127.0.0.1/hook",
		"https://10.1.2.3/hook",
		"https://192.168.0.5/",
		"https://172.16.0.1/",
		"https://169.254.169.254/",
		"https://0.0.0.0/",
		"https://100.64.0.1/",
		"https://[::1]/",
		"https://[fd00::1]/",
		"https://[2002:7f00:1::]/",
		"https://[64:ff9b::7f00:1]/",
		"https://[64:ff9b:1::a00:1]/",
		"https://[64:ff9b:1:7f00:0:1:808:808]/",
		"https://168.63.129.16/",
		"https://[2001:0:0:0:0:0:80ff:fffe]/",
		"https://100.100.100.200/",
		"https://[fd00:ec2::254]/",
		"https://2130706433/hook",
		"https://kubernetes.default/hook",
		"https://bot.cluster.local/hook",
		"https://user:pass@8.8.8.8/",
		"https://localhost/hook",
		"https://metadata.google.internal/computeMetadata/v1/",
		"https://example.internal/hook",
	}
	for _, raw := range ok {
		if err := PublicHTTPS(ctx, raw); err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
	}
	for _, raw := range bad {
		if err := PublicHTTPS(ctx, raw); err == nil {
			t.Fatalf("%s accepted", raw)
		}
	}
}
