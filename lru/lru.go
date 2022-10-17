package lru

import (
	"container/list"
)

// 封装value接口，方便计算内存使用
type Value interface {
	Len() int
}

// 承载缓存的数据结构，缓存的value使用内置双向链表
type Cache struct {
	maxBytes int64
	nbytes   int64
	ll       *list.List
	cache    map[string]*list.Element
	// 记录被移除时的回调函数 可以为nil
	OnEvicted func(key string, value Value)
}

// 单条数据
type entry struct {
	key   string
	value Value
}

func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// 获取缓存值，并将元素移至队尾
func (c *Cache) Get(key string) (value Value, ok bool) {
	if elem, ok := c.cache[key]; ok {
		c.ll.MoveToFront(elem)
		// entry 类型
		kv := elem.Value.(*entry)
		return kv.value, ok
	}
	return
}

// LRU淘汰策略，循环移除队列头部元素
func (c *Cache) RemoveOldest() {
	elem := c.ll.Back()
	if elem != nil {
		c.ll.Remove(elem)
		kv := elem.Value.(*entry)
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// 添加元素
func (c *Cache) Add(key string, value Value) {
	if elem, ok := c.cache[key]; ok {
		c.ll.MoveToFront(elem)
		kv := elem.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		elem := c.ll.PushFront(&entry{key: key, value: value})
		c.cache[key] = elem
		c.nbytes += int64(len(key)) + int64(value.Len())
	}

	for c.maxBytes != 0 && c.nbytes >= c.maxBytes {
		c.RemoveOldest()
	}
}

// 为了方便测试 获取长度
func (c *Cache) Len() int {
	return c.ll.Len()
}
