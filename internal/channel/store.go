package channel

import (
	"context"
	"sync"

	"github.com/gotd/td/session"
)

type Store struct {
	mu      sync.Mutex
	data    []byte
	persist func(context.Context, []byte) error
}

func NewStore(initial []byte, persist func(context.Context, []byte) error) *Store {
	return &Store{
		data:    append([]byte(nil), initial...),
		persist: persist,
	}
}

func (s *Store) LoadSession(context.Context) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.data) == 0 {
		return nil, session.ErrNotFound
	}
	return append([]byte(nil), s.data...), nil
}

func (s *Store) StoreSession(ctx context.Context, data []byte) error {
	if s.persist != nil {
		if err := s.persist(ctx, data); err != nil {
			return err
		}
	}
	s.mu.Lock()
	s.data = append([]byte(nil), data...)
	s.mu.Unlock()
	return nil
}

func (s *Store) Bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.data...)
}
