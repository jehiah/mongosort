package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
)

// structure is taken from 
// https://github.com/mongodb/mongo/blob/v2.0/db/namespace.h#L132

type HashNode struct {
	Offset           int64
	Hash             int32             // 4byte
	Namespace        string            // 128 bytes
	NamespaceDetails *NamespaceDetails // 496 bytes
} // 628 bytes total
const hashNodeSize = 628

func nullTerminatedString(b []byte) string {
	chunks := bytes.SplitAfterN(b, []byte("\x00"), 2)
	return string(chunks[0])
}

func ReadHashNode(f *os.File, offset int64) (*HashNode, error) {
	b := make([]byte, hashNodeSize)
	l, err := f.ReadAt(b, offset)
	if err != nil {
		return nil, err
	}
	if l != hashNodeSize {
		return nil, errors.New("didn't read 628 bytes")
	}
	r := bytes.NewReader(b)
	h := &HashNode{Offset: offset}

	binary.Read(r, binary.LittleEndian, &h.Hash)
	if h.Hash == 0 {
		return h, nil
	}
	h.Namespace = nullTerminatedString(b[4 : 128+4])

	// NamespaceDetails := b[128+4:]

	return h, nil
}

func (h *HashNode) String() string {
	return h.Namespace
}

type NamespaceDetails struct {
	FirstExtent DiskLoc     // firstExtent
	LastExtent  DiskLoc     // lastExtent
	DeletedList [19]DiskLoc // deletedList
	// ofs 168 (8 byte aligned)
	Datasize      int64 // datasize; this includes padding, but not record headers
	NumberRecords int64 //nrecords

	LastExtentSize int32 //lastExtentSize
	NumberIndexes  int32 //nindexes

	// // ofs 192
	// IndexDetails _indexes[NIndexesBase];

	/*-------- end data 496 bytes */
}

type DiskLoc struct {
	FileCounter int32
	Offset      int32
}

// NS file; assert len % (1024*1024) == 0

// Delteed is a set of buckets of appropriate sized deleted records
// int bucketSizes[] = {
//         32, 64, 128, 256, 0x200, 0x400, 0x800, 0x1000, 0x2000, 0x4000,
//         0x8000, 0x10000, 0x20000, 0x40000, 0x80000, 0x100000, 0x200000,
//         0x400000, 0x800000
//     };

// $freelist for extents
