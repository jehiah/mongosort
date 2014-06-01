package main

type Extent struct {
	MyLoc       DiskLoc // 8 myloc
	NextExtent  DiskLoc // 8 next Extent
	PrevExtent  DiskLoc // 8 prev Extent
	Length      int32   // 4 /* size of the extent, including these fields */
	FirstRecord DiskLoc // 4 first Record
	LastRecord  DiskLoc // 4 last Record
	// 4 _extentData ?
	// ... data
}

type Record struct {
	// 4 lengthWithHeaders
	// 4 extent Offset (from self)
	// 4 next Record
	// 4 prev Record
	// .. data (may be larger than obj if padded)
}

type BtreeBucket struct {
	// Parent
	// R child
	// .. stuff
	// KeyNodes (sorted)
	// - 8 left Child
	// - 8 Data Record
	// - 2 key offset
	// KeyObjects (unsorted)
}
