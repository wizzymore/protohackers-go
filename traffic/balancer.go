package traffic

import "slices"

type Balancer[T comparable] struct {
	set   []T
	index int
}

func NewBalancer[T comparable]() *Balancer[T] {
	return &Balancer[T]{}
}

func (self *Balancer[T]) Add(item T) {
	pos := slices.Index(self.set, item)
	if pos != -1 {
		return
	}
	self.set = append(self.set, item)
}

func (self *Balancer[T]) Remove(item T) {
	if len(self.set) == 0 {
		return
	}
	pos := slices.Index(self.set, item)
	if pos == -1 {
		return
	}
	self.set = append(self.set[:pos], self.set[pos+1:]...)
	if len(self.set) >= self.index {
		self.index = 0
	}
}

func (self *Balancer[T]) Len() int {
	return len(self.set)
}

func (self *Balancer[T]) Get() (item T, ok bool) {
	if len(self.set) == 0 {
		return item, false
	}
	item = self.set[self.index]
	self.index = (self.index + 1) % len(self.set)
	return item, true
}

func (self Balancer[T]) Index() int {
	return self.index
}

func (self *Balancer[T]) ResetIndex() {
	self.index = 0
}

func (self *Balancer[T]) Reset() {
	self.ResetIndex()
	self.set = []T{}
}
