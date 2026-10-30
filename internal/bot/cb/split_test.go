package cb

import "testing"

func TestSplit(t *testing.T) {
	tests := []struct {
		in, head, rest string
	}{
		{"get-settings", "get", "settings"},
		{"set-notifications-true", "set", "notifications-true"},
		{"set-admin-42-1", "set", "admin-42-1"},
		{"cancel", "cancel", ""},
	}
	for _, tt := range tests {
		h, r := Split(tt.in)
		if h != tt.head || r != tt.rest {
			t.Fatalf("Split(%q) = %q %q", tt.in, h, r)
		}
	}
}
