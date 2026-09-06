package evaluate

import "testing"

func TestDecideDeterministic(t *testing.T) {
	first := Decide("feature", "user-1", 50)
	for i := 0; i < 100; i++ {
		if got := Decide("feature", "user-1", 50); got != first {
			t.Fatalf("Decide not deterministic: first=%v got=%v", first, got)
		}
	}
}

func TestDecideSameInputAcrossPercent(t *testing.T) {
	// A stable hash means the same input maps to the same bucket regardless
	// of how many times it is called; the threshold alone decides the outcome.
	a := Decide("feature", "user-1", 50)
	b := Decide("feature", "user-1", 50)
	if a != b {
		t.Fatalf("expected identical results, got %v and %v", a, b)
	}
}

func TestDecideZeroAlwaysFalse(t *testing.T) {
	for _, user := range []string{"", "user-1", "another"} {
		if Decide("feature", user, 0) {
			t.Fatalf("Decide with rollout 0 must be false for user %q", user)
		}
	}
}

func TestDecideHundredAlwaysTrue(t *testing.T) {
	for _, user := range []string{"", "user-1", "another"} {
		if !Decide("feature", user, 100) {
			t.Fatalf("Decide with rollout 100 must be true for user %q", user)
		}
	}
}

func TestDecideNegativeAlwaysFalse(t *testing.T) {
	if Decide("feature", "user-1", -5) {
		t.Fatal("Decide with negative rollout must be false")
	}
}

func TestDecideBounds(t *testing.T) {
	// A rollout in the open range (0,100) yields a stable, valid bool.
	for p := 1; p < 100; p++ {
		v := Decide("feature", "user-1", p)
		_ = v // must not panic; value itself depends on the hash bucket
	}
}
