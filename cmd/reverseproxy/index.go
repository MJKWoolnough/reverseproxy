package main

import (
	_ "embed" // required for index embed
	"time"

	"vimagination.zapto.org/httpembed"
)

var (
	//go:embed index.gz
	indexData []byte
	index     = httpembed.HandleBuffer(indexData, 36510, time.Unix(1656846830, 0))
)
