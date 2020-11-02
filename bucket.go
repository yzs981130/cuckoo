package cuckoo

const (
	bucketSize = 4 // assoc
	empty      = 0
)

// 8bit fingerprint
type bucket [bucketSize]uint8

func (b *bucket) insert(fp uint8) bool {
	for i, tfp := range b {
		if tfp == empty {
			b[i] = fp
			return true
		}
	}
	return false
}

func (b *bucket) delete(fp uint8) bool {
	for i, tfp := range b {
		if tfp == fp {
			b[i] = empty
			return true
		}
	}
	return false
}

func (b *bucket) contains(fp uint8) bool {
	for _, fp_ := range b {
		if fp_ == fp {
			return true
		}
	}
	return false
}
