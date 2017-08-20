// Package freifunk implements a Service which gives access to freifunk map nodes.
package freifunk

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
		types.Command{
			Path: []string{"freifunk", "nodes"},
			Command: func(roomID, userID string, args []string) (interface{}, error) {
				return getNodes(args)
			},
		},
	}
}

func getNodes(args []string) (interface{}, error) {
	var nodes int
	var handler func([]byte, []byte, jsonparser.ValueType, int) error
	handler = func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		online, _ := jsonparser.GetBoolean(value, "flags", "online")
		if online {
			nodes++
		}
		return nil
	}

	ffApiJson, err := getApi("https://api.freifunk.net/data/ffSummarizedDir.json")
	if err != nil {
		return nil, err
	}

	arg := strings.Join(args, " ")
	community, _, _, communityErr := jsonparser.Get(ffApiJson, arg)
	if communityErr != nil {
		return nil, communityErr
	}

	var nodesObjectErr error
	_, communityArrayErr := jsonparser.ArrayEach(community, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		mapUrl, _ := jsonparser.GetString(value, "url")
		mapType, _ := jsonparser.GetString(value, "mapType")
		technicalType, _ := jsonparser.GetString(value, "technicalType")

		if technicalType == "meshviewer" {
			if mapType == "geographical" {
				var nodesJsonURL string
				if mapUrl[len(mapUrl)-1:] == "/" {
					nodesJsonURL = mapUrl + "data" + "nodes.json"
				} else {
					nodesJsonURL = mapUrl + "/" + "data" + "nodes.json"
				}

				nodesJson, _ := getApi(nodesJsonURL)
				nodesObject, _, _, _ := jsonparser.Get(nodesJson, "nodes")
				nodesObjectErr = jsonparser.ObjectEach(nodesObject, handler)
			}
		}
	}, "nodeMaps")

	if communityArrayErr != nil {
		return nil, communityArrayErr
	}

	if nodesObjectErr != nil {
		return nil, nodesObjectErr
	}

	return &gomatrix.TextMessage{"m.notice", strconv.Itoa(nodes)}, nil
}

func getCommunities() (interface{}, error) {
	var communities string
	var handler func([]byte, []byte, jsonparser.ValueType, int) error
	handler = func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		_, _, _, nodeMaps := jsonparser.Get(value, "nodeMaps")
		if nodeMaps == nil {
			keyString, err := jsonparser.ParseString(key)
			if err != nil {
				return err
			}
			if communities == "" {
				communities = keyString
			} else {
				communities = communities + ", " + keyString
			}
		}
		return nil
	}
	ffApiJson, err := getApi("https://api.freifunk.net/data/ffSummarizedDir.json")
	if err != nil {
		return nil, err
	}
	jsonparser.ObjectEach(ffApiJson, handler)
	return &gomatrix.TextMessage{"m.notice", communities}, nil
}

// getApi returns parsed Json
func getApi(urlAdress string) ([]byte, error) {
	log.Info("Fetching FF API")
	u, err := url.Parse(urlAdress)
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
