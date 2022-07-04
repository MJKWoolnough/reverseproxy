package main

import (
	_ "embed" // required for index embed
	"time"

	"vimagination.zapto.org/httpembed"
)

var (
	//go:embed index.gz
	indexData []byte
	index     = httpembed.HandleBuffer(indexData, 36414, time.Unix(1656937817, 0))
)
