package golru

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	deltaMinor1 uint64 = 18446744073709551615
)

var SHARD_COUNT = 256

type Stringer interface {
	fmt.Stringer
	comparable
}

type CleanUpCallBack[K comparable, V any] func(key K, value V)

// A "thread" safe map of type string:Anything.
// To avoid lock bottlenecks this map is dived to several (SHARD_COUNT) map shards.
type LRUCache[K comparable, V any] struct {
	shards    []*LRUCacheShard[K, V]
	shardMask uint64
	sharding  func(key K) uint64
	pool      *sync.Pool
	close     chan struct{}
	ticker    *time.Ticker
	cleanUpCb CleanUpCallBack[K, V]

	capacity uint64
	size     uint64
}

// A "thread" safe string to anything map.
type LRUCacheShard[K comparable, V any] struct {
	items        map[K]*node[K, V]
	linkedList   *doubleLinkedList[K, V]
	sync.RWMutex // Read Write mutex, guards access to internal map.
}

func createShard[K comparable, V any](size int) *LRUCacheShard[K, V] {
	return &LRUCacheShard[K, V]{
		items:      make(map[K]*node[K, V], size),
		linkedList: newDoubleLinkedList[K, V](),
	}
}

func create[K comparable, V any](sharding func(key K) uint64, initialSize int) LRUCache[K, V] {
	m := LRUCache[K, V]{
		sharding:  sharding,
		shardMask: uint64(SHARD_COUNT - 1),
		shards:    make([]*LRUCacheShard[K, V], SHARD_COUNT),
		close:     make(chan struct{}),
		pool: &sync.Pool{
			New: func() any {
				return &node[K, V]{
					lock:       new(sync.RWMutex),
					lastAccess: time.Now(),
				}
			},
		},
	}

	shardSize := 10
	if s := initialSize / SHARD_COUNT; s >= shardSize {
		shardSize = s
	}

	for i := 0; i < SHARD_COUNT; i++ {
		m.shards[i] = createShard[K, V](shardSize)
	}
	return m
}

// Creates a new concurrent map.
func New[V any](initialSize int) LRUCache[string, V] {
	return create[string, V](fnv64a, initialSize)
}

// Creates a new concurrent map.
func NewStringer[K Stringer, V any](initialSize int) LRUCache[K, V] {
	return create[K, V](strfnv64a[K], initialSize)
}

// Creates a new concurrent map.
func NewWithCustomShardingFunction[K comparable, V any](sharding func(key K) uint64, initialSize int) LRUCache[K, V] {
	return create[K, V](sharding, initialSize)
}

type setCallBack[V any] func(exist bool, valueInMap V, newValue V) (res V, stop bool)

func (m *LRUCache[K, V]) get(index uint64, key K, update bool) (V, bool) {
	// Get map shard.
	shard := m.shards[index]

	if update {
		shard.Lock()
		defer shard.Unlock()
	} else {
		shard.RLock()
		defer shard.RUnlock()
	}

	n, ok := shard.items[key]
	var value V
	if ok = ok && !n.IsExpire(); ok {
		if update {
			n.lock.Lock()
			defer n.lock.Unlock()
			n.lastAccess = time.Now()
			n.CutOff()
			shard.linkedList.Prepend(n)
		} else {
			n.lock.RLock()
			defer n.lock.RUnlock()
		}

		value = n.val
	}
	return value, ok
}

func deleteOldestNodeInCache[K comparable, V any](cache *LRUCache[K, V]) *node[K, V] {
	curShard := cache.shards[0]
	curShard.Lock()
	curNode := curShard.linkedList.Tail()

	for i := 1; i < SHARD_COUNT; i++ {
		cache.shards[i].Lock()
		tmpNode := cache.shards[i].linkedList.Tail()
		if curNode != nil && curNode.Before(tmpNode) {
			cache.shards[i].Unlock()
			continue
		}

		curShard.Unlock()
		curShard = cache.shards[i]
		curNode = tmpNode
	}

	if curNode != nil {
		curNode.CutOff()
		cache.pool.Put(curNode)
		delete(curShard.items, curNode.Key())
	}

	atomic.AddUint64(&cache.size, deltaMinor1)
	curShard.Unlock()
	return curNode
}

