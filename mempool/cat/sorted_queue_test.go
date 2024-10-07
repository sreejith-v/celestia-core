package cat

import (
	"container/heap"
	"sync"
	"testing"
)

// Item is a custom type that implements the Ordered interface for testing purposes
type Item struct {
	value int
}

// LessThan implements the Ordered interface for the Item type
func (i *Item) LessThan(other Ordered) bool {
	return i.value < other.(*Item).value
}

// TestPriorityQueue_PushAndPop is the table-driven test for priorityQueue's Push and Pop methods
func TestPriorityQueue_PushAndPop(t *testing.T) {
	tests := []struct {
		name     string
		input    []*Item
		popOrder []int // Expected pop order
		finalLen int   // Expected length after pops
	}{
		{
			name: "Basic insertion and pop",
			input: []*Item{
				{value: 5},
				{value: 3},
				{value: 8},
				{value: 1},
			},
			popOrder: []int{1, 3, 5, 8},
			finalLen: 0,
		},
		{
			name: "Insert single element",
			input: []*Item{
				{value: 42},
			},
			popOrder: []int{42},
			finalLen: 0,
		},
		{
			name: "Insert already sorted input",
			input: []*Item{
				{value: 1},
				{value: 2},
				{value: 3},
				{value: 4},
				{value: 5},
			},
			popOrder: []int{1, 2, 3, 4, 5},
			finalLen: 0,
		},
		{
			name: "Insert reverse sorted input",
			input: []*Item{
				{value: 5},
				{value: 4},
				{value: 3},
				{value: 2},
				{value: 1},
			},
			popOrder: []int{1, 2, 3, 4, 5},
			finalLen: 0,
		},
		{
			name: "All identical values",
			input: []*Item{
				{value: 7},
				{value: 7},
				{value: 7},
				{value: 7},
			},
			popOrder: []int{7, 7, 7, 7},
			finalLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pq priorityQueue

			// Initialize priorityQueue as a heap
			heap.Init(&pq)

			// Push all input items into the priority queue
			for _, item := range tt.input {
				heap.Push(&pq, item)
			}

			// Ensure the queue length matches the input size
			if pq.Len() != len(tt.input) {
				t.Errorf("Expected queue length %d, got %d", len(tt.input), pq.Len())
			}

			// Pop items and verify the pop order
			for i, expectedValue := range tt.popOrder {
				poppedItem := heap.Pop(&pq).(*Item)

				if poppedItem.value != expectedValue {
					t.Errorf("Pop at index %d: expected %d, got %d", i, expectedValue, poppedItem.value)
				}
			}

			// Ensure the queue length matches the expected final length
			if pq.Len() != tt.finalLen {
				t.Errorf("Expected final queue length %d, got %d", tt.finalLen, pq.Len())
			}
		})
	}
}

// TestPriorityQueue_PointerBehavior ensures that pointers and references are correctly handled
func TestPriorityQueue_PointerBehavior(t *testing.T) {
	// Create two distinct items
	item1 := &Item{value: 10}
	item2 := &Item{value: 5}

	var pq priorityQueue
	heap.Init(&pq)

	// Push the items
	heap.Push(&pq, item1)
	heap.Push(&pq, item2)

	// Pop the first item and ensure it's the smaller one
	popped := heap.Pop(&pq).(*Item)
	if popped.value != 5 {
		t.Errorf("Expected item with value 5, but got %d", popped.value)
	}

	// Modify the original item2 and ensure that the queue still behaves correctly
	item2.value = 100

	// Push the modified item back into the queue
	heap.Push(&pq, item2)

	// Pop the next item and ensure it's the original item1 (value 10)
	popped = heap.Pop(&pq).(*Item)
	if popped.value != 10 {
		t.Errorf("Expected item with value 10, but got %d", popped.value)
	}

	// Pop the modified item2 (which now has value 100)
	popped = heap.Pop(&pq).(*Item)
	if popped.value != 100 {
		t.Errorf("Expected item with value 100, but got %d", popped.value)
	}
}

