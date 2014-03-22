package scrapinghub

import (
	"encoding/json"
	"errors"
	"net/url"
)

// Simple struct to holds the values of the spiders of the
// project
type Spiders struct {
	Spiders []map[string]string
	Status  string
}

// errors
var spider_list_error = errors.New("Spiders.List: Error while retrieving the spider list")

// Retrieve all the spiders of the project given a connection `conn` and the `project_id`.
// Returns the Spiders object itself and an error (nil in case no error ocurred).
func (spider *Spiders) List(conn *Connection, project_id string) (*Spiders, error) {
	params := url.Values{}
	params.Add("project", project_id)

	content, err := conn.APICallReadBody("/spiders/list.json", GET, &params)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(content, spider)

	if spider.Status != "ok" {
		return nil, spider_list_error
	}
	return spider, nil
}