func (m *LRUCache[K, V]) set(index uint64, key K, value V, expire time.Duration, cb setCallBack[V]) V {
	// Get map shard.
	shard := m.shards[index]
	shard.Lock()
	defer shard.Unlock()

	var (
		delta uint64
		n     *node[K, V]
		ok    bool
	)

	n, ok = shard.items[key]
	if cb != nil {
		var (
			vInMap V
		)
		if n != nil {
			vInMap = n.Val()
		}
		res, stopped := cb(ok, vInMap, value)
		if stopped {
			return value
		}
		value = res
	}

	if !ok {
		if m.capacity != 0 && atomic.LoadUint64(&m.size) >= m.capacity {
			// capacity is full, delete the most remote node according to LRU rule
			shard.Unlock()
			deleteOldestNodeInCache(m)
			shard.Lock()
		}

		delta = 1
		n = m.pool.Get().(*node[K, V])
		n.key = key
		n.SetValWithExpire(value, expire)
		//n = newNode(key, value, expire)
	} else {
		n.CutOff()
		n.SetValWithExpire(value, expire)
	}

	shard.items[key] = n
	shard.linkedList.Prepend(n)
	atomic.AddUint64(&m.size, delta)

	return value
}

func (m *LRUCache[K, V]) delete(index uint64, key K, cb RemoveCb[K, V]) (V, bool) {
	// Get map shard.
	shard := m.shards[index]
	shard.Lock()
	defer shard.Unlock()

	var (
		delta  uint64
		remove = true
		value  V
	)
	n, ok := shard.items[key]
	if n != nil {
		value = n.Val()
	}
	if cb != nil {
		remove = cb(key, value, ok)
	}

	if remove && ok {
		delta = deltaMinor1
		n.CutOff()
		m.pool.Put(n)
		delete(shard.items, key)
	}

	atomic.AddUint64(&m.size, delta)
	return value, remove
}

func (m *LRUCache[K, V]) cleanUp() {
	var wg sync.WaitGroup
	for i := 0; i < SHARD_COUNT; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			shard := m.shards[index]
			shard.Lock()

			for k, v := range shard.items {
				if v == nil {
					delete(shard.items, k)
					continue
				}

				if !v.IsExpire() {
					continue
				}

				val := v.Val()
				v.CutOff()
				m.pool.Put(v)
				delete(shard.items, k)
				atomic.AddUint64(&m.size, deltaMinor1)

				if m.cleanUpCb != nil {
					m.cleanUpCb(k, val)
				}

			}
			shard.Unlock()
		}(i)
	}
	wg.Wait()
}
func (m *LRUCache[K, V]) Close() error {
	close(m.close)
	return nil
}

func (m *LRUCache[K, V]) SetCleanUp(interval time.Duration, callback CleanUpCallBack[K, V]) {
	m.cleanUpCb = callback

	if interval <= 0 {
		if m.ticker != nil {
			m.ticker.Stop()
		}
		return
	}

	if m.ticker == nil {
		m.ticker = time.NewTicker(interval)
	} else {
		m.close <- struct{}{}
		m.ticker.Reset(interval)
	}

	go func() {
		for {
			select {
			case <-m.ticker.C:
				m.cleanUp()
			case <-m.close:
				m.ticker.Stop()
				return
			}
		}
	}()
}

func (m *LRUCache[K, V]) getShardIndex(key K) uint64 {
	return m.sharding(key) & m.shardMask
}

func (m *LRUCache[K, V]) SetCapacity(capacity uint64) *LRUCache[K, V] {
	m.capacity = capacity
	return m
}

// GetShard returns shard under given key
func (m *LRUCache[K, V]) GetShard(key K) *LRUCacheShard[K, V] {
	return m.shards[m.getShardIndex(key)]
}

func (m *LRUCache[K, V]) MSet(data map[K]V) {
	m.MSetWithExpire(data, 0)
}

func (m *LRUCache[K, V]) MSetWithExpire(data map[K]V, expire time.Duration) {
	for key, value := range data {
		m.set(m.getShardIndex(key), key, value, expire, nil)
	}
}

// Sets the given value under the specified key.
func (m *LRUCache[K, V]) Set(key K, value V) {
	m.SetWithExpire(key, value, 0)
}

func (m *LRUCache[K, V]) SetWithExpire(key K, value V, expire time.Duration) {
	m.set(m.getShardIndex(key), key, value, expire, nil)
}

// Callback to return new element to be inserted into the map
// It is called while lock is held, therefore it MUST NOT
// try to access other keys in same map, as it can lead to deadlock since
// Go sync.RWLock is not reentrant
type UpsertCb[V any] func(exist bool, valueInMap V, newValue V) V

