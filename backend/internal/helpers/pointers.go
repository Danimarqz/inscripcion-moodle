package helpers

// Ptr returns the address of the provided value.
func Ptr[T any](value T) *T {
	return &value
}