// TestSortedQueue_Concurrency tests concurrent insertions and pops in SortedQueue
func TestSortedQueue_Concurrency(t *testing.T) {
	// Initialize the sorted queue
	sq := NewSortedQueue()

	var wg sync.WaitGroup
	numGoroutines := 10
	numItemsPerGoroutine := 100

	// Concurrently insert elements into the queue
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for j := 0; j < numItemsPerGoroutine; j++ {
				sq.Insert(&Item{value: base + j})
			}
		}(i * numItemsPerGoroutine)
	}

	wg.Wait()

	// Concurrently pop elements from the queue
	poppedItems := make([]int, 0, numGoroutines*numItemsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < numItemsPerGoroutine; j++ {
			item, ok := sq.Pop()
			if ok {
				poppedItems = append(poppedItems, item.(*Item).value)

			}
		}
	}

	// Check that the popped elements are in sorted order
	for i := 1; i < len(poppedItems); i++ {
		if poppedItems[i] < poppedItems[i-1] {
			t.Errorf("Popped items are not in sorted order: %d < %d at index %d", poppedItems[i], poppedItems[i-1], i)
		}
	}
}

// TestSortedQueue_SimpleInsertAndPop tests sequential inserts and pops
func TestSortedQueue_SimpleInsertAndPop(t *testing.T) {
	sq := NewSortedQueue()

	// Insert items
	sq.Insert(&Item{value: 5})
	sq.Insert(&Item{value: 2})
	sq.Insert(&Item{value: 9})
	sq.Insert(&Item{value: 1})

	// Pop and verify the order is correct
	expectedOrder := []int{1, 2, 5, 9}
	for _, expected := range expectedOrder {
		item, ok := sq.Pop()
		if !ok {
			t.Fatalf("Expected to pop an item, but got none")
		}
		if item.(*Item).value != expected {
			t.Errorf("Expected %d, but got %d", expected, item.(*Item).value)
		}
	}

	// Pop from empty queue
	_, ok := sq.Pop()
	if ok {
		t.Errorf("Expected no item from empty queue, but got one")
	}
}

// TestSortedQueue_ConcurrentInsert tests concurrent inserts without pops
func TestSortedQueue_ConcurrentInsert(t *testing.T) {
	sq := NewSortedQueue()

	var wg sync.WaitGroup
	numGoroutines := 5
	numItemsPerGoroutine := 1000

	// Concurrently insert elements
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for j := 0; j < numItemsPerGoroutine; j++ {
				sq.Insert(&Item{value: base + j})
			}
		}(i * numItemsPerGoroutine)
	}

	// Wait for all insertions to complete
	wg.Wait()

	// Verify queue size after all inserts
	expectedLen := numGoroutines * numItemsPerGoroutine
	if sq.Len() != expectedLen {
		t.Errorf("Expected queue length %d, but got %d", expectedLen, sq.Len())
	}

	// Verify that the smallest element is correctly ordered
	firstItem, ok := sq.Pop()
	if !ok || firstItem.(*Item).value != 0 {
		t.Errorf("Expected first item to be 0, but got %d", firstItem.(*Item).value)
	}
}

// TestSortedQueue_ConcurrentPop ensures that concurrent pops behave correctly
func TestSortedQueue_ConcurrentPop(t *testing.T) {
	sq := NewSortedQueue()

	// Insert some elements sequentially
	for i := 0; i < 1000; i++ {
		sq.Insert(&Item{value: i})
	}

	var wg sync.WaitGroup
	numGoroutines := 5
	numItemsPerGoroutine := 200
	var popMu sync.Mutex
	poppedItems := make([]int, 0, 1000)

	// Concurrently pop elements
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numItemsPerGoroutine; j++ {
				item, ok := sq.Pop()
				if ok {
					popMu.Lock()
					poppedItems = append(poppedItems, item.(*Item).value)
					popMu.Unlock()
				}
			}
		}()
	}

	// Wait for all pops to finish
	wg.Wait()

	// Verify that we popped the correct number of items
	if len(poppedItems) != numGoroutines*numItemsPerGoroutine {
		t.Errorf("Expected %d popped items, but got %d", numGoroutines*numItemsPerGoroutine, len(poppedItems))
	}

	// Ensure that the popped items are in sorted order
	for i := 1; i < len(poppedItems); i++ {
		if poppedItems[i] < poppedItems[i-1] {
			t.Errorf("Popped items are not in sorted order: %d < %d at index %d", poppedItems[i], poppedItems[i-1], i)
		}
	}
}
