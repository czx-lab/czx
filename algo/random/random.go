package random

import "math/rand/v2"

// Define the types that can be used with RangeRandom
// The ~ operator allows for type sets that include all types that are assignable to the specified type.
// This includes all integer types, unsigned integer types, and floating-point types.
type RangeInterface interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

// Generate a random integer between min and max (inclusive)
// If min is greater than max, the values are swapped.
// This function is not thread-safe, so it should be used in a single goroutine or protected by a mutex.
func RangeRandom[T RangeInterface](min, max T) T {
	if min > max {
		min, max = max, min
	}

	switch any(min).(type) {
	case float32, float64:
		// For floating-point numbers, return a random value in [min, max)
		return T(rand.Float64()*float64(max-min) + float64(min))
	default:
		// For integers, return a random value in [min, max]
		return T(rand.IntN(int(max-min+1)) + int(min))
	}
}

// SlicesRandom returns a slice of random elements from the input slice s.
// The number of elements in the result slice is equal to count.
func SlicesRandom[T comparable](s []T, count int) []T {
	exists := make(map[T]struct{})
	var result []T

	for len(result) < count {
		index := rand.IntN(len(s))
		val := s[index]

		if _, ok := exists[val]; ok {
			continue
		}
		result = append(result, val)
	}

	return result
}
