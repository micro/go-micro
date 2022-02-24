// Copyright (c) 2018 ef-ds
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Package deque implements a very fast and efficient general purpose queue/stack/deque
// data structure that is specifically optimized to perform when used by
// Microservices and serverless services running in production environments.
package deque

const (
	// firstSliceSize holds the size of the first slice.
	firstSliceSize = 4

	// sliceGrowthFactor determines by how much and how fast the first internal
	// slice should grow. A growth factor of 4, firstSliceSize = 4 and maxFirstSliceSize = 64,
	// the first slice will start with size 4, then 16 (4*4), then 64 (16*4).
	// The growth factor should be tweaked together with firstSliceSize and specially,
	// maxFirstSliceSize for maximum efficiency.
	// sliceGrowthFactor only applies to the very first slice created. All other
	// subsequent slices are created with fixed size of maxInternalSliceSize.
	sliceGrowthFactor = 4

	// maxFirstSliceSize holds the maximum size of the first slice.
	maxFirstSliceSize = 64

	// maxInternalSliceSize holds the maximum size of each internal slice.
	maxInternalSliceSize = 256

	// maxSpareLinks holds the maximum number of spare slices the deque will keep
	// when shrinking (items are being removed from the deque).
	// 5 means a maximum of 5 slices will be kept as spares, meaning, they
	// have been used before to store data, but are now no longer used.
	// Spare slices are useful in refill situations, when the deque was filled
	// with items and emptied. When the same instance is used to push new items,
	// the spare slices from the previous pushes are already allocated and ready
	// to be used. So the first pushes will push the data into these slices,
	// improving the performance dramatically.
	// A higher spare links number means the refills will have a better performance
	// for larger number of items (as now there's more spare slices ready to be used).
	// The downside is the extra memory usage when the deque shrinks and is
	// holding a small amount of items.
	maxSpareLinks = 5
)

// Deque implements an unbounded, dynamically growing double-ended-queue (deque).
// The zero value for deque is an empty deque ready to use.
type Deque struct {
	// Head points to the first node of the linked list.
	head *node

	// Tail points to the last node of the linked list.
	// In an empty deque, head and tail points to the same node.
	tail *node

	// Hp is the index pointing to the current first element in the deque
	// (i.e. first element added in the current deque values).
	hp int

	// hlp points to the last index in the head slice.
	hlp int

	// tp is the index pointing one beyond the current last element in the deque
	// (i.e. last element added in the current deque values).
	tp int

	// Len holds the current deque values length.
	len int

	// spareLinks holds the number of already used, but now empty, ready-to-be-reused, slices.
	spareLinks int
}

// Node represents a deque node.
// Each node holds a slice of user managed values.
type node struct {
	// v holds the list of user added values in this node.
	v []interface{}

	// n points to the next node in the linked list.
	n *node

	// p points to the previous node in the linked list.
	p *node
}

// New returns an initialized deque.
func New() *Deque {
	return new(Deque)
}

// Init initializes or clears deque d.
func (d *Deque) Init() *Deque {
	*d = Deque{}
	return d
}

// Len returns the number of elements of deque d.
// The complexity is O(1).
func (d *Deque) Len() int { return d.len }

// Front returns the first element of deque d or nil if the deque is empty.
// The second, bool result indicates whether a valid value was returned;
// if the deque is empty, false will be returned.
// The complexity is O(1).
func (d *Deque) Front() (interface{}, bool) {
	if d.len == 0 {
		return nil, false
	}
	return d.head.v[d.hp], true
}

// Back returns the last element of deque d or nil if the deque is empty.
// The second, bool result indicates whether a valid value was returned;
// if the deque is empty, false will be returned.
// The complexity is O(1).
func (d *Deque) Back() (interface{}, bool) {
	if d.len == 0 {
		return nil, false
	}
	return d.tail.v[d.tp-1], true
}

