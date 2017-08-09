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
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
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

type weblateProjectsResult struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []struct {
		Name           string `json:"name"`
		Slug           string `json:"slug"`
		Web            string `json:"web"`
		SourceLanguage struct {
			Code           string `json:"code"`
			Name           string `json:"name"`
			Nplurals       int    `json:"nplurals"`
			Pluralequation string `json:"pluralequation"`
			Direction      string `json:"direction"`
			WebURL         string `json:"web_url"`
			URL            string `json:"url"`
		} `json:"source_language"`
		WebURL            string `json:"web_url"`
		URL               string `json:"url"`
		ComponentsListURL string `json:"components_list_url"`
		RepositoryURL     string `json:"repository_url"`
		StatisticsURL     string `json:"statistics_url"`
		ChangesListURL    string `json:"changes_list_url"`
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
//    !weblate list languages [page]
// Responds with a notice containing all languages being worked on.
//
//    !weblate list projects [page]
// Responds with a notice containing the available Translation projects and their direct links.
//
//    !weblate maintain <language>
// Adds the User as a maintainer to the selected language.
//
//    !weblate unmaintain [language]
// Removes the User as a maintainer from the selected language or completely.
//
//    !weblate ping <language>
// Pings all maintainer of that list.
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
			Path: []string{"weblate", "list", "projects"},
			Command: func(roomID, userID string, args []string) (interface{}, error) {
				return s.cmdWeblateListProjects(roomID, userID, args)
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
			Path: []string{"weblate", "help"},
			Command: func(roomID, userID string, args []string) (interface{}, error) {
				return s.cmdWeblateHelp(roomID, userID, args)
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
	if len(args) == 1 {
		message := "Available Languages on page " + args[0] + ":\r\n"

		weblateRquest, err := s.makeWeblateRequest("GET", "languages/?page="+args[0], nil)
		if weblateRquest != nil {
			defer weblateRquest.Body.Close()
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to query Weblate: %s", err.Error())
		}

		var languages weblateLanguagesResult
		if err := json.NewDecoder(weblateRquest.Body).Decode(&languages); err != nil {
			return nil, fmt.Errorf("Failed to decode response (HTTP %d): %s", weblateRquest.StatusCode, err.Error())
		}

		for _, element := range languages.Results {
			message = message + element.Code + " - " + element.Name + "\r\n"
		}
		return gomatrix.TextMessage{"m.notice", message}, nil
	} else {
		endpoint := "languages"
		message := "Available Languages:\r\n"
		r := strings.NewReplacer(s.ServerURL+"api/", "")

		for len(endpoint) > 0 {
			weblateRquest, err := s.makeWeblateRequest("GET", endpoint, nil)
			if weblateRquest != nil {
				defer weblateRquest.Body.Close()
			}
			if err != nil {
				return nil, fmt.Errorf("Failed to query Weblate: %s", err.Error())
			}

			var languages weblateLanguagesResult
			if err := json.NewDecoder(weblateRquest.Body).Decode(&languages); err != nil {
				return nil, fmt.Errorf("Failed to decode response (HTTP %d): %s", weblateRquest.StatusCode, err.Error())
			}

			for _, element := range languages.Results {
				message = message + element.Code + " - " + element.Name + "\r\n"
			}
			endpoint = r.Replace(languages.Next)
		}
		return gomatrix.TextMessage{"m.notice", message}, nil
	}
	return nil, fmt.Errorf("You somehow exploited this command")
}

func (s *Service) cmdWeblateListProjects(roomID, userID string, args []string) (interface{}, error) {
	if len(args) == 1 {
		message := "Available Projects on page " + args[0] + ":\r\n"

		weblateRquest, err := s.makeWeblateRequest("GET", "projects/?page="+args[0], nil)
		if weblateRquest != nil {
			defer weblateRquest.Body.Close()
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to query Weblate: %s", err.Error())
		}

		var projects weblateProjectsResult
		if err := json.NewDecoder(weblateRquest.Body).Decode(&projects); err != nil {
			return nil, fmt.Errorf("Failed to decode response (HTTP %d): %s", weblateRquest.StatusCode, err.Error())
		}

		for _, element := range projects.Results {
			message = message + element.Name + " - " + element.WebURL + "\r\n"
		}

		return gomatrix.TextMessage{"m.notice", message}, nil
	} else {
		endpoint := "projects"
		message := "Available Projects:\r\n"
		r := strings.NewReplacer(s.ServerURL+"api/", "")

		for len(endpoint) > 0 {
			weblateRquest, err := s.makeWeblateRequest("GET", endpoint, nil)
			if weblateRquest != nil {
				defer weblateRquest.Body.Close()
			}
			if err != nil {
				return nil, fmt.Errorf("Failed to query Weblate: %s", err.Error())
			}

			var projects weblateProjectsResult
			if err := json.NewDecoder(weblateRquest.Body).Decode(&projects); err != nil {
				return nil, fmt.Errorf("Failed to decode response (HTTP %d): %s", weblateRquest.StatusCode, err.Error())
			}

			for _, element := range projects.Results {
				message = message + element.Name + " - " + element.WebURL + "\r\n"
			}
			endpoint = r.Replace(projects.Next)
		}

		return gomatrix.TextMessage{"m.notice", message}, nil
	}
}

func (s *Service) cmdWeblateHelp(roomID, userID string, args []string) (interface{}, error) {
	var message string
	if len(args) == 0 {
		message = "Available Commands:\r\n\r\n" +
			"- !weblate help [command] - Shows this help\r\n" +
			"- !weblate list languages - Lists available Languages\r\n" +
			"- !weblate list projects - Lists available Projects"
		return gomatrix.TextMessage{"m.notice", message}, nil
	} else if len(args) == 1 {
		if args[0] == "list" {
			message = "\"!weblate list\":\r\n\r\n" +
				"Shows a list of either all languages or projects\r\n\r\n" +
				"Subcommands:\r\n" +
				"- !weblate list languages - Lists available Languages\r\n" +
				"- !weblate list projects - Lists available Projects"
			return gomatrix.TextMessage{"m.notice", message}, nil
		} else {
			message = "Command not found"
			return nil, fmt.Errorf(message)
		}
	} else if len(args) == 2 {
		if args[0] == "list" {
			if args[1] == "languages" {
				message = "\"!weblate list languages\":\r\n\r\n" +
					"Lists available Languages"
				return gomatrix.TextMessage{"m.notice", message}, nil
			} else if args[1] == "projects" {
				message = "\"!weblate list languages\":\r\n\r\n" +
					"Lists available Projects"
				return gomatrix.TextMessage{"m.notice", message}, nil
			} else {
				message = "Command not found"
				return nil, fmt.Errorf(message)
			}
		}
	} else {
		message = "Command not found"
		return nil, fmt.Errorf(message)
	}
	return nil, fmt.Errorf("You somehow exploited this command")
}

func (s *Service) makeWeblateRequest(method string, endpoint string, body []byte) (*http.Response, error) {
	reader := bytes.NewReader(body)

	req, err := http.NewRequest(method, s.ServerURL+"api/"+endpoint, reader)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	if len(s.APIKey) > 0 {
		req.Header.Add("Autorization", "Token "+s.APIKey)
	}

	res, err := httpClient.Do(req)
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
