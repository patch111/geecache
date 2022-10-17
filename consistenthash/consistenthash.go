package consistenthash

// 一致性哈希 节点选择模块
import (
	"hash/crc32"
	"sort"
	"strconv"
)

// 可插拔的哈希函数  默认crc32.ChecksumIEEE
type Hash func(data []byte) uint32

// 一致性哈希数据结构
type Map struct {
	hash Hash
	// 虚拟节点倍数
	replicas int
	// 哈希环
	keys []int
	// 虚拟节点和真实节点映射关系
	hashMap map[int]string
}

// 实例化 Map  哈希函数可自定义
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 添加节点
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		// 根据虚拟节点倍数添加
		for i := 0; i < m.replicas; i++ {
			hashv := int(m.hash([]byte(strconv.Itoa(i) + key)))
			// 将哈希值添加到哈希环
			m.keys = append(m.keys, hashv)
			// 映射关系
			m.hashMap[hashv] = key
		}
	}
	// 排序
	sort.Ints(m.keys)
}

// 根据key的哈希值 在哈希环上寻找最近的节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	hashv := int(m.hash([]byte(key)))
	// 二分法查找
	indx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hashv
	})
	// 返回真实节点
	return m.hashMap[m.keys[indx%len(m.keys)]]
}
