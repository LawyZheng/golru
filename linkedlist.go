package golru

import (
	"fmt"
	"sync"
	"time"
)

func newNode[T any](val T, expire time.Duration) *node[T] {
	var t time.Time
	if expire > 0 {
		t = time.Now().Add(expire)
	}

	return &node[T]{
		val:        val,
		expireTime: t,
		lock:       new(sync.RWMutex),
	}
}

type node[T any] struct {
	pre  *node[T]
	next *node[T]
	lock *sync.RWMutex

	val        T
	expireTime time.Time
}

func (n *node[T]) Val() T {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return n.val
}

func (n *node[T]) SetVal(val T) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.val = val
}

func (n *node[T]) SetExpire(expire time.Duration) {
	n.lock.Lock()
	defer n.lock.Unlock()

	if expire <= 0 {
		n.expireTime = time.Time{}
	} else {
		n.expireTime = time.Now().Add(expire)
	}
}

func (n *node[T]) SetValWithExpire(val T, expire time.Duration) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.val = val
	if expire <= 0 {
		n.expireTime = time.Time{}
	} else {
		n.expireTime = time.Now().Add(expire)
	}
}

func (n *node[T]) IsExpire() bool {
	return !n.expireTime.IsZero() && n.expireTime.Before(time.Now())
}

func (n *node[T]) Pre() *node[T] {
	return n.pre
}

func (n *node[T]) SetPre(pre *node[T]) *node[T] {
	n.pre = pre
	return n
}

func (n *node[T]) Next() *node[T] {
	return n.next
}

func (n *node[T]) SetNext(next *node[T]) *node[T] {
	n.next = next
	return n
}

func (n *node[T]) CutOff() {
	pre := n.pre
	next := n.next

	if pre != nil {
		pre.next = next
	}

	if next != nil {
		next.pre = pre
	}

	n.pre = nil
	n.next = nil
	return
}

func (n *node[T]) ReadForward(until *node[T]) []T {
	result := make([]T, 0)
	cur := n
	for cur != until && cur != nil {
		result = append(result, cur.Val())
		cur = cur.next
	}
	return result
}

func (n *node[T]) ReadBackward(until *node[T]) []T {
	result := make([]T, 0)
	cur := n
	for cur != until && cur != nil {
		result = append(result, cur.Val())
		cur = cur.pre
	}
	return result
}

func newDoubleLinkedList[T any]() *doubleLinkedList[T] {
	dummyH := &node[T]{lock: new(sync.RWMutex)}
	dummyT := &node[T]{lock: new(sync.RWMutex)}

	dummyH.next = dummyT
	dummyT.pre = dummyH

	return &doubleLinkedList[T]{
		dummyHead: dummyH,
		dummyTail: dummyT,
	}
}

type doubleLinkedList[T any] struct {
	dummyHead *node[T]
	dummyTail *node[T]
}

func (l *doubleLinkedList[T]) Head() *node[T] {
	if l.dummyHead.next == l.dummyTail {
		return nil
	} else {
		return l.dummyHead.next
	}
}

func (l *doubleLinkedList[T]) Tail() *node[T] {
	if l.dummyTail.pre == l.dummyHead {
		return nil
	} else {
		return l.dummyTail.pre
	}
}

func (l *doubleLinkedList[T]) Append(n *node[T]) {
	n.pre = l.dummyTail.pre
	n.next = l.dummyTail
	l.dummyTail.pre.next = n
	l.dummyTail.pre = n
}

func (l *doubleLinkedList[T]) Prepend(n *node[T]) {
	n.next = l.dummyHead.next
	n.pre = l.dummyHead
	l.dummyHead.next.pre = n
	l.dummyHead.next = n
}

func (l *doubleLinkedList[T]) String() string {
	data := l.dummyHead.next.ReadForward(l.dummyTail)
	return fmt.Sprintf("%v", data)
}
