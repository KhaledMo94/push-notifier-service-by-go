package repository

// chunk splits items into consecutive slices of at most size elements.
// The returned slices share the backing array of items.
func chunk[T any](items []T, size int) [][]T {
	chunks := make([][]T, 0, (len(items)+size-1)/size)
	for size < len(items) {
		items, chunks = items[size:], append(chunks, items[:size])
	}
	if len(items) > 0 {
		chunks = append(chunks, items)
	}
	return chunks
}

// uniqueNonEmpty returns values without empty strings and duplicates, keeping first-seen order.
func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
