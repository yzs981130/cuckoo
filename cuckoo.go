package cuckoo

import (
	"math/rand"
	"sync"
)

type Filter struct {
	buckets []bucket
	size    uint
	// bucketMask: use bitwise-and rather than % len(buckets)
	bucketMask uint

	mu sync.RWMutex

	hashF       func([]byte, uint64) uint64
	seed        uint64
	maxKickouts int

	boolRand boolgen
	fp       []byte
}

func New(maxNumKeys uint) *Filter {
	numBuckets := upperPower2(uint64(maxNumKeys / bucketSize))
	if numBuckets < 1 {
		numBuckets = 1
	}
	if frac := float64(maxNumKeys) / float64(numBuckets*bucketSize); frac > 0.96 {
		numBuckets <<= 1
	}

	bucket := make([]bucket, numBuckets)
	return &Filter{
		buckets:     bucket,
		size:        0,
		bucketMask:  numBuckets - 1,
		maxKickouts: 500,
		seed:        1337,
		hashF:       metroHash,
		boolRand:    boolgen{src: rand.NewSource(1)},
		fp:          make([]byte, 1),
	}
}

// insert fp into buckets[id]
func (f *Filter) insert(id uint, fp uint8) bool {
	if f.buckets[id].insert(fp) {
		f.size++
		return true
	}
	return false
}

func (f *Filter) delete(fp uint8, i uint) bool {
	if f.buckets[i].delete(fp) {
		f.size--
		return true
	}
	return false
}

func (f *Filter) Add(data []byte) bool {
	id1, fp := f.getIndexAndFingerprint(data)
	id2 := f.getAltIndex(fp, id1)
	if f.insert(id1, fp) || f.insert(id2, fp) {
		return true
	}

	// cuckoo
	i := [2]uint{id1, id2}[f.boolRand.Bool()]
	for k := 0; k < f.maxKickouts; k++ {
		// random pick
		j := f.boolRand.Bool() << 1 & f.boolRand.Bool()
		f.buckets[i][j], fp = fp, f.buckets[i][j]
		i = f.getAltIndex(fp, i)
		if f.insert(i, fp) {
			return true
		}
	}
	return false
}

func (f *Filter) Delete(data []byte) bool {
	id1, fp := f.getIndexAndFingerprint(data)
	id2 := f.getAltIndex(fp, id1)
	if f.delete(fp, id1) || f.delete(fp, id2) {
		return true
	}
	return false
}

func (f *Filter) Size() uint {
	return f.size
}

func (f *Filter) SizeInBytes() {

}

func (f *Filter) Contain(data []byte) bool {
	id1, fp := f.getIndexAndFingerprint(data)
	id2 := f.getAltIndex(fp, id1)
	if f.buckets[id1].contains(fp) || f.buckets[id2].contains(fp) {
		return true
	}
	return false
}

// getIndexAndFingerprint returns index1 and fingerprint
func (f *Filter) getIndexAndFingerprint(data []byte) (uint, uint8) {
	hash_ := f.hashF(data, f.seed)
	// index = hash_ % len(bucket)
	i1 := uint(hash_) & f.bucketMask
	// fingerprint of data
	tag := tagHash(hash_)
	return i1, tag
}

// getAltIndex returns alt index of given fingerprint and id1
// (hash(x) \xor hash(fp)) % len(buckets)
func (f *Filter) getAltIndex(fp uint8, oldIndex uint) uint {
	f.fp[0] = fp
	return (oldIndex ^ uint(f.hashF(f.fp, f.seed))) & f.bucketMask
}

func (f *Filter) LoadFactor() float64 {
	return float64(f.size) / float64(len(f.buckets)*bucketSize)
}


func (f *Filter) SafeAdd(data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Add(data)
}

func (f *Filter) SafeContain(data []byte) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.Contain(data)
}