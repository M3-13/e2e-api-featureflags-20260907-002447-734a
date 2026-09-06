package evaluate

import (
	"fmt"
	"hash/fnv"
)

// hashInput serializes the key/user pair with an explicit length prefix so
// that ambiguous concatenations (e.g. "a"+"b:c" vs "a:b"+"c") cannot collide
// with each other.
func hashInput(key, user string) string {
	return fmt.Sprintf("%d:%s:%d:%s", len(key), key, len(user), user)
}

// Decide returns a deterministic boolean rollout decision for the given flag
// key and user. The input is serialized with length prefixes and hashed with
// FNV-1a, then reduced to a bucket in [0, 100), so the same (key, user) pair
// always produces the same result. rolloutPercent 0 is always false and 100
// is always true.
func Decide(key, user string, rolloutPercent int) bool {
	if rolloutPercent <= 0 {
		return false
	}
	if rolloutPercent >= 100 {
		return true
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(hashInput(key, user)))
	return int(h.Sum32()%100) < rolloutPercent
}
