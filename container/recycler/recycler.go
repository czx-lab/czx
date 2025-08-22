package recycler

// Recycler is an interface that defines methods for recycling and managing memory usage of data structures.
type Recycler interface {
	// Shrink attempts to reduce the memory usage of the container.
	// It returns true if the container was successfully shrunk.
	Shrink(len_ int, cap_ int) bool
}
