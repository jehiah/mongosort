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
	namespace, err := ReadNamespace(f)
	if err != nil {
		log.Fatalf("%s", err)
	}
	for _, hn := range namespace.HashTable {
		if hn.Hash != 0 {
			log.Printf("at file offset %d hashtable entry %d for %s", hn.Offset, hn.Hash, hn.Namespace)
			log.Printf("\t%s", hn.NamespaceDetails)
			if hn.NamespaceDetails.NumberIndexes != 0 {
				hn.NamespaceDetails.DumpIndexDetails()
			}
		}
	}
}
