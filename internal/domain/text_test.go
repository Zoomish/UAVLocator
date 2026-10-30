package domain

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Москва", "москва"},
		{"  Санкт-Петербург! ", "санкт петербург"},
		{"Краснодарский край", "краснодарский край"},
		{"Ёлка", "лка"},
		{"Белгород-22", "белгород 22"},
		{"<blockquote expandable>Калуга</blockquote>Москва", "калуга москва"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := Normalize(tt.in); got != tt.want {
				t.Fatalf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeLocations(t *testing.T) {
	got := NormalizeLocations("Москва, Санкт-Петербург, , Краснодарский край")
	want := []string{"москва", "санкт петербург", "краснодарский край"}
	if len(got) != len(want) {
		t.Fatalf("len %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d]=%q want %q", i, got[i], want[i])
		}
	}
	if len(NormalizeLocations(" , ")) != 0 {
		t.Fatal("expected empty")
	}
}
