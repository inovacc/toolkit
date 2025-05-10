package ds

// LinkedList is a generic interface for a singly linked list implementation.
type LinkedList[T any] interface {
	// Insert adds a value to the end of the list.
	Insert(value T)

	// InsertAt adds a value at the specified position.
	// Returns true if successful, false if the position is invalid.
	InsertAt(value T, pos int) bool

	// DeleteAt removes the value at the specified position.
	// Returns true if successful, false if the position is invalid.
	DeleteAt(pos int) bool

	// Get retrieves the value at the specified position.
	// Returns the value and true if successful, zero value and false if the position is invalid.
	Get(pos int) (T, bool)

	// Len returns the number of elements in the list.
	Len() int
}

type node[T any] struct {
	value T
	next  *node[T]
}

type SinglyLinkedList[T any] struct {
	head *node[T]
	len  int
}

func NewLinkedList[T any]() *SinglyLinkedList[T] {
	return &SinglyLinkedList[T]{}
}

func (l *SinglyLinkedList[T]) Insert(value T) {
	n := &node[T]{value: value}
	if l.head == nil {
		l.head = n
	} else {
		curr := l.head
		for curr.next != nil {
			curr = curr.next
		}
		curr.next = n
	}
	l.len++
}

func (l *SinglyLinkedList[T]) InsertAt(value T, pos int) bool {
	if pos < 0 || pos > l.len {
		return false
	}
	n := &node[T]{value: value}
	if pos == 0 {
		n.next = l.head
		l.head = n
		l.len++
		return true
	}
	curr := l.head
	for i := 0; i < pos-1; i++ {
		curr = curr.next
	}
	n.next = curr.next
	curr.next = n
	l.len++
	return true
}

func (l *SinglyLinkedList[T]) DeleteAt(pos int) bool {
	if pos < 0 || pos >= l.len {
		return false
	}
	if pos == 0 {
		l.head = l.head.next
		l.len--
		return true
	}
	curr := l.head
	for i := 0; i < pos-1; i++ {
		curr = curr.next
	}
	curr.next = curr.next.next
	l.len--
	return true
}

func (l *SinglyLinkedList[T]) Get(pos int) (T, bool) {
	var zero T
	if pos < 0 || pos >= l.len {
		return zero, false
	}
	curr := l.head
	for i := 0; i < pos; i++ {
		curr = curr.next
	}
	return curr.value, true
}

func (l *SinglyLinkedList[T]) Len() int {
	return l.len
}
