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
	log.Printf("Database Namespace: %s", *fileName)
	for _, hn := range namespace.HashTable {
		if hn.Hash != 0 {
			log.Printf("collection %s", hn.Namespace)
			log.Printf("\t%s", hn.NamespaceDetails)
			if hn.NamespaceDetails.NumberIndexes != 0 {
				hn.NamespaceDetails.DumpIndexDetails()
			}
		}
	}
}
