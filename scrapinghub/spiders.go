// Implment all the methods related to Spider information
package scrapinghub

import (
	"encoding/json"
	"errors"
	"net/url"
)

type Spiders struct {
	Spiders []map[string]string
	Status  string
}

// errors
var spider_list_error = errors.New("Spiders.List: Error while retrieving the spider list")

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