// Insert or Update - updates existing element or inserts a new one using UpsertCb
func (m *LRUCache[K, V]) Upsert(key K, value V, cb UpsertCb[V]) (res V) {
	return m.UpsertWithExpire(key, value, 0, cb)
}

func (m *LRUCache[K, V]) UpsertWithExpire(key K, value V, expire time.Duration, cb UpsertCb[V]) (res V) {
	callback := func(exist bool, valueInMap V, newValue V) (res V, stop bool) {
		return cb(exist, valueInMap, newValue), false
	}
	return m.set(m.getShardIndex(key), key, value, expire, callback)
}

// Sets the given value under the specified key if no value was associated with it.
func (m *LRUCache[K, V]) SetIfAbsent(key K, value V) bool {
	return m.SetIfAbsentWithExpire(key, value, 0)
}

func (m *LRUCache[K, V]) SetIfAbsentWithExpire(key K, value V, expire time.Duration) bool {
	var ok bool
	callback := func(exist bool, valueInMap V, newValue V) (res V, stop bool) {
		ok = !exist
		return newValue, exist
	}
	m.set(m.getShardIndex(key), key, value, expire, callback)
	return ok
}

// Get retrieves an element from map under given key.
func (m *LRUCache[K, V]) Get(key K) (V, bool) {
	return m.get(m.getShardIndex(key), key, true)
}

func (m *LRUCache[K, V]) Size() uint64 {
	return m.size
}

// Count returns the number of elements within the map.
func (m *LRUCache[K, V]) Count() int {
	count := 0
	for i := 0; i < SHARD_COUNT; i++ {
		shard := m.shards[i]
		shard.RLock()
		count += len(shard.items)
		shard.RUnlock()
	}
	return count
}

// Looks up an item under specified key
func (m *LRUCache[K, V]) Has(key K) bool {
	_, ok := m.get(m.getShardIndex(key), key, false)
	return ok
}

// Remove removes an element from the map.
func (m *LRUCache[K, V]) Remove(key K) {
	m.delete(m.getShardIndex(key), key, nil)
}

// RemoveCb is a callback executed in a map.RemoveCb() call, while Lock is held
// If returns true, the element will be removed from the map
type RemoveCb[K any, V any] func(key K, v V, exists bool) (remove bool)

// RemoveCb locks the shard containing the key, retrieves its current value and calls the callback with those params
// If callback returns true and element exists, it will remove it from the map
// Returns the value returned by the callback (even if element was not present in the map)
func (m *LRUCache[K, V]) RemoveCb(key K, cb RemoveCb[K, V]) bool {
	_, remove := m.delete(m.getShardIndex(key), key, cb)
	return remove
}

// Pop removes an element from the map and returns it
func (m *LRUCache[K, V]) Pop(key K) (v V, exists bool) {
	callback := func(key K, v V, exist bool) bool {
		exists = exist
		return true
	}
	v, _ = m.delete(m.getShardIndex(key), key, callback)
	return v, exists
}

// IsEmpty checks if map is empty.
func (m *LRUCache[K, V]) IsEmpty() bool {
	return m.Count() == 0
}

// Used by the Iter & IterBuffered functions to wrap two variables together over a channel,
type Tuple[K comparable, V any] struct {
	Key K
	Val V
}

// Iter returns an iterator which could be used in a for range loop.
//
// Deprecated: using IterBuffered() will get a better performence
func (m *LRUCache[K, V]) Iter() <-chan Tuple[K, V] {
	chans := snapshot(m)
	ch := make(chan Tuple[K, V])
	go fanIn(chans, ch)
	return ch
}

// IterBuffered returns a buffered iterator which could be used in a for range loop.
func (m *LRUCache[K, V]) IterBuffered() <-chan Tuple[K, V] {
	chans := snapshot(m)
	total := 0
	for _, c := range chans {
		total += cap(c)
	}
	ch := make(chan Tuple[K, V], total)
	go fanIn(chans, ch)
	return ch
}

// Clear removes all items from map.
func (m *LRUCache[K, V]) Clear() {
	for i := 0; i < SHARD_COUNT; i++ {
		for k := range m.shards[i].items {
			m.Remove(k)
		}
	}
}

