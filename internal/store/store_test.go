package store

import (
	"errors"
	"sync"
	"testing"
)

func TestCreateAndGet(t *testing.T) {
	s := NewStore()

	f := Flag{Key: "feature-x", Enabled: true, Description: "desc", RolloutPercent: 50}
	if err := s.Create(f); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	got, ok := s.Get("feature-x")
	if !ok {
		t.Fatal("Get returned ok=false for existing key")
	}
	if got != f {
		t.Fatalf("Get returned %+v, want %+v", got, f)
	}
}

func TestCreateDuplicateKey(t *testing.T) {
	s := NewStore()

	f := Flag{Key: "dup", Enabled: true}
	if err := s.Create(f); err != nil {
		t.Fatalf("first Create returned error: %v", err)
	}

	if err := s.Create(f); !errors.Is(err, ErrKeyExists) {
		t.Fatalf("second Create error = %v, want ErrKeyExists", err)
	}
}

func TestGetUnknownKey(t *testing.T) {
	s := NewStore()

	if _, ok := s.Get("missing"); ok {
		t.Fatal("Get returned ok=true for unknown key")
	}
}

func TestGetAll(t *testing.T) {
	s := NewStore()

	if got := s.GetAll(); len(got) != 0 {
		t.Fatalf("GetAll on empty store = %+v, want empty", got)
	}

	if err := s.Create(Flag{Key: "a"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Create(Flag{Key: "b"}); err != nil {
		t.Fatal(err)
	}

	got := s.GetAll()
	if len(got) != 2 {
		t.Fatalf("GetAll length = %d, want 2", len(got))
	}

	keys := map[string]bool{}
	for _, f := range got {
		keys[f.Key] = true
	}
	if !keys["a"] || !keys["b"] {
		t.Fatalf("GetAll missing keys, got %+v", got)
	}
}

func TestUpdate(t *testing.T) {
	s := NewStore()

	if err := s.Create(Flag{Key: "k", Enabled: true, Description: "old", RolloutPercent: 10}); err != nil {
		t.Fatal(err)
	}

	u := FlagUpdate{Enabled: false, Description: "new", RolloutPercent: 90}
	got, err := s.Update("k", u)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	if got.Key != "k" {
		t.Fatalf("Update changed key to %q, want unchanged", got.Key)
	}
	if got.Enabled != false || got.Description != "new" || got.RolloutPercent != 90 {
		t.Fatalf("Update result = %+v, want updated fields", got)
	}

	// Verify persistence.
	stored, ok := s.Get("k")
	if !ok {
		t.Fatal("flag missing after update")
	}
	if stored != got {
		t.Fatalf("stored %+v != returned %+v", stored, got)
	}
}

func TestUpdateUnknownKey(t *testing.T) {
	s := NewStore()

	if _, err := s.Update("missing", FlagUpdate{Enabled: true}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update error = %v, want ErrNotFound", err)
	}
}

func TestDelete(t *testing.T) {
	s := NewStore()

	if err := s.Create(Flag{Key: "k"}); err != nil {
		t.Fatal(err)
	}

	if !s.Delete("k") {
		t.Fatal("Delete returned false for existing key")
	}
	if _, ok := s.Get("k"); ok {
		t.Fatal("flag still present after delete")
	}

	if s.Delete("k") {
		t.Fatal("Delete returned true for already-deleted key")
	}
}

func TestDeleteUnknownKey(t *testing.T) {
	s := NewStore()

	if s.Delete("missing") {
		t.Fatal("Delete returned true for unknown key")
	}
}

func TestConcurrentReadsAndWrites(t *testing.T) {
	s := NewStore()

	const writers = 8
	const readers = 16
	const iterations = 200

	var wg sync.WaitGroup

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := string(rune('a'+id)) + "-" + string(rune('a'+j%26))
				_ = s.Create(Flag{Key: key, Enabled: true})
				_, _ = s.Update(key, FlagUpdate{Enabled: false, RolloutPercent: 100})
				s.Delete(key)
			}
		}(i)
	}

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, _ = s.Get("some-key")
				_ = s.GetAll()
			}
		}()
	}

	wg.Wait()
}
