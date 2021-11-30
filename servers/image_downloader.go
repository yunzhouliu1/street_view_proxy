package servers

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/Daniel-W-Innes/street_view_proxy/config"
	"github.com/Daniel-W-Innes/street_view_proxy/proxy"
	"github.com/Daniel-W-Innes/street_view_proxy/view"
	"github.com/Daniel-W-Innes/street_view_proxy/workers"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
)

func saveImage(img image.Image, path string) error {
	out, err := os.Create(path + ".jpeg")
	if err != nil {
		return err
	}
	var opt jpeg.Options
	opt.Quality = 100

	err = jpeg.Encode(out, img, &opt)
	if err != nil {
		return err
	}
	return nil
}

type ImageDownloaderServer struct {
	ApiKey string
	view.UnimplementedImageDownloaderServer
}

func getImage(tileWorker *workers.TileWorker, location *view.Location, key string, saveImages bool) (*view.Response, error) {
	log.Println("getting metadata")
	metadata, err := proxy.GetMetadata(location, key)
	log.Printf("got metadata: %v\n", metadata)
	if err != nil {
		return nil, err
	}
	if metadata.Status != "OK" {
		return nil, errors.New(metadata.Status)
	}
	log.Println("downloading mosaic")
	go tileWorker.DownloadMosaic(metadata)
	tile, err := tileWorker.GetMosaic()
	log.Println("downloaded mosaic")
	if err != nil {
		return nil, err
	}
	log.Println("encoding response")
	buf := new(bytes.Buffer)
	enc := &png.Encoder{
		CompressionLevel: png.NoCompression,
	}
	err = enc.Encode(buf, tile)
	if err != nil {
		return nil, err
	}
	if saveImages {
		go func() {
			err := saveImage(tile, fmt.Sprintf("%f,%f_x:%d-%d_y:%d-%d_%d", metadata.Location.Latitude, metadata.Location.Longitude, config.MinX, config.MaxX, config.MinY, config.MaxY, config.Zoom))
			if err != nil {
				log.Println("failed to save image")
			}
		}()
	}
	outImage := view.Image{
		Width:     int32(tile.Bounds().Dx()),
		Height:    int32(tile.Bounds().Dy()),
		ImageData: buf.Bytes(),
	}
	log.Println("sending response")
	return &view.Response{Image: &outImage}, nil
}

func (s *ImageDownloaderServer) GetImage(server view.ImageDownloader_GetImageServer) error {
	log.Println("opened request channel")
	tileWorker := workers.GetTileWorkers()
	log.Println("created workers")
	for {
		log.Println("ready for request")
		in, err := server.Recv()
		log.Println("got request")
		if err != nil {
			tileWorker.Exit <- struct{}{}
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		location := in.Location
		if location != nil {
			response, err := getImage(tileWorker, location, s.ApiKey, config.SaveImages)
			if err != nil {
				tileWorker.Exit <- struct{}{}
				return server.Send(&view.Response{Error: &view.Error{Description: err.Error()}})
			}
			return server.Send(response)
		}
	}
}
