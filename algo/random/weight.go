package random

import (
	"math/rand/v2"
	"sort"
)

type (
	// WeightItem is a weighted item.
	WeightItem[T any, WT RangeInterface] struct {
		Value  T
		Weight WT
	}
	// WeightPool is a weighted random pool.
	// It is used to select a random item from a list of items with different weights.
	WeightPool[T any, WT RangeInterface] struct {
		items      []WeightItem[T, WT]
		prefixSums []WT // prefix sums of weights
		total      WT   // total weight of all items
	}
)

func NewWeightPool[T any, WT RangeInterface](items []WeightItem[T, WT]) *WeightPool[T, WT] {
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
	}
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
		r = WT(rand.Float64() * float64(p.total))
	default:
		r = WT(rand.Int64N(int64(p.total)))
	}

	// Binary search to find the index of the first prefix sum greater than r
	// This is equivalent to finding the index of the first item with weight greater than r
	index := sort.Search(len(p.prefixSums), func(i int) bool {
		return p.prefixSums[i] > r
	})

	return p.items[index].Value
}
