package cuckoo

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"
)

func TestInsertion(t *testing.T) {
	cf := New(1000000)
	fd, err := os.Open("/usr/share/dict/words")
	if err != nil {
		t.Fatalf("failed reading words: %v", err)
	}
	scanner := bufio.NewScanner(fd)

	var values [][]byte
	var lineCount uint
	for scanner.Scan() {
		s := []byte(scanner.Text())
		if cf.Add(s) {
			lineCount++
		}
		values = append(values, s)
	}

	if got, want := cf.Size(), lineCount; got != want {
		t.Errorf("After inserting: Count() = %d, want %d", got, want)
	}

	for _, v := range values {
		cf.Delete(v)
	}

	if got, want := cf.Size(), uint(0); got != want {
		t.Errorf("After deleting: Count() = %d, want %d", got, want)
	}
	if got, want := cf.LoadFactor(), float64(0); got != want {
		t.Errorf("After deleting: LoadFactor() = %f, want %f", got, want)
	}
}

func TestLookup(t *testing.T) {
	cf := New(4)
	cf.Add([]byte("one"))
	cf.Add([]byte("two"))
	cf.Add([]byte("three"))

	testCases := []struct {
		word string
		want bool
	}{
		{"one", true},
		{"two", true},
		{"three", true},
		{"four", false},
		{"five", false},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("cf.Lookup(%q)", tc.word), func(t *testing.T) {
			t.Parallel()
			if got := cf.Contain([]byte(tc.word)); got != tc.want {
				t.Errorf("cf.Lookup(%q) got %v, want %v", tc.word, got, tc.want)
			}
		})
	}
}

func TestFilter_Insert(t *testing.T) {
	filter := New(10000)

	var hash [32]byte

	for i := 0; i < 100; i++ {
		io.ReadFull(rand.Reader, hash[:])
		filter.Add(hash[:])
	}

	if got, want := filter.Size(), uint(100); got != want {
		t.Errorf("inserting 100 items, Count() = %d, want %d", got, want)
	}
}

func BenchmarkFilter_Insert(b *testing.B) {
	filter := New(10000)

	b.ResetTimer()

	var hash [32]byte
	for i := 0; i < b.N; i++ {
		io.ReadFull(rand.Reader, hash[:])
		filter.Add(hash[:])
	}
}

func BenchmarkFilter_Lookup(b *testing.B) {
	filter := New(10000)

	var hash [32]byte
	for i := 0; i < 10000; i++ {
		io.ReadFull(rand.Reader, hash[:])
		filter.Add(hash[:])
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		io.ReadFull(rand.Reader, hash[:])
		filter.Contain(hash[:])
	}
}

func TestDelete(t *testing.T) {
	cf := New(8)
	cf.Add([]byte("one"))
	cf.Add([]byte("two"))
	cf.Add([]byte("three"))

	testCases := []struct {
		word string
		want bool
	}{
		{"four", false},
		{"five", false},
		{"one", true},
		{"two", true},
		{"three", true},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("cf.Delete(%q)", tc.word), func(t *testing.T) {
			if got := cf.Delete([]byte(tc.word)); got != tc.want {
				t.Errorf("cf.Delete(%q) got %v, want %v", tc.word, got, tc.want)
			}
		})
	}
}

func TestDeleteMultipleSame(t *testing.T) {
	cf := New(4)
	for i := 0; i < 5; i++ {
		cf.Add([]byte("some_item"))
	}

	testCases := []struct {
		word      string
		want      bool
		wantCount uint
	}{
		{"missing", false, 5},
		{"missing2", false, 5},
		{"some_item", true, 4},
		{"some_item", true, 3},
		{"some_item", true, 2},
		{"some_item", true, 1},
		{"some_item", true, 0},
		{"some_item", false, 0},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("cf.Delete(%q)", tc.word), func(t *testing.T) {
			if got, gotCount := cf.Delete([]byte(tc.word)), cf.Size(); got != tc.want || gotCount != tc.wantCount {
				t.Errorf("cf.Delete(%q) = %v, count = %d; want %v, count = %d", tc.word, got, gotCount, tc.want, tc.wantCount)
			}
		})
	}
}

func TestThreadSafe(t *testing.T) {
	f := New(1000)

	testCases := []struct {
		item byte
		want bool
	}{
		{1, true},
		{99, false},
	}

	var wg sync.WaitGroup
	for i := byte(0); i < 50; i++ {
		wg.Add(1)
		go func(item byte) {
			defer wg.Done()
			f.SafeAdd([]byte{item})
		}(i)
	}

	for i := byte(0); i < 100; i++ {
		wg.Add(1)
		go func(item byte) {
			defer wg.Done()
			f.SafeContain([]byte{item})
		}(i)
	}
	wg.Wait()

	for _, tc := range testCases {
		if got := f.SafeContain([]byte{tc.item}); got != tc.want {
			t.Errorf("cf.SafeContain(%d) = %v, want %v", tc.item, got, tc.want)
		}
	}

}