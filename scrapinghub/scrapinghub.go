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

// Do a HTTP request, using method `method` and returns a reponse type (http.Reponse)
// Argument `params` is a map with the POST parameters, in case method = POST.
func (conn *Connection) do_request(rurl string, method string, params map[string][]string) (resp *http.Response, err error) {
	var req *http.Request

	if method == "GET" {
		req, err = http.NewRequest("GET", rurl, nil)
	} else if method == "POST" {
		data := url.Values{}
		for k, vals := range params {
			for _, v := range vals {
				data.Add(k, v)
			}
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

// Do a HTTP request, using method `method` and return the response body in content
// Argument `params` is a map with the POST parameters, in case method = POST.
func (conn *Connection) do_request_content(rurl string, method string, params map[string][]string) ([]byte, error) {
	resp, err := conn.do_request(rurl, method, params)
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

type Spiders struct {
	Spiders []map[string]string
	Status  string
}

// errors
var (
	spider_list_error  = errors.New("Spiders.List: Error while retrieving the spider list")
	jobs_list_error    = errors.New("Jobs.List: Error while retrieving the jobs list")
	wrong_job_id_error = errors.New("Job ID is empty or not in the right format (e.g: 123/1/2)")
)

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

type Job struct {
	CloseReason       string `json:"close_reason"`
	Elapsed           int
	ErrorsCount       int `json:"errors_count"`
	Id                string
	ItemsScraped      int    `json:"items_scraped"`
	SpiderType        string `json:"spider_type"`
	ResponsesReceived int    `json:"responses_received"`
	Logs              int
	Priority          int
	Spider            string
	SpiderArgs        map[string]string `json:"spider_args"`
	StartedTime       string            `json:"started_time"`
	State             string
	Tags              []string
	UpdatedTime       string `json:"updated_time"`
	Version           string
}

type Jobs struct {
	Status  string
	Count   int
	Total   int
	Jobs    []Job
	JobId   string
	Message string
}

// Returns the list of Jobs for project_id limited by count and those which
// match the filters
func (jobs *Jobs) List(conn *Connection, project_id string, count int, filters map[string]string) (*Jobs, error) {
	method := fmt.Sprintf("/jobs/list.json?project=%s&count=%d", project_id, count)
	for fname, fval := range filters {
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
func (jobs *Jobs) JobInfo(conn *Connection, job_id string) (*Job, error) {
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

	return &jobs.Jobs[0], nil
}

func project_data_map(project_id string, spider_name string, job_id string, args map[string]string) map[string][]string {
	data := map[string][]string{
		"project": []string{project_id},
	}
	if spider_name != "" {
		data["spider"] = []string{spider_name}
	}
	if job_id != "" {
		data["job"] = []string{job_id}
	}
	for k, v := range args {
		if k != "project" && k != "spider" {
			data[k] = []string{v}
		}
	}
	return data
}

// Schedule the spider with name `spider_name` and arguments `args` on `project_id`.
func (jobs *Jobs) Schedule(conn *Connection, project_id string, spider_name string, args map[string]string) (string, error) {
	method := "/schedule.json"
	data := project_data_map(project_id, spider_name, "", args)
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

// Re-schedule the spider with `job_id` using the same tags and parameters
func (jobs *Jobs) Reschedule(conn *Connection, job_id string) (string, error) {
	result := re_jobid.FindStringSubmatch(job_id)
	if len(result) == 0 {
		return "", wrong_job_id_error
	}
	project_id := result[1]

	job, err := jobs.JobInfo(conn, job_id)
	if err != nil {
		return "", err
	}
	method := "/schedule.json"
	data := project_data_map(project_id, job.Spider, "", job.SpiderArgs)
	data["add_tag"] = job.Tags
	content, err := conn.do_request_content(baseUrl+method, "POST", data)
	if err != nil {
		return "", err
	}
	json.Unmarshal(content, jobs)

	if jobs.Status != "ok" {
		return "", errors.New(fmt.Sprintf("Jobs.Reschedule: Error while scheduling the job. Message: %s", jobs.Message))
	}
	return jobs.JobId, nil
}

func (jobs *Jobs) postAction(conn *Connection, job_id string, method string, error_string string, update_data map[string]string) error {
	result := re_jobid.FindStringSubmatch(job_id)
	if len(result) == 0 {
		return wrong_job_id_error
	}
	project_id := result[1]

	data := project_data_map(project_id, "", job_id, update_data)
	content, err := conn.do_request_content(baseUrl+method, "POST", data)
	if err != nil {
		return err
	}
	json.Unmarshal(content, jobs)
	if jobs.Status != "ok" {
		return errors.New(fmt.Sprintf("%s. Message: %s", error_string, jobs.Message))
	}
	return nil
}

// Stop the job with `job_id`.
func (jobs *Jobs) Stop(conn *Connection, job_id string) error {
	return jobs.postAction(conn, job_id, "/jobs/stop.json",
		"Jobs.Stop: Error while stopping the job", nil)
}

// Update the job with `job_id` with the `update_data`.
func (jobs *Jobs) Update(conn *Connection, job_id string, update_data map[string]string) error {
	return jobs.postAction(conn, job_id, "/jobs/update.json",
		"Jobs.Update: Error while updating the job", update_data)
}

// Delete the job with `job_id`.
func (jobs *Jobs) Delete(conn *Connection, job_id string) error {
	return jobs.postAction(conn, job_id, "/jobs/delete.json",
		"Jobs.Delete: Error while deleting the job", nil)
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
func ItemsAsJsonLines(conn *Connection, job_id string) (<-chan string, error) {
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
func JobsAsJsonLines(conn *Connection, project_id string, count int, filters map[string]string) (<-chan string, error) {
	method := fmt.Sprintf("/jobs/list.jl?project=%s&count=%d", project_id, count)
	for fname, fval := range filters {
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
