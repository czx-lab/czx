package random

import (
	"math/rand/v2"
	"sort"
)

type (
	// WeightItem is a weighted item.
	WeightItem[T any] struct {
		Value  T
		Weight int
	}
	// WeightPool is a weighted random pool.
	// It is used to select a random item from a list of items with different weights.
	WeightPool[T any] struct {
		items      []WeightItem[T]
		prefixSums []int // prefix sums of weights
		total      int   // total weight of all items
	}
)

func NewWeightPool[T any](items []WeightItem[T]) *WeightPool[T] {
	var total int
	prefixSums := make([]int, len(items))
	for i, item := range items {
		total += item.Weight
		prefixSums[i] = total
	}

	return &WeightPool[T]{
		items:      items,
		total:      total,
		prefixSums: prefixSums,
	}
}

// Select a random item from the pool based on its weight.
// The probability of selecting an item is proportional to its weight.
// The higher the weight, the more likely it is to be selected.
func (p *WeightPool[T]) Random() T {
	if p.total <= 0 || len(p.items) == 0 {
		var zero T
		return zero
	}
	r := rand.IntN(p.total)

	// Binary search to find the index of the first prefix sum greater than r
	// This is equivalent to finding the index of the first item with weight greater than r
	index := sort.Search(len(p.prefixSums), func(i int) bool {
		return p.prefixSums[i] > r
	})

	return p.items[index].Value
}
