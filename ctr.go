package seekctr

import (
	"crypto/cipher"
	"encoding/binary"
)

const streamBufferSize = 512

type ctr struct {
	b       cipher.Block
	ctr     []byte
	iv      []byte
	out     []byte
	outUsed int
}

func newCTR(block cipher.Block, iv []byte) *ctr {
	if len(iv) != block.BlockSize() {
		panic("IV length must equal block size")
	}
	bufSize := streamBufferSize
	if bs := block.BlockSize(); bufSize < bs {
		bufSize = bs
	}
	return &ctr{block, dup(iv), dup(iv), make([]byte, 0, bufSize), 0}
}

func (x *ctr) XORKeyStream(dst, src []byte) {
	for len(src) > 0 {
		if x.outUsed >= len(x.out)-x.b.BlockSize() {
			x.refill()
		}
		n := xor(dst, src, x.out[x.outUsed:])
		dst, src = dst[n:], src[n:]
		x.outUsed += n
	}
}

// seek will take the offset and then divide it by the blocksize for the chunk
// offset before refilling the buffer.
func (x *ctr) seek(offset int64) {
	// offset in chunks
	chunks := uint64(int(offset) / x.b.BlockSize())
	// convert chunks to []byte
	b := make([]byte, len(x.iv))
	binary.BigEndian.PutUint64(b[len(b)-8:], chunks)

	// add x.iv (a) and chunks (b) with the result being x.ctr and c
	// representing the carry
	var c uint16
	for i := len(b) - 1; i >= 0; i-- {
		c = uint16(x.iv[i]) + uint16(b[i]) + c
		x.ctr[i] = byte(c)
		c >>= 8
	}

	x.outUsed = len(x.out)
	x.refill()
	x.outUsed = int(offset) % x.b.BlockSize()
}

func (x *ctr) refill() {
	remain := len(x.out) - x.outUsed
	copy(x.out, x.out[x.outUsed:])
	x.out = x.out[:cap(x.out)]
	bs := x.b.BlockSize()
	for remain <= len(x.out)-bs {
		x.b.Encrypt(x.out[remain:], x.ctr)
		remain += bs
		for i := len(x.ctr) - 1; i >= 0; i-- {
			x.ctr[i]++
			if x.ctr[i] != 0 {
				break
			}
		}
	}
	x.out = x.out[:remain]
	x.outUsed = 0
}

func dup(b []byte) []byte {
	buf := make([]byte, len(b))
	copy(buf, b)
	return buf
}

func xor(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
	return n
}
