package main

import (
	_ "embed" // required for index embed
	"time"

	"vimagination.zapto.org/httpembed"
)

var (
	//go:embed index.gz
	indexData []byte
	index     = httpembed.HandleBuffer(indexData, 36506, time.Unix(1656327277, 0))
)
