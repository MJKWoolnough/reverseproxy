package main

import (
	"bytes"
	"compress/gzip"
	_ "embed" // required to embed index.gz
	"net/http"
	"time"

	"vimagination.zapto.org/httpencoding"
	"vimagination.zapto.org/memio"
)

var (
	//go:embed index.gz
	compressedIndex   []byte
	uncompressedIndex []byte
	indexUpdatedTime  time.Time
	isGzip            = httpencoding.HandlerFunc(func(enc httpencoding.Encoding) bool { return enc == "gzip" })
)

func index(w http.ResponseWriter, r *http.Request) {
	var b *bytes.Reader
	if httpencoding.HandleEncoding(r, isGzip) {
		b = bytes.NewReader(compressedIndex)
		w.Header().Add("Content-Encoding", "gzip")
	} else {
		b = bytes.NewReader(uncompressedIndex)
	}
	http.ServeContent(w, r, "index.html", indexUpdatedTime, b)
}

func init() {
	uncompressedIndex = make([]byte, uncompressedSize)
	r := memio.Buffer(compressedIndex)
	g, _ := gzip.NewReader(&r)
	g.Read(uncompressedIndex)
	indexUpdatedTime = time.Unix(indexUpdated, 0)
}
