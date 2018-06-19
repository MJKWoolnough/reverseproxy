package buffer

import (
	"sync"

	"vimagination.zapto.org/memio"
)

const (
	bufferSize = 8 << 10 // 8KB
)

type Buffer struct {
	buffer *[bufferSize]byte
	memio.LimitedBuffer
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new([bufferSize]byte)
	},
}

func (b *Buffer) Init() {
	if b.buffer == nil {
		b.buffer = bufferPool.New().(*[bufferSize]byte)
	}
	b.LimitedBuffer = memio.LimitedBuffer((*b.buffer)[:0])
}

func (b *Buffer) AsSlice() []byte {
	return (*b.buffer)[:bufferSize-cap(b.LimitedBuffer)+len(b.LimitedBuffer)]
}

func (b *Buffer) Skip(n int) {
	if n > len(b.LimitedBuffer) {
		n = len(b.LimitedBuffer)
	}
	b.LimitedBuffer = b.LimitedBuffer[n:]
}

func (b *Buffer) Close() error {
	b.LimitedBuffer = nil
	if b.buffer != nil {
		bufferPool.Put(b.buffer)
		b.buffer = nil
	}
	return nil
}
