package main

import (
	"net/http/httptest"
	"testing"
)

func BenchmarkView(b *testing.B) {
	*log_filename = "testdata"
	for i := 0; i < b.N; i++ {
		view_handler(httptest.NewRecorder(), nil)
	}
}
