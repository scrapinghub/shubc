package scrapinghub

import (
	"encoding/json"
	"net/url"
	"strconv"
)

// Returns up to `count` items for the job `job_id`, starting at `offset`. Each
// item is returned as a map with string key but value of type `interface{}`
func RetrieveItems(conn *Connection, job_id string, count, offset int) ([]map[string]interface{}, error) {
	if err := ValidateJobID(job_id); err != nil {
		return nil, err
	}
	project_id := ProjectID(job_id)

	params := url.Values{}
	params.Add("project", project_id)
	params.Add("job", job_id)
	params.Add("offset", strconv.Itoa(offset))
	if count > 0 {
		params.Add("count", strconv.Itoa(count))
	}

	content, err := conn.APICallReadBody("/items.json", GET, &params)
	if err != nil {
		return nil, err
	}

	var f interface{}
	err = json.Unmarshal(content, &f)
	if err != nil {
		return nil, err
	}
	jarray := f.([]interface{})

	items := make([]map[string]interface{}, len(jarray))
	for i, e := range jarray {
		items[i] = e.(map[string]interface{})
	}
	return items, nil
}
