// Go bindings for Scrapinghub API (http://doc.scrapinghub.com/api.html)
package scrapinghub

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// Scrapinghub Base API URL
var baseUrl = "https://dash.scrapinghub.com/api"
var re_jobid = regexp.MustCompile(`(?P<project_id>\d+)/\d+/\d+`)

type Connection struct {
	client *http.Client
	apikey string
}

// Create a new connection to Scrapinghub API
func (conn *Connection) New(apikey string) {
	// Create TLS config
	tlsConfig := tls.Config{RootCAs: nil}

	tr := &http.Transport{
		TLSClientConfig:    &tlsConfig,
		DisableCompression: true,
	}

	conn.apikey = apikey
	conn.client = &http.Client{Transport: tr}
}

// Do a HTTP request, using method `method` and return the response body in content
// Argument `params` is a map with the POST parameters, in case method = POST.
func (conn *Connection) do_request_content(rurl string, method string, params map[string]string) ([]byte, error) {
	var req *http.Request
	var err error

	if method == "GET" {
		req, err = http.NewRequest("GET", rurl, nil)
	} else if method == "POST" {
		data := url.Values{}
		for k, v := range params {
			data.Add(k, v)
		}
		req, err = http.NewRequest("POST", rurl, bytes.NewBufferString(data.Encode()))
	}

	if err != nil {
		return nil, err
	}
	// Set Scrapinghub api key to request
	req.SetBasicAuth(conn.apikey, "")
	resp, err := conn.client.Do(req)
	if err != nil {
		return nil, err
	}

	// Create buffer
	content := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if n == 0 {
			break
		}
		content = append(content, buf[:n]...)
	}
	return content, nil

}

// Do a HTTP request, using method `method` and returns a reponse type (http.Reponse)
// Argument `params` is a map with the POST parameters, in case method = POST.
func (conn *Connection) do_request(rurl string, method string, params map[string]string) (*http.Response, error) {
	var req *http.Request
	var err error

	if method == "GET" {
		req, err = http.NewRequest("GET", rurl, nil)
	} else if method == "POST" {
		data := url.Values{}
		for k, v := range params {
			data.Add(k, v)
		}
		req, err = http.NewRequest("POST", rurl, bytes.NewBufferString(data.Encode()))
	}

	if err != nil {
		return nil, err
	}
	// Set Scrapinghub api key to request
	req.SetBasicAuth(conn.apikey, "")
	return conn.client.Do(req)
}

type Spiders struct {
	Spiders []map[string]string
	Status  string
}

// errors
var spider_list_error = errors.New("Spiders.List: Error while retrieving the spider list")
var jobs_list_error = errors.New("Jobs.List: Error while retrieving the jobs list")
var wrong_job_id_error = errors.New("Job ID is empty or not in the right format (e.g: 123/1/2)")

func (spider *Spiders) List(conn *Connection, project_id string) (*Spiders, error) {
	method := "/spiders/list.json?project=" + project_id
	content, err := conn.do_request_content(baseUrl+method, "GET", nil)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(content, spider)

	if spider.Status != "ok" {
		return nil, spider_list_error
	}
	return spider, nil
}

type Jobs struct {
	Status  string
	Count   int
	Total   int
	Jobs    []map[string]interface{}
	JobId   string
	Message string
}

// Returns a map given a list of ["key=value", ...] strings
func equality_list_to_map(data []string) map[string]string {
	result := make(map[string]string)
	for _, e := range data {
		if strings.Index(e, "=") > 0 {
			res := strings.Split(e, "=")
			result[strings.TrimSpace(res[0])] = strings.TrimSpace(res[1])
		}
	}
	return result
}

// Returns the list of Jobs for project_id limited by count and those which
// match the filters
func (jobs *Jobs) List(conn *Connection, project_id string, count int, filters []string) (*Jobs, error) {
	method := fmt.Sprintf("/jobs/list.json?project=%s&count=%d", project_id, count)
	mfilters := equality_list_to_map(filters)
	for fname, fval := range mfilters {
		method = fmt.Sprintf("%s&%s=%s", method, fname, fval)
	}
	content, err := conn.do_request_content(baseUrl+method, "GET", nil)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(content, jobs)

	if jobs.Status != "ok" {
		return nil, jobs_list_error
	}
	return jobs, nil
}

// Returns the job information in map object given the job_id
func (jobs *Jobs) JobInfo(conn *Connection, job_id string) (map[string]string, error) {
	result := re_jobid.FindStringSubmatch(job_id)
	if len(result) == 0 {
		return nil, wrong_job_id_error
	}
	project_id := result[1]

	method := fmt.Sprintf("/jobs/list.json?project=%s&job=%s", project_id, job_id)
	content, err := conn.do_request_content(baseUrl+method, "GET", nil)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(content, jobs)

	if jobs.Status != "ok" {
		return nil, errors.New("Jobs.JobInfo: Error while retrieving job info")
	}
	if len(jobs.Jobs) <= 0 {
		return nil, errors.New(fmt.Sprintf("Jobs.JobInfo: Job %s does not exist", job_id))
	}

	m := make(map[string]string)
	for k, v := range jobs.Jobs[0] {
		m[k] = fmt.Sprintf("%v", v)
	}
	return m, nil
}

