package address

import "sort"

type (
	// fastStore is a fast read only map from string to Address object
	// Lookups are about 5x faster than the built-in Go map type
	fastStore struct {
		store []objectValue
	}

	objectValue struct {
		nextLo     uint32   // index in store of next byteValues
		nextLen    byte     // number of byteValues in store used for next possible bytes
		nextOffset byte     // offset from zero byte value of first element of range of byteValues
		valid      bool     // is the byte sequence with no more bytes in the map?
		value      *Address // value for byte sequence with no more bytes
	}

	// ObjectsSource is for supplying data to initialise fastStore
	ObjectsSource interface {
		// AppendKeys should append the keys of the maps to the supplied slice and return the resulting slice
		AppendKeys([]string) []string
		// Get should return the value for the supplied key
		Get(string) *Address
	}

	// mapBuilder is used only during construction
	mapBuilder struct {
		all [][]objectValue
		src ObjectsSource
		len int
	}
)

// newAddressMemStore creates from the data supplied in src
func newAddressMemStore(src ObjectsSource) fastStore {
	if keys := src.AppendKeys([]string(nil)); len(keys) > 0 {
		sort.Strings(keys)
		return fastStore{store: objectBuild(keys, src)}
	}
	return fastStore{store: []objectValue{{}}}
}

// objectBuild constructs the map by allocating memory in blocks
// and then copying into the eventual slice at the end. This is
// more efficient than continually using append.
func objectBuild(keys []string, src ObjectsSource) []objectValue {
	b := mapBuilder{
		all: [][]objectValue{make([]objectValue, 1, firstBufSize(len(keys)))},
		src: src,
		len: 1,
	}
	b.makeByteValue(&b.all[0][0], keys, 0)
	// copy all blocks to one slice
	s := make([]objectValue, 0, b.len)
	for _, a := range b.all {
		s = append(s, a...)
	}
	return s
}

// makeByteValue will initialise the supplied objectValue for
// the sorted strings in slice a considering bytes at byteIndex in the strings
func (b *mapBuilder) makeByteValue(bv *objectValue, a []string, byteIndex int) {
	// if there is a string with no more bytes then it is always first because they are sorted
	if len(a[0]) == byteIndex {
		bv.valid = true
		bv.value = b.src.Get(a[0])
		a = a[1:]
	}
	if len(a) == 0 {
		return
	}
	bv.nextOffset = a[0][byteIndex]       // lowest value for next byte
	bv.nextLen = a[len(a)-1][byteIndex] - // highest value for next byte
		bv.nextOffset + 1 // minus lowest value +1 = number of possible next bytes
	bv.nextLo = uint32(b.len)   // first objectValue struct in eventual built slice
	next := b.alloc(bv.nextLen) // new byteValues default to "not valid"

	for i, n := 0, len(a); i < n; {
		// find range of strings starting with the same byte
		iSameByteHi := i + 1
		for iSameByteHi < n && a[iSameByteHi][byteIndex] == a[i][byteIndex] {
			iSameByteHi++
		}
		b.makeByteValue(&next[(a[i][byteIndex]-bv.nextOffset)], a[i:iSameByteHi], byteIndex+1)
		i = iSameByteHi
	}
}

const maxBuildBufSize = 1 << 20

func firstBufSize(mapSize int) int {
	size := 1 << 4
	for size < mapSize && size < maxBuildBufSize {
		size <<= 1
	}
	return size
}

// alloc will grab space in the current block if available or allocate a new one if not
func (b *mapBuilder) alloc(nByteValues byte) []objectValue {
	n := int(nByteValues)
	b.len += n
	cur := &b.all[len(b.all)-1] // current
	curCap, curLen := cap(*cur), len(*cur)
	if curCap-curLen >= n { // enough space in current
		*cur = (*cur)[: curLen+n : curCap]
		return (*cur)[curLen:]
	}
	newCap := curCap * 2
	for newCap < n {
		newCap *= 2
	}
	if newCap > maxBuildBufSize {
		newCap = maxBuildBufSize
	}
	a := make([]objectValue, n, newCap)
	b.all = append(b.all, a)
	return a
}

// LookupString looks up the supplied string in the map
func (m *fastStore) LookupString(s string) (*Address, bool) {
	bv := &m.store[0]
	for i, n := 0, len(s); i < n; i++ {
		b := s[i]
		if b < bv.nextOffset {
			return nil, false
		}
		ni := b - bv.nextOffset
		if ni >= bv.nextLen {
			return nil, false
		}
		bv = &m.store[bv.nextLo+uint32(ni)]
	}
	return bv.value, bv.valid
}

// LookupBytes looks up the supplied byte slice in the map
func (m *fastStore) LookupBytes(s []byte) (*Address, bool) {
	bv := &m.store[0]
	for _, b := range s {
		if b < bv.nextOffset {
			return nil, false
		}
		ni := b - bv.nextOffset
		if ni >= bv.nextLen {
			return nil, false
		}
		bv = &m.store[bv.nextLo+uint32(ni)]
	}
	return bv.value, bv.valid
}
