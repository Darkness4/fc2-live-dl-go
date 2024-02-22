// Package ptr provides a simple way to get the pointer of a value.
package ptr

// Ref returns the pointer of the value.
func Ref[T any](v T) *T {
	return &v
}
