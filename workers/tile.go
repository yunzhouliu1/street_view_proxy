package workers

import (
	"github.com/Daniel-W-Innes/street_view_proxy/config"
	"github.com/Daniel-W-Innes/street_view_proxy/proxy"
	"image"
	"image/draw"
	"log"
	"runtime"
)

type TileRequest struct {
	x, y, zoom int
	panoId     string
}

type Tile struct {
	image       image.Image
	tileRequest TileRequest
	error       error
}
type TileWorker struct {
	Input  chan *TileRequest
	Exit   chan struct{}
	Output chan *Tile
}

func (t *TileWorker) GetTileWorker() {
	for {
		select {
		case tileRequest := <-t.Input:
			tile, err := proxy.GetTile(tileRequest.x, tileRequest.y, tileRequest.zoom, tileRequest.panoId)
			if err != nil {
				t.Output <- &Tile{error: err}
			} else {
				t.Output <- &Tile{image: *tile, tileRequest: *tileRequest}
			}
		case <-t.Exit:
			return
		}
	}
}

func (t *TileWorker) DownloadMosaic(metadata *proxy.Metadata) {
	for x := config.MinX; x < config.MaxX; x++ {
		for y := config.MinY; y < config.MaxY; y++ {
			t.Input <- &TileRequest{x: x, y: y, zoom: config.Zoom, panoId: metadata.PanoId}
		}
	}
}

func (t *TileWorker) GetMosaic() (image.Image, error) {
	rgba := image.NewRGBA(image.Rect(0, 0, config.OutputSizeX, config.OutputSizeY))
	for x := config.MinX; x < config.MaxX; x++ {
		for y := config.MinY; y < config.MaxY; y++ {
			output := <-t.Output
			if output.error != nil {
				log.Printf("err from X: %d, y: %d\n", x, y)
			} else {
				draw.Draw(rgba, image.Rect(
					(output.tileRequest.x-config.MinX)*config.InputSizeX, (output.tileRequest.y-config.MinY)*config.InputSizeY,
					(output.tileRequest.x+1-config.MinX)*config.InputSizeX, (output.tileRequest.y+1-config.MinY)*config.InputSizeY),
					output.image, image.Point{}, draw.Src)
			}
		}
	}
	return rgba, nil
}

func GetTileWorkers() *TileWorker {
	tileWorker := TileWorker{
		Input:  make(chan *TileRequest, (config.MaxY-config.MinY)*(config.MaxX-config.MinX)),
		Exit:   make(chan struct{}),
		Output: make(chan *Tile, (config.MaxY-config.MinY)*(config.MaxX-config.MinX)),
	}
	for i := 0; i < runtime.NumCPU()*config.WorkerMultiplier; i++ {
		go tileWorker.GetTileWorker()
	}
	return &tileWorker
}
