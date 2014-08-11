/*
 * Deploy module
 *
 */

package scrapinghub

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/vaughan0/go-ini"
)

const SCRAPY_ENVAR = "SCRAPY_SETTINGS_MODULE"
const PY_CHECK_IMP = "import os, importlib; importlib.import_module(os.environ.get('%s'))"
const SCRAPINGHUB_DEPLOY_URL = "http://dash.scrapinghub.com/api/scrapyd/"

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

//TODO: implement
func Deploy(target, project, version string, egg string) bool {
	return false
}
