// Implements all the API methods related to Eggs handling
package scrapinghub

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type Egg struct {
	Name    string
	Version string
}
type Eggs struct {
	Status  string
	Message string
	EggData Egg   `json:"egg"`
	EggList []Egg `json:"eggs"`
}

func (eggs *Eggs) decodeContent(content []byte, err error) error {
	json.Unmarshal(content, eggs)
	if eggs.Status != "ok" {
		return err
	}
	return nil
}

// Add a python egg to the project `project_id` with `name` and `version` given.
func (eggs *Eggs) Add(conn *Connection, project_id, name, version, egg_path string) (*Egg, error) {
	params := url.Values{}
	params.Add("project", project_id)
	params.Add("name", name)
	params.Add("version", version)

	content, err := conn.APIPostFilesReadBody("/eggs/add.json", &params, map[string]string{"egg": egg_path})
	if err != nil {
		return nil, err
	}
	err = eggs.decodeContent(content, fmt.Errorf("Eggs.Add: Error ocurred while uploading egg: %s", eggs.Message))
	return &eggs.EggData, err
}

// Delete the egg `egg_name` from project `project_id`
func (eggs *Eggs) Delete(conn *Connection, project_id, egg_name string) error {
	params := url.Values{}
	params.Add("project", project_id)
	params.Add("name", egg_name)

	content, err := conn.APICallReadBody("/eggs/delete.json", POST, &params)
	if err != nil {
		return err
	}
	err = eggs.decodeContent(content, fmt.Errorf("Eggs.Delete: Error ocurred while deleting the egg: ", eggs.Message))
	return err
}

// List all the eggs in the project `project_id`
func (eggs *Eggs) List(conn *Connection, project_id string) ([]Egg, error) {
	params := url.Values{}
	params.Add("project", project_id)

	content, err := conn.APICallReadBody("/eggs/list.json", GET, &params)
	if err != nil {
		return nil, err
	}
	err = eggs.decodeContent(content, fmt.Errorf("Eggs.List: Error ocurred while listing the project <%s> eggs: %s", project_id, eggs.Message))
	return eggs.EggList, err
}
