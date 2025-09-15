package util

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

// CacheConfig 用于配置LRU缓存的行为。
type CacheConfig[K comparable, V any] struct {
	// Capacity 是缓存的最大元素数量。如果为0，则不限制数量。
	Capacity int
	// MaxWeight 是缓存中所有元素的最大权重总和。如果为0，则不限制权重。
	MaxWeight int
	// TTL 是元素的存活时间。如果为0，则元素永不过期。
	TTL time.Duration
}

// entry 结构体用于存储链表节点中的实际数据。
type entry[K comparable, V any] struct {
	key        K
	value      V
	weight     int       // 元素的权重
	expiration time.Time // 元素的过期时间
}

// LRUCache 是一个支持泛型、可配置且线程安全的LRU缓存。
type LRUCache[K comparable, V any] struct {
	config        CacheConfig[K, V]
	ll            *list.List
	cache         map[K]*list.Element
	currentWeight int
	lock          sync.RWMutex // 读写锁保证并发安全
}

// NewWithConfig 使用指定的配置创建一个LRU缓存实例。
func NewWithConfig[K comparable, V any](config CacheConfig[K, V]) (*LRUCache[K, V], error) {
	// 至少要有一个限制条件
	if config.Capacity <= 0 && config.MaxWeight <= 0 {
		return nil, fmt.Errorf("必须设置 Capacity 或 MaxWeight 中的至少一个")
	}
	return &LRUCache[K, V]{
		config: config,
		ll:     list.New(),
		cache:  make(map[K]*list.Element),
	}, nil
}

// Get 方法根据键获取一个值。
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	element, ok := c.cache[key]
	if !ok {
		var zeroV V
		return zeroV, false
	}

	// 检查TTL是否过期（被动淘汰）
	entry := element.Value.(*entry[K, V])
	if c.config.TTL > 0 && time.Now().After(entry.expiration) {
		// 已过期，从缓存中移除
		c.removeElement(element)
		var zeroV V
		return zeroV, false
	}

	// 标记为最近使用
	c.ll.MoveToFront(element)
	return entry.value, true
}

// Put 方法向缓存中添加或更新一个键值对，并指定其权重。
// 如果使用基于容量的淘汰，可以为 weight 传入 1。
func (c *LRUCache[K, V]) Put(key K, value V, weight int) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// 检查键是否已经存在
	if element, ok := c.cache[key]; ok {
		// --- 更新现有元素 ---
		entry := element.Value.(*entry[K, V])
		// 更新权重
		c.currentWeight += (weight - entry.weight)
		entry.weight = weight
		entry.value = value
		// 更新TTL
		if c.config.TTL > 0 {
			entry.expiration = time.Now().Add(c.config.TTL)
		}
		c.ll.MoveToFront(element)
	} else {
		// --- 插入新元素 ---
		newEntry := &entry[K, V]{
			key:    key,
			value:  value,
			weight: weight,
		}
		if c.config.TTL > 0 {
			newEntry.expiration = time.Now().Add(c.config.TTL)
		}
		element := c.ll.PushFront(newEntry)
		c.cache[key] = element
		c.currentWeight += weight
	}

	// 检查是否需要淘汰元素
	// 使用 for 循环，因为一个大的新元素可能需要淘汰多个旧元素
	for c.isOverCapacity() {
		c.evict()
	}
}

// isOverCapacity 检查缓存是否超出容量或权重限制。
// 此方法假设已持有锁。
func (c *LRUCache[K, V]) isOverCapacity() bool {
	// 检查容量
	if c.config.Capacity > 0 && c.ll.Len() > c.config.Capacity {
		return true
	}
	// 检查权重
	if c.config.MaxWeight > 0 && c.currentWeight > c.config.MaxWeight {
		return true
	}
	return false
}

// evict 淘汰最久未使用的元素。
// 此方法假设已持有锁。
func (c *LRUCache[K, V]) evict() {
	backElement := c.ll.Back()
	if backElement != nil {
		c.removeElement(backElement)
	}
}

// removeElement 是一个内部辅助函数，用于从链表和map中移除元素。
// 此方法假设已持有锁。
func (c *LRUCache[K, V]) removeElement(e *list.Element) {
	c.ll.Remove(e)
	entry := e.Value.(*entry[K, V])
	delete(c.cache, entry.key)
	c.currentWeight -= entry.weight
}

// Len 返回当前缓存中的条目数量。
func (c *LRUCache[K, V]) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.ll.Len()
}

// Weight 返回当前缓存中所有元素的总权重。
func (c *LRUCache[K, V]) Weight() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.currentWeight
}
