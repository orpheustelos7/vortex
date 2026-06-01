package config

import "sync"

type Store struct {
mu       sync.RWMutex
policies map[string]Policy
}

func NewStore() *Store {
return &Store{policies: make(map[string]Policy)}
}

func (s *Store) Get(tenant string) (Policy, bool) {
s.mu.RLock()
defer s.mu.RUnlock()
p, ok := s.policies[tenant]
return p, ok
}

func (s *Store) Set(p Policy) {
s.mu.Lock()
defer s.mu.Unlock()
s.policies[p.TenantID] = p
}

func (s *Store) Delete(tenant string) {
s.mu.Lock()
defer s.mu.Unlock()
delete(s.policies, tenant)
}
