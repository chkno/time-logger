package main

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func BenchmarkView(b *testing.B) {
	buf, err := ioutil.ReadFile("testdata")
	if err != nil {
		b.Fatal("Could not read testdata", err)
	}
	reader := bytes.NewReader(buf)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader.Seek(0, 0)
		view_handler(reader, ioutil.Discard)
	}
}