// Schedule the spider with name `spider_name` and arguments `args` on `project_id`.
func (jobs *Jobs) Schedule(conn *Connection, project_id string, spider_name string, args []string) (string, error) {
	method := "/schedule.json"
	data := map[string]string{
		"project": project_id,
		"spider":  spider_name,
	}
	params := equality_list_to_map(args)
	for k, v := range params {
		data[k] = v
	}
	content, err := conn.do_request_content(baseUrl+method, "POST", data)
	if err != nil {
		return "", err
	}
	json.Unmarshal(content, jobs)

	if jobs.Status != "ok" {
		return "", errors.New(fmt.Sprintf("Jobs.Schedule: Error while scheduling the job. Message: %s", jobs.Message))
	}
	return jobs.JobId, nil
}

// Stop the job with `job_id`.
func (jobs *Jobs) Stop(conn *Connection, job_id string) error {
	result := re_jobid.FindStringSubmatch(job_id)
	if len(result) == 0 {
		return wrong_job_id_error
	}
	project_id := result[1]

	method := "/jobs/stop.json"

	data := map[string]string{
		"project": project_id,
		"job":     job_id,
	}
	content, err := conn.do_request_content(baseUrl+method, "POST", data)
	if err != nil {
		return err
	}
	json.Unmarshal(content, jobs)

	if jobs.Status != "ok" {
		return errors.New(fmt.Sprintf("Jobs.Stop: Error while stopping the job. Message: %s", jobs.Message))
	}
	return nil
}

// Returns up to `count` items for the job `job_id`, starting at `offset`. Each
// item is returned as a map with string key but value of type `interface{}`
func RetrieveItems(conn *Connection, job_id string, count, offset int) ([]map[string]interface{}, error) {
	result := re_jobid.FindStringSubmatch(job_id)
	if len(result) == 0 {
		return nil, wrong_job_id_error
	}
	project_id := result[1]

	method := fmt.Sprintf("/items.json?project=%s&job=%s&count=%d&offset=%d", project_id, job_id, count, offset)

	content, err := conn.do_request_content(baseUrl+method, "GET", nil)
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

// Download the slybot project for the project `project_id` and the spiders given.
// The method write the zip file to `out` argument.
func RetrieveSlybotProject(conn *Connection, project_id string, spiders []string, out *os.File) error {
	method := fmt.Sprintf("/as/project-slybot.zip?project=%s", project_id)
	for _, spider := range spiders {
		method = method + fmt.Sprintf("&spider=%s", spider)
	}

	resp, err := conn.do_request(baseUrl+method, "GET", nil)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	// Create buffer
	buf := make([]byte, 1024)

	for {
		n, err := resp.Body.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		if _, err := out.Write(buf[:n]); err != nil {
			return err
		}
	}
	return nil
}

func retrieveLinesStream(conn *Connection, method string) (<-chan string, error) {
	resp, err := conn.do_request(baseUrl+method, "GET", nil)
	if err != nil {
		return nil, err
	}
	ch := make(chan string)

	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			ch <- scanner.Text()
		}
		close(ch)
	}()
	return ch, nil
}

//  Given a job_id, returns a channel of strings where each element is a line of
//  the JsonLines returned by the API items.jl endpoint.
func RetrieveItemsJsonLines(conn *Connection, job_id string) (<-chan string, error) {
	result := re_jobid.FindStringSubmatch(job_id)
	if len(result) == 0 {
		return nil, wrong_job_id_error
	}
	project_id := result[1]
	method := fmt.Sprintf("/items.jl?project=%s&job=%s", project_id, job_id)

	return retrieveLinesStream(conn, method)
}

// Returns a channel of strings which each element is a JSON serialized job for
// the project `project_id`. `count` and filters (a list of string of the type
// key=value to apply to the result (see http://doc.scrapinghub.com/api.html#jobs-list-json)
func RetrieveJobsJsonLines(conn *Connection, project_id string, count int, filters []string) (<-chan string, error) {
	method := fmt.Sprintf("/jobs/list.jl?project=%s&count=%d", project_id, count)
	mfilters := equality_list_to_map(filters)
	for fname, fval := range mfilters {
		method = fmt.Sprintf("%s&%s=%s", method, fname, fval)
	}

	return retrieveLinesStream(conn, method)
}

// Returns a channel of strings which each element is a line of the log for job with `job_id`
// Count and offset parameters are accepted to paginate results.
func LogLines(conn *Connection, job_id string, count, offset int) (<-chan string, error) {
	result := re_jobid.FindStringSubmatch(job_id)
	if len(result) == 0 {
		return nil, wrong_job_id_error
	}
	project_id := result[1]
	method := fmt.Sprintf("/log.txt?project=%s&job=%s&count=%d&offset=%d", project_id, job_id, count, offset)

	return retrieveLinesStream(conn, method)
}
