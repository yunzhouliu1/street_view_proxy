package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Daniel-W-Innes/street_view_proxy/view"
	"google.golang.org/grpc"
	"image"
	"image/draw"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
)

const (
	port             = 8080
	minX             = 13
	minY             = 6
	maxX             = 19
	maxY             = 10
	zoom             = 5
	inputSizeX       = 512
	inputSizeY       = inputSizeX
	outputSizeX      = inputSizeX * (maxX - minX)
	outputSizeY      = inputSizeY * (maxY - minY)
	numRetries       = 10
	workerMultiplier = 10
	saveImages       = true
)

type Location struct {
	Latitude  float64 `json:"lat,omitempty"`
	Longitude float64 `json:"lng,omitempty"`
}

type Metadata struct {
	Copyright string   `json:"copyright,omitempty"`
	Date      string   `json:"date,omitempty"`
	Location  Location `json:"location"`
	PanoId    string   `json:"pano_id,omitempty"`
	Status    string   `json:"status,omitempty"`
}

type TileRequest struct {
	x, y, zoom int
	panoId     string
}

type Tile struct {
	image       image.Image
	tileRequest TileRequest
	error       error
}

func getMetadata(location *view.Location, key string) (*Metadata, error) {
	response, err := http.Get(fmt.Sprintf("https://maps.googleapis.com/maps/api/streetview/metadata?location=%f,%f&key=%s", location.Latitude, location.Longitude, key))
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error bad status from googleapis %d", response.StatusCode)
	}
	defer func() {
		err := response.Body.Close()
		if err != nil {
			log.Fatalf(err.Error())
		}
	}()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	metadata := Metadata{}
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}

func getTile(x, y, zoom int, panoid string) (*image.Image, error) {
	ok := false
	var resp *http.Response
	var err error
	url := fmt.Sprintf("https://streetviewpixels-pa.googleapis.com/v1/tile?cb_client=maps_sv.tactile&panoid=%s&x=%d&y=%d&zoom=%d&nbt=1&fover=2", panoid, x, y, zoom)
	for i := 0; i < numRetries; i++ {
		resp, err = http.Get(url)
		if err != nil {
			log.Fatalln(err)
		}
		if resp.StatusCode == http.StatusOK {
			ok = true
			break
		}
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	if !ok {
		return nil, fmt.Errorf("bad status url: \"%s\" code: %d ,data: %s", url, resp.StatusCode, data)
	}
	decode, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Fatalln(err)
	}
	return &decode, nil
}

type TileWorker struct {
	input  chan *TileRequest
	exit   chan struct{}
	output chan *Tile
}

func (t *TileWorker) getTileWorker() {
	for {
		select {
		case tileRequest := <-t.input:
			tile, err := getTile(tileRequest.x, tileRequest.y, tileRequest.zoom, tileRequest.panoId)
			if err != nil {
				t.output <- &Tile{error: err}
			} else {
				t.output <- &Tile{image: *tile, tileRequest: *tileRequest}
			}
		case <-t.exit:
			return
		}
	}
}

func (t *TileWorker) downloadMosaic(metadata *Metadata) {
	for x := minX; x < maxX; x++ {
		for y := minY; y < maxY; y++ {
			t.input <- &TileRequest{x: x, y: y, zoom: zoom, panoId: metadata.PanoId}
		}
	}
}

func (t *TileWorker) getMosaic() (image.Image, error) {
	rgba := image.NewRGBA(image.Rect(0, 0, outputSizeX, outputSizeY))
	for x := minX; x < maxX; x++ {
		for y := minY; y < maxY; y++ {
			output := <-t.output
			if output.error != nil {
				log.Printf("err from X: %d, y: %d\n", x, y)
			} else {
				draw.Draw(rgba, image.Rect(
					(output.tileRequest.x-minX)*inputSizeX, (output.tileRequest.y-minY)*inputSizeY,
					(output.tileRequest.x+1-minX)*inputSizeX, (output.tileRequest.y+1-minY)*inputSizeY),
					output.image, image.Point{}, draw.Src)
			}
		}
	}
	return rgba, nil
}

func saveImage(img image.Image, path string) error {
	out, err := os.Create(path)
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

func (s *ImageDownloaderServer) GetImage(server view.ImageDownloader_GetImageServer) error {
	log.Println("opened request channel")
	tileWorker := TileWorker{
		input:  make(chan *TileRequest, (maxY-minY)*(maxX-minX)),
		exit:   make(chan struct{}),
		output: make(chan *Tile, (maxY-minY)*(maxX-minX)),
	}
	for i := 0; i < runtime.NumCPU()*workerMultiplier; i++ {
		go tileWorker.getTileWorker()
	}
	log.Println("created workers")
	for {
		log.Println("ready for request")
		in, err := server.Recv()
		log.Println("got request")
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		location := in.Location
		if location != nil {
			log.Println("getting metadata")
			metadata, err := getMetadata(location, s.ApiKey)
			log.Printf("got metadata: %v\n", metadata)
			if err != nil {
				return server.Send(&view.Response{Error: &view.Error{Description: err.Error()}})
			}
			if metadata.Status != "OK" {
				return server.Send(&view.Response{Error: &view.Error{Description: metadata.Status}})
			}
			log.Println("downloading mosaic")
			go tileWorker.downloadMosaic(metadata)
			tile, err := tileWorker.getMosaic()
			log.Println("downloaded mosaic")
			if err != nil {
				return server.Send(&view.Response{Error: &view.Error{Description: err.Error()}})
			}
			log.Println("encoding response")
			buf := new(bytes.Buffer)
			err = jpeg.Encode(buf, tile, nil)
			if err != nil {
				return server.Send(&view.Response{Error: &view.Error{Description: err.Error()}})
			}
			if saveImages {
				go func() {
					err := saveImage(tile, fmt.Sprintf("%f,%f_x:%d-%d_y:%d-%d_%d.png", metadata.Location.Latitude, metadata.Location.Longitude, minX, maxX, minY, maxY, zoom))
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
			err = server.Send(&view.Response{Image: &outImage})
			if err != nil {
				return err
			}
		}
	}
}

func main() {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		log.Fatalf("failed to listen on %d: %v", port, err)
	}
	var opts []grpc.ServerOption

	server := grpc.NewServer(opts...)
	view.RegisterImageDownloaderServer(server, &ImageDownloaderServer{ApiKey: os.Getenv("API_KEY")})
	fmt.Printf("listen on port %d\n", port)
	err = server.Serve(lis)
	if err != nil {
		log.Fatalf("failed to serve server: %v", err)
	}
}
