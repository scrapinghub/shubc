/*
 * Deploy module
 *
 */

package scrapinghub

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"encoding/json"

	"github.com/vaughan0/go-ini"
)

const SCRAPY_ENVAR = "SCRAPY_SETTINGS_MODULE"
const PY_CHECK_IMP = "import os, importlib; importlib.import_module(os.environ.get('%s'))"
const SCRAPINGHUB_DEPLOY_URL = "https://dash.scrapinghub.com/api/scrapyd/"

const SETUP_PY_TEMPLATE = `# Automatically created by: shub deploy

from setuptools import setup, find_packages

setup(
    name         = 'project',
    version      = '1.0',
    packages     = find_packages(),
    entry_points = {'scrapy': ['settings = %s']},
)
`

// Return the path to the closest scrapy.cfg file by traversing the current
// directory and its parents
func closest_scrapy_cfg(path, prevpath string) string {
	if path == prevpath {
		return ""
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	cfgfile := filepath.Join(path, "scrapy.cfg")
	if _, err := os.Stat(cfgfile); err == nil {
		return cfgfile
	}
	return closest_scrapy_cfg(filepath.Dir(path), path)
}

func Inside_scrapy_project() bool {
	to_exec := fmt.Sprintf(PY_CHECK_IMP, SCRAPY_ENVAR)
	_, err := exec.Command("python", "-c", to_exec).Output()
	if err != nil {
		if closest_scrapy_cfg(".", "") != "" {
			return true
		} else {
			return false
		}
	}
	return true
}

// Get Scrapy config file, returns a File (it's a map really) object from go-ini library
func scrapy_get_config() ini.File {
	usr, _ := user.Current()
	dir := usr.HomeDir
	sources := []string{"/etc/scrapy.cfg", "c:\\scrapy\\scrapy.cfg", filepath.Join(dir, ".scrapy.cfg")}

	cl_cfg := closest_scrapy_cfg(".", "")
	if cl_cfg != "" {
		sources = append(sources, cl_cfg)
	}

	var file ini.File
	file = make(ini.File)
	for _, src := range sources {
		// if exist
		if _, err := os.Stat(src); err == nil {
			fread, err := ini.LoadFile(src)
			if err != nil {
				continue
			}
			for k, v := range fread {
				curd, ok := file[k]
				if !ok {
					file[k] = v
				} else {
					for kk, vv := range v {
						curd[kk] = vv
					}
				}
			}
		}
	}
	return file
}

func copy_ini_section(src ini.Section) ini.Section {
	dst := make(ini.Section)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func Scrapy_cfg_targets() ini.File {
	cfg := scrapy_get_config()
	baset, ok := cfg["deploy"]
	if !ok {
		baset = make(ini.Section)
	}
	_, ok = baset["url"]
	if !ok {
		baset["url"] = SCRAPINGHUB_DEPLOY_URL
	}
	targets := make(ini.File)
	targets["default"] = baset
	for sec_name, section := range cfg {
		if strings.HasPrefix(sec_name, "deploy:") {
			tmp := copy_ini_section(baset)
			for k, v := range section {
				tmp[k] = v
			}
			targets[sec_name[7:]] = tmp
		}
	}
	return targets
}

func Scrapy_cfg_target(name string) (ini.Section, error) {
	target, ok := Scrapy_cfg_targets()[name]
	if !ok {
		return nil, errors.New(fmt.Sprintf("Unknown target: %s", name))
	}
	return target, nil
}

func createDefaultSetupPy(settings string) {
	str := fmt.Sprintf(SETUP_PY_TEMPLATE, settings)
	ioutil.WriteFile("setup.py", []byte(str), os.ModeAppend)
}

var err_scrapy_cfg_ver error = errors.New("scray_cfg_version: can't get version")

func Scrapy_cfg_version(versys string) (string, error) {
	if versys == "HG" {
		out, err := exec.Command("hg", "tip", "--template", "{rev}").Output()
		if err != nil {
			return "", err_scrapy_cfg_ver
		}
		rev := fmt.Sprintf("r%s", out)
		out, err = exec.Command("hg", "branch").Output()
		if err != nil {
			return "", err_scrapy_cfg_ver
		}
		branch := strings.Trim(string(out), "\n")
		return fmt.Sprintf("%s-%s", rev, branch), nil
	} else if versys == "GIT" {
		out, err := exec.Command("git", "describe").Output()
		rev := strings.Trim(string(out), "\n")
		if err != nil {
			out, err := exec.Command("git", "rev-list", "--count", "HEAD").Output()
			if err != nil {
				return "", err_scrapy_cfg_ver
			}
			rev = fmt.Sprintf("r%s", strings.Trim(string(out), "\n"))
		}
		out, err = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
		if err != nil {
			return "", err_scrapy_cfg_ver
		}
		branch := strings.Trim(string(out), "\n")
		return fmt.Sprintf("%s-%s", rev, branch), nil

	} else if versys != "" {
		return versys, nil
	} else {
		return fmt.Sprintf(".2f", float64(time.Now().UnixNano())/1000000000.0), nil
	}
}

// Build an egg and returns:
//  => egg file path, tmpdir with egg creation data, error in case exists
func BuildEgg() (string, string, error) {
	closest_cfg := closest_scrapy_cfg(".", "")
	err := os.Chdir(filepath.Dir(closest_cfg))
	if err != nil {
		return "", "", err
	}
	// if not exists 'setup.py'
	if _, err := os.Stat("setup.py"); err != nil {
		settings, ok := scrapy_get_config()["settings"]
		if !ok {
			return "", "", errors.New("BuildEgg: No 'settings' section found on scrapy.cfg")
		}
		createDefaultSetupPy(settings["default"])
	}
	tmpdir, err := ioutil.TempDir(os.TempDir(), "shubc-deploy")
	if err != nil {
		return "", "", errors.New("BuildEgg: Can't create temporary directory")
	}
	cmd := exec.Command("python", "setup.py", "clean", "-a", "bdist_egg", "-d", tmpdir)
	tout, _ := os.Create(filepath.Join(tmpdir, "stdout"))
	terr, _ := os.Create(filepath.Join(tmpdir, "stderr"))
	cmd.Stdout = tout
	cmd.Stderr = terr
	if err := cmd.Run(); err != nil {
		return "", tmpdir, errors.New(fmt.Sprintf("BuildEgg: Can't create egg file - details: %s", err))
	}
	matches, err := filepath.Glob(filepath.Join(tmpdir, "*.egg"))
	if err != nil {
		return "", tmpdir, errors.New(fmt.Sprintf("BuildEgg: No '.egg' file foun on %d", tmpdir))
	}
	tout.Close()
	terr.Close()
	return matches[0], tmpdir, nil
}

// Deploy functions

type DeployMessage struct {
	Status  string
	Message string
	Project string
	Version string
	Spiders int
}

// Add a python egg to the project `project_id` with `name` and `version` given.
func (d *DeployMessage) UploadEgg(conn *Connection, target ini.Section, project_id, version, egg string) (*DeployMessage, error) {
	params := url.Values{}
	params.Add("project", project_id)
	params.Add("version", version)

	url, ok := target["url"]
	if ok && url != "" {
		conn.SetAPIUrl(url)
	}
	content, err := conn.APIPostFilesReadBody("/addversion.json", &params, map[string]string{"egg": egg})
	if err != nil {
		return nil, err
	}

	json.Unmarshal(content, d)
	if d.Status != "ok" {
		return nil, fmt.Errorf("Deploy.UploadEgg: Error ocurred while uploading egg: %s", d.Message)
	}
	return d, nil
}
