package postgres

// deref returns the value pointed to by ptr, or the zero value of T if ptr is nil.
func deref[T any](ptr *T) T {
	if ptr == nil {
		var zero T
		return zero
	}
	return *ptr
}
