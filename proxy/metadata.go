package proxy

import (
	"encoding/json"
	"fmt"
	"github.com/Daniel-W-Innes/street_view_proxy/view"
	"io/ioutil"
	"log"
	"net/http"
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

func GetMetadata(location *view.Location, key string) (*Metadata, error) {
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
