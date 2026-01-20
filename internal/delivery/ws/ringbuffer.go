package ws

// RingBuffer is a fixed-size circular buffer for storing message history
// It provides O(1) append and efficient memory usage
type RingBuffer struct {
	data  [][]byte
	head  int  // next write position
	size  int  // current number of elements
	cap   int  // maximum capacity
}

// NewRingBuffer creates a new ring buffer with the given capacity
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		data: make([][]byte, capacity),
		head: 0,
		size: 0,
		cap:  capacity,
	}
}

// Add appends a message to the buffer, overwriting oldest if full
func (rb *RingBuffer) Add(msg []byte) {
	// Copy message to avoid external modification
	copied := make([]byte, len(msg))
	copy(copied, msg)
	
	rb.data[rb.head] = copied
	rb.head = (rb.head + 1) % rb.cap
	
	if rb.size < rb.cap {
		rb.size++
	}
}

// GetAll returns all messages in chronological order (oldest first)
func (rb *RingBuffer) GetAll() [][]byte {
	if rb.size == 0 {
		return nil
	}
	
	result := make([][]byte, rb.size)
	
	if rb.size < rb.cap {
		// Buffer not full yet, elements are at indices 0..size-1
		copy(result, rb.data[:rb.size])
	} else {
		// Buffer is full, head points to oldest element
		// Copy from head to end, then from start to head
		copy(result, rb.data[rb.head:])
		copy(result[rb.cap-rb.head:], rb.data[:rb.head])
	}
	
	return result
}

// Len returns the current number of elements
func (rb *RingBuffer) Len() int {
	return rb.size
}

// Clear removes all elements from the buffer
func (rb *RingBuffer) Clear() {
	rb.head = 0
	rb.size = 0
	// Zero out data to allow GC
	for i := range rb.data {
		rb.data[i] = nil
	}
}
