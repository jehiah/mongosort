package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	fileName := flag.String("filename", "", "")
	flag.Parse()

	f, err := os.Open(*fileName)
	if err != nil {
		log.Fatalf("failed opening %s %s", *fileName, err)
	}
	s, err := os.Stat(*fileName)
	if err != nil {
		log.Fatalf("failed stating %s %s", *fileName, err)
	}
	size := s.Size()
	log.Printf("%s size %d", *fileName, size)

	var i int64
	blockSize := int64(628)
	for ; i < size; i += blockSize {
		if i+4+128 > size {
			break
		}
		h, err := ReadHashNode(f, i)
		if err != nil {
			log.Fatalf("err reading at %d %s", i, err)
		}
		if h.Hash == 0 {
			continue
		}
		log.Printf("at %d hashtable entry %d %s", i, h.Hash, h.String())
	}
}