// PushFront adds value v to the the front of the deque.
// The complexity is O(1).
func (d *Deque) PushFront(v interface{}) {
	switch {
	case d.head == nil:
		// No nodes present yet.
		h := &node{v: make([]interface{}, firstSliceSize)}
		h.n = h
		h.p = h
		d.head = h
		d.tail = h
		d.tp = firstSliceSize
		d.hp = firstSliceSize - 1
		d.hlp = d.hp
	case d.hp > 0:
		// There's already room in the head slice.
		d.hp--
	case d.head.p != d.tail:
		// There's at least one spare link between head and tail nodes.
		d.head = d.head.p
		d.hp = len(d.head.v) - 1
		d.hlp = d.hp
		d.spareLinks--
		if d.len == 0 {
			d.tail = d.head
			d.tp = len(d.head.v)
		}
	case len(d.head.v) < maxFirstSliceSize:
		// The first slice hasn't grown big enough yet.
		l := len(d.head.v)
		nl := l * sliceGrowthFactor
		n := make([]interface{}, nl)
		diff := nl - l
		d.tp += diff
		d.hp += diff
		d.hlp = nl - 1
		copy(n[d.hp:], d.head.v)
		d.head.v = n
		d.hp--
	case d.len == 0:
		// The head slice is empty, so reuse it.
		d.tail = d.head
		d.tp = len(d.head.v)
		d.hp = d.tp - 1
		d.hlp = d.hp
	default:
		// No available nodes, so make one.
		n := &node{v: make([]interface{}, maxInternalSliceSize)}
		n.n = d.head
		n.p = d.tail
		d.head.p = n
		d.tail.n = n
		d.head = n
		d.hp = maxInternalSliceSize - 1
		d.hlp = d.hp
	}
	d.len++
	d.head.v[d.hp] = v
}

// PushBack adds value v to the the back of the deque.
// The complexity is O(1).
func (d *Deque) PushBack(v interface{}) {
	switch {
	case d.head == nil:
		// No nodes present yet.
		h := &node{v: make([]interface{}, firstSliceSize)}
		h.n = h
		h.p = h
		d.head = h
		d.tail = h
		d.tail.v[0] = v
		d.hlp = firstSliceSize - 1
		d.tp = 1
	case d.tp < len(d.tail.v):
		// There's room in the tail slice.
		d.tail.v[d.tp] = v
		d.tp++
	case d.tp < maxFirstSliceSize:
		// We're on the first slice and it hasn't grown large enough yet.
		nv := make([]interface{}, len(d.tail.v)*sliceGrowthFactor)
		copy(nv, d.tail.v)
		d.tail.v = nv
		d.tail.v[d.tp] = v
		d.tp++
		d.hlp = len(nv) - 1
	case d.tail.n != d.head:
		// There's at least one spare link between head and tail nodes.
		d.spareLinks--
		n := d.tail.n
		d.tail = n
		d.tail.v[0] = v
		d.tp = 1
	default:
		// No available nodes, so make one.
		n := &node{v: make([]interface{}, maxInternalSliceSize)}
		n.n = d.head
		n.p = d.tail
		d.tail.n = n
		d.head.p = n
		d.tail = n
		d.tail.v[0] = v
		d.tp = 1
	}
	d.len++
}

// PopFront retrieves and removes the current element from the front of the deque.
// The second, bool result indicates whether a valid value was returned;
// if the deque is empty, false will be returned.
// The complexity is O(1).
func (d *Deque) PopFront() (interface{}, bool) {
	if d.len == 0 {
		return nil, false
	}
	vp := &d.head.v[d.hp]
	v := *vp
	*vp = nil // Avoid memory leaks
	d.len--
	switch {
	case d.hp < d.hlp:
		// The head isn't at the end of the slice, so just
		// move on one place.
		d.hp++
	case d.head == d.tail:
		// There's only a single element at the end of the slice
		// so we can't increment hp, so change tp instead.
		d.tp = d.hp
	case d.spareLinks >= maxSpareLinks:
		// Eliminate this link
		d.hp = 0
		d.head.p.n = d.head.n
		d.head.n.p = d.head.p
		d.head = d.head.n
		d.hlp = len(d.head.v) - 1
	default:
		// Leave the link spare.
		d.hp = 0
		d.head = d.head.n
		d.spareLinks++
		d.hlp = len(d.head.v) - 1
	}
	return v, true
}

// PopBack retrieves and removes the current element from the back of the deque.
// The second, bool result indicates whether a valid value was returned;
// if the deque is empty, false will be returned.
// The complexity is O(1).
func (d *Deque) PopBack() (interface{}, bool) {
	if d.len == 0 {
		return nil, false
	}
	d.len--
	d.tp--
	vp := &d.tail.v[d.tp]
	v := *vp
	*vp = nil // Avoid memory leaks
	switch {
	case d.tp > 0:
		// There's space before tp.
	case d.head == d.tail:
		// The list is now empty, so tp==0 is appropriate.
	case d.spareLinks >= maxSpareLinks:
		// Eliminate this link
		d.tail.p.n = d.tail.n
		d.tail.n.p = d.tail.p
		d.tail = d.tail.p
		d.tp = len(d.tail.v)
	default:
		// Leave the link spare.
		d.spareLinks++
		d.tail = d.tail.p
		d.tp = len(d.tail.v)
	}
	return v, true
}
