// Package freifunk implements a Service which gives access to freifunk map nodes.
package freifunk

import (
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/buger/jsonparser"
	"github.com/matrix-org/go-neb/types"
	"github.com/matrix-org/gomatrix"
	"github.com/prometheus/common/log"
)

// ServiceType of the Freifunk service
const ServiceType = "freifunk"

// Service represents the Echo service. It has no Config fields.
type Service struct {
	types.DefaultService
}

// Commands supported:
//    !freifunk communities
// Responds with a notice of a list with all communities.
func (s *Service) Commands(cli *gomatrix.Client) []types.Command {
	return []types.Command{
		types.Command{
			Path: []string{"freifunk", "communities"},
			Command: func(roomID, userID string, args []string) (interface{}, error) {
				return getCommunities()
			},
		},
	}
}

func getCommunities() (interface{}, error) {
	var communities string
	var handler func([]byte, []byte, jsonparser.ValueType, int) error
	handler = func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		keyString := jsonparser.ParseString(key)
		communities = communities + "\n" + keyString
	}
	jsonparser.ObjectEach(getFFApi(), handler)
	return &gomatrix.TextMessage{"m.notice", communities}, nil
}

// searchGiphy returns info about a gif
func (s *Service) getFFApi() ([]byte, error) {
	log.Info("Fetching FF API File for ")
	u, err := url.Parse("https://api.freifunk.net/data/ffSummarizedDir.json")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	u.RawQuery = q.Encode()
	res, err := http.Get(u.String())
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func init() {
	types.RegisterService(func(serviceID, serviceUserID, webhookEndpointURL string) types.Service {
		return &Service{
			DefaultService: types.NewDefaultService(serviceID, serviceUserID, ServiceType),
		}
	})
}
