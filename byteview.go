package geecache

// 方便支持多种类型缓存
type ByteView struct {
	b []byte
}

// 获取长度
func (v ByteView) Len() int {
	return len(v.b)
}

// 拷贝一份，防止缓存之被外部修改 只读
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// 转换string类型
func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
