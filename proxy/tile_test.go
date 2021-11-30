package proxy

import "testing"

const panoId = "NByXiiB08r9stAGnKIAe2w"

func BenchmarkGetTile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GetTile(16, 8, 5, panoId)
	}
}
