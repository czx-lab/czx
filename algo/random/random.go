package random

import (
	"math/rand/v2"
	"sort"
	"time"
)

// proportionality factor
const k = 0x9e3779b97f4a7c13

type (
	// Define the types that can be used with RangeRandom
	// The ~ operator allows for type sets that include all types that are assignable to the specified type.
	// This includes all integer types, unsigned integer types, and floating-point types.
	INumber interface {
		~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
	}
	// WeightItem is a weighted item.
	WeightItem[T any, WT INumber] struct {
		Value  T
		Weight WT
	}
	// WeightPool is a weighted random pool.
	// It is used to select a random item from a list of items with different weights.
	WeightPool[T any, WT INumber] struct {
		items      []WeightItem[T, WT]
		prefixSums []WT // prefix sums of weights
		total      WT   // total weight of all items
		rng        *rand.Rand
	}
)

func NewWeightPool[T any, WT INumber](items []WeightItem[T, WT]) *WeightPool[T, WT] {
	var total WT
	prefixSums := make([]WT, len(items))
	for i, item := range items {
		total += item.Weight
		prefixSums[i] = total
	}

	return &WeightPool[T, WT]{
		items:      items,
		total:      total,
		prefixSums: prefixSums,
		rng:        rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), k)),
	}
}

// WithRngSource .
func (p *WeightPool[T, WT]) WithRngSource(src rand.Source) {
	p.rng = rand.New(src)
}

// Select a random item from the pool based on its weight.
// The probability of selecting an item is proportional to its weight.
// The higher the weight, the more likely it is to be selected.
func (p *WeightPool[T, WT]) Random() T {
	if p.total <= 0 || len(p.items) == 0 {
		var zero T
		return zero
	}

	var r WT
	switch any(p.total).(type) {
	case float32, float64:
		r = WT(p.rng.Float64() * float64(p.total))
	default:
		r = WT(p.rng.Int64N(int64(p.total)))
	}

	// Binary search to find the index of the first prefix sum greater than r
	// This is equivalent to finding the index of the first item with weight greater than r
	index := sort.Search(len(p.prefixSums), func(i int) bool {
		return p.prefixSums[i] > r
	})

	return p.items[index].Value
}

// Generate a random integer between min and max (inclusive)
// If min is greater than max, the values are swapped.
// This function is not thread-safe, so it should be used in a single goroutine or protected by a mutex.
func Range[T INumber](rng *rand.Rand, min, max T) T {
	if min > max {
		min, max = max, min
	}

	switch any(min).(type) {
	case float32, float64:
		// For floating-point numbers, return a random value in [min, max)
		return T(rng.Float64()*float64(max-min) + float64(min))
	default:
		// For integers, return a random value in [min, max]
		return T(rng.IntN(int(max-min+1)) + int(min))
	}
}

// Slices returns a slice of random elements from the input slice s.
// The number of elements in the result slice is equal to count.
func Slices[T comparable, S ~[]T](rng *rand.Rand, s S, count int) S {
	exists := make(map[T]struct{})
	var result S

	for len(result) < count {
		index := rng.IntN(len(s))
		val := s[index]

		if _, ok := exists[val]; ok {
			continue
		}
		result = append(result, val)
	}

	return result
}
