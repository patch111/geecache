package geecache

// 节点选择
type PeerPicker interface {
	//寻找持有key的节点peer
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// peer 从group实例中 寻找key对应的value
type PeerGetter interface {
	Get(group string, key string) ([]byte, error)
}