// Returns a array of channels that contains elements in each shard,
// which likely takes a snapshot of `m`.
// It returns once the size of each buffered channel is determined,
// before all the channels are populated using goroutines.
func snapshot[K comparable, V any](m *LRUCache[K, V]) (chans []chan *node[K, V]) {
	//When you access map items before initializing.
	if len(m.shards) == 0 {
		panic(`cmap.LRUCache is not initialized. Should run New() before usage.`)
	}
	chans = make([]chan *node[K, V], SHARD_COUNT)
	wg := sync.WaitGroup{}
	wg.Add(SHARD_COUNT)
	// Foreach shard.
	for index, shard := range m.shards {
		go func(index int, shard *LRUCacheShard[K, V]) {
			// Foreach key, value pair.
			shard.RLock()
			chans[index] = make(chan *node[K, V], len(shard.items))
			wg.Done()
			for _, n := range shard.items {
				chans[index] <- n
			}
			shard.RUnlock()
			close(chans[index])
		}(index, shard)
	}
	wg.Wait()
	return chans
}

// fanIn reads elements from channels `chans` into channel `out`
func fanIn[K comparable, V any](chans []chan *node[K, V], out chan Tuple[K, V]) {
	wg := sync.WaitGroup{}
	wg.Add(len(chans))
	for _, ch := range chans {
		go func(ch chan *node[K, V]) {
			for t := range ch {
				if t == nil {
					continue
				}
				if t.IsExpire() {
					continue
				}
				out <- Tuple[K, V]{t.Key(), t.Val()}
			}
			wg.Done()
		}(ch)
	}
	wg.Wait()
	close(out)
}

// Items returns all items as map[string]V
func (m LRUCache[K, V]) Items() map[K]V {
	tmp := make(map[K]V)

	// Insert items to temporary map.
	for item := range m.IterBuffered() {
		tmp[item.Key] = item.Val
	}

	return tmp
}

// Iterator callbacalled for every key,value found in
// maps. RLock is held for all calls for a given shard
// therefore callback sess consistent view of a shard,
// but not across the shards
type IterCb[K comparable, V any] func(key K, v V)

// Callback based iterator, cheapest way to read
// all elements in a map.
func (m LRUCache[K, V]) IterCb(fn IterCb[K, V]) {
	//for item := range m.IterBuffered() {
	//	fn(item.Key, item.Val)
	//}

	for idx := range m.shards {
		shard := (m.shards)[idx]
		shard.RLock()
		for key, n := range shard.items {
			if n == nil {
				continue
			}

			if n.IsExpire() {
				continue
			}

			var value V
			if n != nil {
				value = n.Val()
			}
			fn(key, value)
		}
		shard.RUnlock()
	}
}

// Keys returns all keys as []string
func (m LRUCache[K, V]) Keys() []K {
	count := m.Count()
	ch := make(chan K, count)
	go func() {
		// Foreach shard.
		wg := sync.WaitGroup{}
		wg.Add(SHARD_COUNT)
		for _, shard := range m.shards {
			go func(shard *LRUCacheShard[K, V]) {
				// Foreach key, value pair.
				shard.RLock()
				for key, value := range shard.items {
					if value == nil {
						continue
					}
					if value.IsExpire() {
						continue
					}
					ch <- key
				}
				shard.RUnlock()
				wg.Done()
			}(shard)
		}
		wg.Wait()
		close(ch)
	}()

	// Generate keys
	keys := make([]K, 0, count)
	for k := range ch {
		keys = append(keys, k)
	}
	return keys
}

// Reviles LRUCache "private" variables to json marshal.
func (m LRUCache[K, V]) MarshalJSON() ([]byte, error) {
	// Create a temporary map, which will hold all item spread across shards.
	tmp := make(map[K]V)

	// Insert items to temporary map.
	for item := range m.IterBuffered() {
		tmp[item.Key] = item.Val
	}
	return json.Marshal(tmp)
}
func strfnv32[K fmt.Stringer](key K) uint32 {
	return fnv32(key.String())
}

func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	keyLength := len(key)
	for i := 0; i < keyLength; i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}

func strfnv64a[K fmt.Stringer](key K) uint64 {
	return fnv64a(key.String())
}

func fnv64a(key string) uint64 {
	hash := uint64(14695981039346656037)
	const prime64 = uint64(1099511628211)
	for i := 0; i < len(key); i++ {
		hash ^= uint64(key[i])
		hash *= prime64
	}
	return hash
}

// Reverse process of Marshal.
func (m *LRUCache[K, V]) UnmarshalJSON(b []byte) (err error) {
	tmp := make(map[K]V)

	// Unmarshal into a single map.
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	// foreach key,value pair in temporary map insert into our concurrent map.
	for key, val := range tmp {
		m.Set(key, val)
	}
	return nil
}
