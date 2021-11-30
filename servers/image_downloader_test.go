package servers

import (
	"github.com/Daniel-W-Innes/street_view_proxy/view"
	"github.com/Daniel-W-Innes/street_view_proxy/workers"
	"os"
	"testing"
)

func BenchmarkGetImage(b *testing.B) {
	tileWorker := workers.GetTileWorkers()
	location := view.Location{
		Latitude:  45.389661,
		Longitude: -75.693499,
	}

	for i := 0; i < b.N; i++ {
		_, _ = getImage(tileWorker, &location, os.Getenv("API_KEY"), false)
	}
}
