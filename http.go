package geecache

import (
	"fmt"
	"geecache/consistenthash"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)

// 节点通信使用 url :http://127.0.0.1:8001/_geecache/
type HTTPPool struct {
	// 节点url e.g."http://127.0.0.1:8001"
	self string
	// 默认 defaultBasePath = "/_geecache/"
	basePath string
	mu       sync.Mutex
	// 一致性哈希 Map
	peers *consistenthash.Map
	// 映射远程节点与对应的 httpGetter
	httpGetters map[string]*httpGetter
}

// http客户端
type httpGetter struct {
	// 封装访问远程节点url http://127.0.0.1:80001/_geecache/
	baseURL string
}

// 实例化HTTPPool
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// httpGetter 实现 PeerGetter 接口
func (h *httpGetter) Get(groupName string, key string) ([]byte, error) {
	u := fmt.Sprintf("%v/%v/%v", h.baseURL, url.QueryEscape(groupName), url.QueryEscape(key))
	// get请求 url : eg. http://127.0.0.1:80001/_geecache/groupname/key
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}
	// 获取返回的value
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("server returned : %v", res.Status)
	}
	return bytes, nil
}

var _ PeerGetter = (*httpGetter)(nil)

// 打印server name url
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// 实例化一致性哈希算法，并添加传入的节点。
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

var _ PeerPicker = (*HTTPPool)(nil)

// HTTPPool 更具具体的key 创建 HTTP 客户端从远程节点获取缓存值
func (p *HTTPPool) PickPeer(key string) (peer PeerGetter, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

// 其他节点通过get请求 获取缓存数据
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// 约定访问url为:/<basepath>/<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// 通过 groupname获取group实例 得到key-value
	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// 返回value
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}
