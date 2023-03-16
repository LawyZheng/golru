package golru

import (
	"fmt"
	"sync"
	"time"
)

func newNode[K comparable, V any](key K, val V, expire time.Duration) *node[K, V] {
	var t time.Time
	if expire > 0 {
		t = time.Now().Add(expire)
	}

	return &node[K, V]{
		key:        key,
		val:        val,
		expireTime: t,
		lastAccess: time.Now(),
		lock:       new(sync.RWMutex),
	}
}

type node[K comparable, V any] struct {
	pre  *node[K, V]
	next *node[K, V]
	lock *sync.RWMutex

	key        K
	val        V
	expireTime time.Time
	lastAccess time.Time
}

// Before compare the target node, return bool
// if ture returned, means the object is older; otherwise, false returned
// note that is always regarded as the newest one.
func (n *node[K, V]) Before(target *node[K, V]) bool {
	if target == nil {
		return true
	}

	return n.lastAccess.Before(target.lastAccess)
}

func (n *node[K, V]) Key() K {
	return n.key
}

func (n *node[K, V]) Val() V {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return n.val
}

func (n *node[K, V]) SetVal(val V) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.val = val
	n.lastAccess = time.Now()
}

func (n *node[K, V]) SetExpire(expire time.Duration) {
	n.lock.Lock()
	defer n.lock.Unlock()

	if expire <= 0 {
		n.expireTime = time.Time{}
	} else {
		n.expireTime = time.Now().Add(expire)
	}
}

func (n *node[K, V]) SetValWithExpire(val V, expire time.Duration) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.val = val
	if expire <= 0 {
		n.expireTime = time.Time{}
	} else {
		n.expireTime = time.Now().Add(expire)
	}
	n.lastAccess = time.Now()
}

func (n *node[K, V]) IsExpire() bool {
	return !n.expireTime.IsZero() && n.expireTime.Before(time.Now())
}

func (n *node[K, V]) Pre() *node[K, V] {
	return n.pre
}

func (n *node[K, V]) SetPre(pre *node[K, V]) *node[K, V] {
	n.pre = pre
	return n
}

func (n *node[K, V]) Next() *node[K, V] {
	return n.next
}

func (n *node[K, V]) SetNext(next *node[K, V]) *node[K, V] {
	n.next = next
	return n
}

func (n *node[K, V]) CutOff() {
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
}

func (n *node[K, V]) ReadForward(until *node[K, V]) []V {
	result := make([]V, 0)
	cur := n
	for cur != until && cur != nil {
		result = append(result, cur.Val())
		cur = cur.next
	}
	return result
}

func (n *node[K, V]) ReadBackward(until *node[K, V]) []V {
	result := make([]V, 0)
	cur := n
	for cur != until && cur != nil {
		result = append(result, cur.Val())
		cur = cur.pre
	}
	return result
}

func newDoubleLinkedList[K comparable, V any]() *doubleLinkedList[K, V] {
	dummyH := &node[K, V]{lock: new(sync.RWMutex)}
	dummyT := &node[K, V]{lock: new(sync.RWMutex)}

	dummyH.next = dummyT
	dummyT.pre = dummyH

	return &doubleLinkedList[K, V]{
		dummyHead: dummyH,
		dummyTail: dummyT,
	}
}

type doubleLinkedList[K comparable, V any] struct {
	dummyHead *node[K, V]
	dummyTail *node[K, V]
}

func (l *doubleLinkedList[K, V]) Head() (*node[K, V], bool) {
	if l.dummyHead.next == l.dummyTail {
		return nil, false
	} else {
		return l.dummyHead.next, true
	}
}

func (l *doubleLinkedList[K, V]) Tail() (*node[K, V], bool) {
	if l.dummyTail.pre == l.dummyHead {
		return nil, false
	} else {
		return l.dummyTail.pre, true
	}
}

func (l *doubleLinkedList[K, V]) Append(n *node[K, V]) {
	n.pre = l.dummyTail.pre
	n.next = l.dummyTail
	l.dummyTail.pre.next = n
	l.dummyTail.pre = n
}

func (l *doubleLinkedList[K, V]) Prepend(n *node[K, V]) {
	n.next = l.dummyHead.next
	n.pre = l.dummyHead
	l.dummyHead.next.pre = n
	l.dummyHead.next = n
}

func (l *doubleLinkedList[K, V]) Pop() (*node[K, V], bool) {
	n, ok := l.Tail()
	if !ok {
		return n, ok
	}
	n.CutOff()
	return n, ok
}

func (l *doubleLinkedList[K, V]) String() string {
	data := l.ReadForward()
	return fmt.Sprintf("%v", data)
}

func (l *doubleLinkedList[K, V]) ReadForward() []V {
	return l.dummyHead.next.ReadForward(l.dummyTail)
}

func (l *doubleLinkedList[K, V]) ReadBackward() []V {
	return l.dummyTail.pre.ReadBackward(l.dummyHead)
}
