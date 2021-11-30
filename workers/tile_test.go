package workers

import (
	"github.com/Daniel-W-Innes/street_view_proxy/proxy"
	"testing"
)

const panoId = "NByXiiB08r9stAGnKIAe2w"

func BenchmarkTileWorker_GetMosaic(b *testing.B) {
	tileWorker := GetTileWorkers()

	metadata := proxy.Metadata{
		PanoId: panoId,
	}

	for i := 0; i < b.N; i++ {
		go tileWorker.DownloadMosaic(&metadata)
		_, _ = tileWorker.GetMosaic()
	}

	tileWorker.Exit <- struct{}{}
}
