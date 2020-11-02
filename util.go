package cuckoo

import (
	metro "github.com/dgryski/go-metro"
	"github.com/twmb/murmur3"
)

// upperPower2 does what upperpower2 does in bitsutil.h
func upperPower2(x uint64) uint {
	x--
	x |= x >> 1
	x |= x >> 2
	x |= x >> 4
	x |= x >> 8
	x |= x >> 16
	x |= x >> 32
	x++
	return uint(x)
}

func metroHash(data []byte, seed uint64) uint64 {
	return metro.Hash64(data, seed)
}

func murmur3Hash(data []byte, seed uint64) uint64 {
	hash_ := murmur3.SeedNew64(seed)
	_, _ = hash_.Write(data)
	return hash_.Sum64()
}

func tagHash(hash_ uint64) uint8 {
	// reserve 0 for empty
	// return [1, 2^8]
	return uint8(hash_%255-1) + 1
}
