package scrapinghub

import (
	"errors"
	"regexp"
)

var re_jobid = regexp.MustCompile(`(?P<project_id>\d+)/\d+/\d+`)
var re_projectid = regexp.MustCompile(`\d+`)
var (
	wrong_job_id_error     = errors.New("Job ID is empty or not in the right format (e.g: 123/1/2)")
	wrong_project_id_error = errors.New("Project ID is empty or not in the right format (e.g: NNNN where N is a digit)")
)

// Given an error return a channel with the error on it
func fromErrToErrChan(err error) <-chan error {
	errch := make(chan error)
	go func() {
		defer close(errch)
		errch <- err
	}()
	return errch
}

// Validate an Scrapinghub job id
// Returns an error in case is wrong, nil otherwise
func ValidateJobID(job_id string) error {
	if re_jobid.MatchString(job_id) {
		return wrong_job_id_error
	}
	return nil

}

// Validate an Scrapinghub project id
// Returns an error in case is wrong, nil otherwise
func ValidateProjectID(project_id string) error {
	if re_projectid.MatchString(project_id) {
		return wrong_project_id_error
	}
	return nil
}

// Extract the project_id from a job_id
// Precondition: the function assume job_id is a valid Scrapinghub job id
func ProjectID(job_id string) string {
	result := re_jobid.FindStringSubmatch(job_id)
	return result[1]
}
