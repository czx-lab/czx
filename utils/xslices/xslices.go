package xslices

import "math/rand/v2"

// Search returns the index of the first element in s that satisfies f, or -1 if no such element exists.
// It returns a boolean indicating whether the element was found.
func Search[T any, S ~[]T](s S, f func(T) bool) (int, bool) {
	for i, v := range s {
		if f(v) {
			return i, true
		}
	}
	return -1, false
}

// Reverse reverses the order of elements in a slice.
// It creates a new slice and returns it, leaving the original slice unchanged.
func Reverse[T any, S ~[]T](s S) []T {
	reversed := make(S, len(s))
	copy(reversed, s)
	for i := range len(reversed) / 2 {
		j := len(reversed) - 1 - i
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	return reversed
}

// Shuffle randomly shuffles the elements of a slice in place.
// It uses the rand.Shuffle function from the math/rand/v2 package.
func Shuffle[T any, S ~[]T](s S) {
	rand.Shuffle(len(s), func(i, j int) {
		s[i], s[j] = s[j], s[i]
	})
}
