package ws

import (
	"bytes"
	"testing"
)

func TestRingBuffer_New(t *testing.T) {
	rb := NewRingBuffer(10)
	
	if rb.Len() != 0 {
		t.Errorf("Expected empty buffer, got %d elements", rb.Len())
	}
	
	if rb.cap != 10 {
		t.Errorf("Expected capacity 10, got %d", rb.cap)
	}
}

func TestRingBuffer_AddAndGetAll(t *testing.T) {
	rb := NewRingBuffer(5)
	
	// Add 3 messages (not full)
	rb.Add([]byte("msg1"))
	rb.Add([]byte("msg2"))
	rb.Add([]byte("msg3"))
	
	if rb.Len() != 3 {
		t.Fatalf("Expected 3 elements, got %d", rb.Len())
	}
	
	all := rb.GetAll()
	if len(all) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(all))
	}
	
	// Check order
	if !bytes.Equal(all[0], []byte("msg1")) {
		t.Errorf("Expected msg1 first, got %s", all[0])
	}
	if !bytes.Equal(all[2], []byte("msg3")) {
		t.Errorf("Expected msg3 last, got %s", all[2])
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := NewRingBuffer(3)
	
	// Add 5 messages to a capacity-3 buffer
	rb.Add([]byte("msg1"))
	rb.Add([]byte("msg2"))
	rb.Add([]byte("msg3"))
	rb.Add([]byte("msg4")) // overwrites msg1
	rb.Add([]byte("msg5")) // overwrites msg2
	
	if rb.Len() != 3 {
		t.Fatalf("Expected 3 elements (capped), got %d", rb.Len())
	}
	
	all := rb.GetAll()
	
	// Should only have msg3, msg4, msg5 in order
	expected := []string{"msg3", "msg4", "msg5"}
	for i, exp := range expected {
		if !bytes.Equal(all[i], []byte(exp)) {
			t.Errorf("Position %d: expected %s, got %s", i, exp, all[i])
		}
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer(5)
	
	rb.Add([]byte("msg1"))
	rb.Add([]byte("msg2"))
	
	rb.Clear()
	
	if rb.Len() != 0 {
		t.Errorf("Expected empty after clear, got %d", rb.Len())
	}
	
	all := rb.GetAll()
	if all != nil {
		t.Errorf("Expected nil from empty buffer, got %v", all)
	}
}

func TestRingBuffer_Empty(t *testing.T) {
	rb := NewRingBuffer(5)
	
	all := rb.GetAll()
	if all != nil {
		t.Errorf("Expected nil from empty buffer, got %v", all)
	}
}
