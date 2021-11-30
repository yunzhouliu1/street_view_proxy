package proxy

import (
	"github.com/Daniel-W-Innes/street_view_proxy/view"
	"os"
	"testing"
)

func BenchmarkGetMetadata(b *testing.B) {
	location := view.Location{
		Latitude:  45.389661,
		Longitude: -75.693499,
	}

	for i := 0; i < b.N; i++ {
		_, _ = GetMetadata(&location, os.Getenv("API_KEY"))
	}
}
