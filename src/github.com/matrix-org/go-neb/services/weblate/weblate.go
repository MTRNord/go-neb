// Package weblate implements a Service which lets you access statistics of Weblate and allows to easy manage Translators.
package weblate

import (
	"net/http"
	"strings"

	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/matrix-org/go-neb/types"
	"github.com/matrix-org/gomatrix"
	"io/ioutil"
)

// ServiceType of the Weblate service
const ServiceType = "weblate"

var httpClient = &http.Client{}

type weblateLanguagesResult struct {
	Count    int         `json:"count"`
	Next     string      `json:"next"`
	Previous interface{} `json:"previous"`
	Results  []struct {
		Code           string `json:"code"`
		Name           string `json:"name"`
		Nplurals       int    `json:"nplurals"`
		Pluralequation string `json:"pluralequation"`
		Direction      string `json:"direction"`
		WebURL         string `json:"web_url"`
		URL            string `json:"url"`
	} `json:"results"`
}

// Service represents the Echo service. It has no Config fields.
type Service struct {
	types.DefaultService
	// The Weblate API key to use when making HTTP requests to Weblate.
	APIKey string `json:"api_key"`
	// The Weblate Server url to use when making HTTP requests.
	ServerURL string `json:"server_url"`
}

// Commands supported:
//    !weblate status [language]
// Responds with a notice containing the Translation status for either the hole project or a selected language.
//
//    !weblate list languages
// Responds with a notice containing all languages being worked on.
//
//    !weblate maintain <language>
// Adds the User as a maintainer to the selected language.
//
//    !weblate unmaintain [language]
// Removes the User as a maintainer from the selected language or completely.
//
//    !weblate ping <language>
// Pings all maintainer of that list.
//
//    !weblate list projects
// Responds with a notice containing the available Translation projects and their direct links.
func (s *Service) Commands(cli *gomatrix.Client) []types.Command {
	return []types.Command{
		types.Command{
			Path: []string{"weblate", "status"},
			Command: func(roomID, userID string, args []string) (interface{}, error) {
				return s.cmdWeblateStatus(roomID, userID, args)
			},
		},
		types.Command{
			Path: []string{"weblate", "list", "languages"},
			Command: func(roomID, userID string, args []string) (interface{}, error) {
				return s.cmdWeblateListLanguages(roomID, userID, args)
			},
		},
		types.Command{
			Path: []string{"weblate", "maintain"},
			Command: func(roomID, userID string, args []string) (interface{}, error) {
				return &gomatrix.TextMessage{"m.notice", strings.Join(args, " ")}, nil
			},
		},
		types.Command{
			Path: []string{"weblate", "unmaintain"},
			Command: func(roomID, userID string, args []string) (interface{}, error) {
				return &gomatrix.TextMessage{"m.notice", strings.Join(args, " ")}, nil
			},
		},
		types.Command{
			Path: []string{"weblate", "ping"},
			Command: func(roomID, userID string, args []string) (interface{}, error) {
				return &gomatrix.TextMessage{"m.notice", strings.Join(args, " ")}, nil
			},
		},
		types.Command{
			Path: []string{"weblate", "list", "projects"},
			Command: func(roomID, userID string, args []string) (interface{}, error) {
				return &gomatrix.TextMessage{"m.notice", strings.Join(args, " ")}, nil
			},
		},
	}
}

func (s *Service) cmdWeblateStatus(roomID, userID string, args []string) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("Too many arguments")
	}

	return gomatrix.TextMessage{"m.notice", "Not yet implemented"}, nil
}

func (s *Service) cmdWeblateListLanguages(roomID, userID string, args []string) (interface{}, error) {
	if len(args) > 0 {
		return nil, fmt.Errorf("Too many arguments")
	}

	weblateRquest, err := s.makeWeblateRequest("GET", "languages", nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to query Weblate: %s", err.Error())
	}

	var languages weblateLanguagesResult
	if err := json.NewDecoder(weblateRquest.Body).Decode(&languages); err != nil {
		return nil, fmt.Errorf("Failed to decode response (HTTP %d): %s", weblateRquest.StatusCode, err.Error())
	}

	var message string

	for _, element := range languages.Results {
		message = message + element.Code + " - " + element.Name + "\r\n"
	}

	return gomatrix.TextMessage{"m.notice", message}, nil
}

func (s *Service) makeWeblateRequest(method string, endpoint string, body []byte) (*http.Response, error) {
	reader := bytes.NewReader(body)

	req, err := http.NewRequest(method, s.ServerURL+"api/"+endpoint, reader)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Autorization", "Token "+s.APIKey)

	res, err := httpClient.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		resBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.WithError(err).Error("Failed to decode Weblate response body")
		}
		log.WithFields(log.Fields{
			"code": res.StatusCode,
			"body": string(resBytes),
		}).Error("Failed to query Weblate")
		return nil, fmt.Errorf("Failed to decode response (HTTP %d)", res.StatusCode)
	}

	return res, nil
}

func init() {
	types.RegisterService(func(serviceID, serviceUserID, webhookEndpointURL string) types.Service {
		return &Service{
			DefaultService: types.NewDefaultService(serviceID, serviceUserID, ServiceType),
		}
	})
}
