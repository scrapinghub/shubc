package scrapinghub

import (
	"errors"
	"fmt"
	"io"
	"os"
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
		if err != nil {
			errch <- err
		}
	}()
	return errch
}

func emptyStringChan() <-chan string {
	outch := make(chan string)
	go func() {
		close(outch)
	}()
	return outch
}

// Validate an Scrapinghub job id
// Returns an error in case is wrong, nil otherwise
func ValidateJobID(job_id string) error {
	if !re_jobid.MatchString(job_id) {
		return wrong_job_id_error
	}
	return nil

}

// Validate an Scrapinghub project id
// Returns an error in case is wrong, nil otherwise
func ValidateProjectID(project_id string) error {
	if !re_projectid.MatchString(project_id) {
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

/*
 The following code was borrwed:
 - http://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file-in-golang
*/

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = copyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
