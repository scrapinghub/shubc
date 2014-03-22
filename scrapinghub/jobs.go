package scrapinghub

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

// Represent a Scrapinghub Job with all the fields returned
// by the API
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

// Jobs is a collection of jobs, in some cases it may contain
// just a single JobId (when scheduling for example)
type Jobs struct {
	Status  string
	Count   int
	Total   int
	Jobs    []Job
	JobId   string
	Message string
}

var jobs_list_error = errors.New("Jobs.List: Error while retrieving the jobs list")

func (jobs *Jobs) decodeContent(content []byte, err error) error {
	json.Unmarshal(content, jobs)
	if jobs.Status != "ok" {
		return err
	}
	return nil
}

// Returns the list of Jobs for project_id limited by count and those which
// match the filters
func (jobs *Jobs) List(conn *Connection, project_id string, count int, filters map[string]string) (*Jobs, error) {
	params := url.Values{}
	params.Add("project", project_id)
	if count > 0 {
		params.Add("count", strconv.Itoa(count))
	}
	for fname, fval := range filters {
		params.Add(fname, fval)
	}

	content, err := conn.APICallReadBody("/jobs/list.json", GET, &params)
	if err != nil {
		return nil, err
	}
	err = jobs.decodeContent(content, jobs_list_error)
	return jobs, err
}

// Returns the job information in map object given the job_id
func (jobs *Jobs) JobInfo(conn *Connection, job_id string) (*Job, error) {
	if err := ValidateJobID(job_id); err != nil {
		return nil, err
	}
	project_id := ProjectID(job_id)

	params := url.Values{}
	params.Add("project", project_id)
	params.Add("job_id", job_id)

	content, err := conn.APICallReadBody("/jobs/list.json", GET, &params)
	if err != nil {
		return nil, err
	}
	err = jobs.decodeContent(content, fmt.Errorf("Jobs.JobInfo: Error while retrieving job info"))
	if err == nil && len(jobs.Jobs) <= 0 {
		return nil, fmt.Errorf("Jobs.JobInfo: Job %s does not exist", job_id)
	}
	return &jobs.Jobs[0], err
}

// Schedule the spider with name `spider_name` and arguments `args` on `project_id`.
func (jobs *Jobs) Schedule(conn *Connection, project_id string, spider_name string, args map[string]string) (string, error) {
	if err := ValidateProjectID(project_id); err != nil {
		return "", err
	}

	params := url.Values{}
	params.Add("project", project_id)
	params.Add("spider", spider_name)
	for k, v := range args {
		params.Set(k, v)
	}

	content, err := conn.APICallReadBody("/schedule.json", POST, &params)
	if err != nil {
		return "", err
	}
	err = jobs.decodeContent(content, fmt.Errorf("Jobs.Schedule: Error while scheduling the job. Message: %s", jobs.Message))
	return jobs.JobId, err
}

// Re-schedule the spider with `job_id` using the same tags and parameters
func (jobs *Jobs) Reschedule(conn *Connection, job_id string) (string, error) {
	if err := ValidateJobID(job_id); err != nil {
		return "", err
	}
	project_id := ProjectID(job_id)

	job, err := jobs.JobInfo(conn, job_id)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Add("project", project_id)
	params.Add("spider", job.Spider)
	for k, v := range job.SpiderArgs {
		params.Set(k, v)
	}
	for _, tag := range job.Tags {
		params.Add("add_tag", tag)
	}

	content, err := conn.APICallReadBody("/schedule.json", POST, &params)
	if err != nil {
		return "", err
	}
	err = jobs.decodeContent(content, fmt.Errorf("Jobs.Reschedule: Error while scheduling the job. Message: %s", jobs.Message))
	return jobs.JobId, err
}

func (jobs *Jobs) postAction(conn *Connection, job_id string, method string, error_string string, update_data map[string]string) error {
	if err := ValidateJobID(job_id); err != nil {
		return err
	}
	project_id := ProjectID(job_id)

	params := url.Values{}
	params.Add("project", project_id)
	params.Add("job", job_id)
	for k, v := range update_data {
		params.Set(k, v)
	}

	content, err := conn.APICallReadBody(method, POST, &params)
	if err != nil {
		return err
	}
	return jobs.decodeContent(content, fmt.Errorf("%s. Message: %s", error_string, jobs.Message))
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
