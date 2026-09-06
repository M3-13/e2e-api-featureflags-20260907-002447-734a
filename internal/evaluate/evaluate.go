package evaluate

import "hash/fnv"

// Decide returns a deterministic boolean rollout decision for the given flag
// key and user. The input is hashed with FNV-1a and reduced to a bucket in
// [0, 100), so the same (key, user) pair always produces the same result.
// rolloutPercent 0 is always false and 100 is always true.
func Decide(key, user string, rolloutPercent int) bool {
	if rolloutPercent <= 0 {
		return false
	}
	if rolloutPercent >= 100 {
		return true
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key + ":" + user))
	return int(h.Sum32()%100) < rolloutPercent
}
