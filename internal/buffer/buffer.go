package buffer

import (
	"sync"
)

const (
	bufferSize = 8 << 10 // 8KB
)

type Buffer [bufferSize]byte

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(Buffer)
	},
}

func Get() *Buffer {
	return bufferPool.New().(*Buffer)
}

func Put(buf *Buffer) {
	bufferPool.Put(buf)
}

type BufferLength [4]byte

func (b BufferLength) ReadUint() uint {
	return uint(b[0] |
		b[1]<<8 |
		b[2]<<16 |
		b[3]<<24,
	)
}

func (b *BufferLength) WriteUint(u uint) {
	*b = [4]byte{
		byte(u),
		byte(u >> 8),
		byte(u >> 16),
		byte(u >> 24),
	}
}
