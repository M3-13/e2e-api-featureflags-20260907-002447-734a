package store

import "sync"

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
	return nil
}

func (s *Store) Get(key string) (Flag, bool) {
	return Flag{}, false
}

func (s *Store) GetAll() []Flag {
	return nil
}

func (s *Store) Update(key string, u FlagUpdate) (Flag, error) {
	return Flag{}, nil
}

func (s *Store) Delete(key string) bool {
	return false
}
