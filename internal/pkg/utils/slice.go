package utils

// ToSet converts a slice to a map for O(1) membership lookups.
func ToSet[T comparable](items []T) map[T]struct{} {
	set := make(map[T]struct{}, len(items))
	for _, item := range items {
		set[item] = struct{}{}
	}
	return set
}
