package actor

import "github.com/czx-lab/czx/container/recycler"

type DefaultRecycler struct {
}

func NewRecycler() *DefaultRecycler {
	return &DefaultRecycler{}
}

// Shrink implements recycler.Recycler.
func (r *DefaultRecycler) Shrink(len_ int, cap_ int) bool {
	return float64(len_) < float64(cap_)/1.35
}

var _ recycler.Recycler = (*DefaultRecycler)(nil)
