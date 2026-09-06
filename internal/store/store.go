package store

import (
	"errors"
	"sort"
	"sync"
)

var ErrKeyExists = errors.New("key already exists")

var ErrNotFound = errors.New("flag not found")

type Flag struct {
	Key            string `json:"key"`
	Enabled        bool   `json:"enabled"`
	Description    string `json:"description,omitempty"`
	RolloutPercent int    `json:"rollout_percent"`
}

type FlagUpdate struct {
	Enabled        bool   `json:"enabled"`
	Description    string `json:"description"`
	RolloutPercent int    `json:"rollout_percent"`
}

type Store struct {
	mu    sync.RWMutex
	flags map[string]Flag
}

func NewStore() *Store {
	return &Store{
		flags: make(map[string]Flag),
	}
}

func (s *Store) Create(f Flag) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.flags[f.Key]; exists {
		return ErrKeyExists
	}

	s.flags[f.Key] = f
	return nil
}

func (s *Store) Get(key string) (Flag, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, ok := s.flags[key]
	return f, ok
}

func (s *Store) GetAll() []Flag {
	s.mu.RLock()
	defer s.mu.RUnlock()

	flags := make([]Flag, 0, len(s.flags))
	for _, f := range s.flags {
		flags = append(flags, f)
	}
	sort.Slice(flags, func(i, j int) bool { return flags[i].Key < flags[j].Key })
	return flags
}

func (s *Store) Update(key string, u FlagUpdate) (Flag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, ok := s.flags[key]
	if !ok {
		return Flag{}, ErrNotFound
	}

	f.Enabled = u.Enabled
	f.Description = u.Description
	f.RolloutPercent = u.RolloutPercent
	s.flags[key] = f
	return f, nil
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.flags[key]; !ok {
		return false
	}

	delete(s.flags, key)
	return true
}
