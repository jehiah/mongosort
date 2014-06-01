package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"labix.org/v2/mgo/bson"
)

// structure is taken from
// https://github.com/mongodb/mongo/blob/v2.0/db/namespace.h#L132

func ReadNamespace(f *os.File) (*Namespace, error) {
	n := &Namespace{File: f}
	s, err := f.Stat()
	if err != nil {
		log.Printf("failed stating file %s", err)
		return nil, err
	}
	n.Dir = path.Dir(f.Name())
	size := s.Size()
	if size%(1024*1024) != 0 {
		return nil, fmt.Errorf("file size %d must be multiple of 1048576", size)
	}

	var i int64
	for ; i <= size-hashNodeSize; i += hashNodeSize {
		h, err := ReadHashNode(f, i, n.Dir)
		if err != nil {
			return nil, fmt.Errorf("error reading at %d of size %d %s", i, size, err)
		}
		n.HashTable = append(n.HashTable, h)
		if h.Hash == 0 {
			continue
		}
	}
	return n, nil
}

type Namespace struct {
	File      *os.File
	Dir       string
	HashTable []*HashNode
}

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

func ReadHashNode(f *os.File, offset int64, dir string) (*HashNode, error) {
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

	namespaceDetailsReader := bytes.NewReader(b[128+4:])
	nd, err := ReadNamespaceDetails(namespaceDetailsReader)
	nd.NamespaceBase = strings.Split(h.Namespace, ".")[0]
	if err != nil {
		return nil, err
	}
	h.NamespaceDetails = nd
	h.NamespaceDetails.Dir = dir
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
	IndexDetails [10]IndexDetails // _indexes

	/*-------- end data 496 bytes */
	NamespaceBase string
	Dir           string
}

func (nd *NamespaceDetails) String() string {

	return fmt.Sprintf("first: %s, last: %s deletedList:%s size: %d records:%d extentSize: %d ",
		nd.FirstExtent,
		nd.LastExtent,
		nd.DeletedList,
		nd.Datasize,
		nd.NumberRecords,
		nd.LastExtentSize)
}

func (nd *NamespaceDetails) DumpIndexDetails() {
	log.Printf("\tIndexes: %d", nd.NumberIndexes)
	for _, i := range nd.IndexDetails[:nd.NumberIndexes] {
		o, err := i.Info.GetBsonObj(nd.Dir, nd.NamespaceBase)
		if err != nil {
			log.Fatalf("failed getting %s", err)
		}
		log.Printf("\t\t%s %#v", i.Head, o)
	}
}

func ReadNamespaceDetails(r io.Reader) (*NamespaceDetails, error) {
	nd := &NamespaceDetails{}
	if err := binary.Read(r, binary.LittleEndian, &nd.FirstExtent); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &nd.LastExtent); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &nd.DeletedList); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &nd.Datasize); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &nd.NumberRecords); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &nd.LastExtentSize); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &nd.NumberIndexes); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &nd.IndexDetails); err != nil {
		return nil, err
	}
	return nd, nil
}

type IndexDetails struct {
	Head DiskLoc
	Info DiskLoc
}

func (id IndexDetails) String() string {
	// todo: get bson for .Info
	return fmt.Sprintf("<Index head:%s info: %s>", id.Head, id.Info)
}

type DiskLoc struct {
	FileCounter int32
	Offset      int32
}

type Info struct {
	Key string `bson:"key"`
	Ns  string `bson:"ns"`
}

func (dl DiskLoc) GetBsonObj(dir string, namespace string) (interface{}, error) {
	file := path.Join(dir, fmt.Sprintf("%s.%d", namespace, dl.FileCounter))
	f, err := os.Open(file)
	if err != nil {
		log.Printf("failed opening %s %s", file, err)
		return nil, err
	}
	f.Seek(int64(dl.Offset), 0)
	var dataSize int32
	if err := binary.Read(f, binary.LittleEndian, &dataSize); err != nil {
		return nil, err
	}

	// this is record packed
	// namespace_details_collection_entry.cpp line

	f.Seek(recordHeader-4, 1)
	// log.Printf("is %d size", dataSize)
	b := make([]byte, dataSize-recordHeader)
	var data interface{}
	l, err := f.Read(b)
	if int32(l) != dataSize-recordHeader || (err != nil && err != io.EOF) {
		if err != nil {
			return nil, err
		} else {
			return nil, fmt.Errorf("unexpected read of %d for size %d", l, dataSize)
		}
	}
	if b[dataSize-recordHeader-1] != '\x00' {
		return nil, fmt.Errorf("bson not null terminated %q", b)
	}
	if err := bson.Unmarshal(b, &data); err != nil {
		log.Printf("raw bson is %q", b)
		log.Printf("failed unmarshaling %s %s", b, err)
		return nil, err
	}
	return data, nil
}

func (d DiskLoc) String() string {
	if d.FileCounter < 0 {
		return "{}"
	}
	return fmt.Sprintf("{%d offset:%d}", d.FileCounter, d.Offset)
}

// NS file; assert len % (1024*1024) == 0

// Delteed is a set of buckets of appropriate sized deleted records
// int bucketSizes[] = {
//         32, 64, 128, 256, 0x200, 0x400, 0x800, 0x1000, 0x2000, 0x4000,
//         0x8000, 0x10000, 0x20000, 0x40000, 0x80000, 0x100000, 0x200000,
//         0x400000, 0x800000
//     };

// $freelist for extents
