package repository

import "testing"

func TestChooseSession(t *testing.T) {
	env := []byte("env-session")
	db := []byte("db-session")
	envSeed := SessionSeed(env)
	tests := []struct {
		name   string
		env    []byte
		db     []byte
		dbSeed string
		want   []byte
		write  bool
	}{
		{name: "empty"},
		{name: "env only", env: env, want: env, write: true},
		{name: "db only", db: db, dbSeed: "old", want: db},
		{name: "same env keeps db updates", env: env, db: db, dbSeed: envSeed, want: db},
		{name: "new env replaces db", env: env, db: db, dbSeed: "other", want: env, write: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, seed, write := chooseSession(tt.env, tt.db, tt.dbSeed)
			if write != tt.write {
				t.Fatalf("write = %v, want %v", write, tt.write)
			}
			if string(got) != string(tt.want) {
				t.Fatalf("data = %q, want %q", got, tt.want)
			}
			if tt.write && seed != SessionSeed(tt.want) {
				t.Fatalf("seed = %s", seed)
			}
		})
	}
}
