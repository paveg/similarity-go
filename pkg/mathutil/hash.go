package mathutil

// CreateConsistentKey creates a consistent cache key from two hash strings.
// Always puts the lexicographically smaller hash first to ensure
// (A,B) and (B,A) produce the same key.
func CreateConsistentKey(hash1, hash2 string) string {
	if hash1 < hash2 {
		return hash1 + "|" + hash2
	}
	return hash2 + "|" + hash1
}
