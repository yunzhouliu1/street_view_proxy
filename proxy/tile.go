package proxy

import (
	"bytes"
	"fmt"
	"github.com/Daniel-W-Innes/street_view_proxy/config"
	"image"
	_ "image/jpeg"
	"io/ioutil"
	"log"
	"net/http"
)

func GetTile(x, y, zoom int, panoid string) (*image.Image, error) {
	ok := false
	var resp *http.Response
	var err error
	url := fmt.Sprintf("https://streetviewpixels-pa.googleapis.com/v1/tile?cb_client=maps_sv.tactile&panoid=%s&x=%d&y=%d&zoom=%d&nbt=1&fover=2", panoid, x, y, zoom)
	for i := 0; i < config.NumRetries; i++ {
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
