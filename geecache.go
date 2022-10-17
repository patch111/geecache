package geecache

// 缓存流程控制模块
import (
	"fmt"
	"log"
	"sync"
)

// 类似 http Handler
// 抽象出接口 获取数据具体逻辑留给客户端 方便应对各种形式获取数据
type Getter interface {
	Get(key string) ([]byte, error)
}

// 回调函数 GetterFunc
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// 给缓存分装命名空间、回调函数
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	// 节点信息
	peers PeerPicker
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// 实例化group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

// 获取group
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get value for a key from cache
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[Geecache] hit")
		return v, nil
	}
	return g.load(key)
}

// RegisterPeers registers a PeerPicker for choosing remote peer
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// 加载数据
func (g *Group) load(key string) (value ByteView, err error) {
	if g.peers != nil {
		if peer, ok := g.peers.PickPeer(key); ok {
			value, err := g.getFromPeer(peer, key)
			if err != nil {
				return ByteView{}, err
			}
			return value, nil
		}
		log.Println("[GeeCache] Failed to get from peer", err)
	}
	return g.getLocally(key)
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, err
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bt, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
		log.Println("getLocally data Failed!")
	}
	value := ByteView{b: cloneBytes(bt)}
	// 添加至本地缓存
	g.populateCache(key, value)
	return value, err
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
