package lru

import "container/list"

// Cache is a LRU cache. It is not safe for concurrent access.
type Cache struct {
	//允许使用的最大内存
	maxBytes int64
	//当前已使用的内存
	nbytes int64
	ll     *list.List
	cache  map[string]*list.Element
	//某条记录被移除时的回调函数
	OnEvicted func(key string, value Value)
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

// 如果键对应的链表节点存在，则将对应节点移动到队尾，并返回查找到的值
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

//缓存淘汰。即移除最近最少访问的节点
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		//删除尾巴
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		//从map删除
		delete(c.cache, kv.key)
		//修改当前大小
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		//回调
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		//如果键存在，则更新对应节点的值，并将该节点移到队尾。
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		//调整大小，key不变，只需要value差值
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		//不存在则是新增场景，首先队尾添加新节点 &entry{key, value}, 并字典中添加 key 和节点的映射关系。
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}

	//更新 c.nbytes，如果超过了设定的最大值 c.maxBytes，则移除最少访问的节点
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

func (c *Cache) Len() int {
	return c.ll.Len()
}
