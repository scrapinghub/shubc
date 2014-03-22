// Go bindings for Scrapinghub API (http://doc.scrapinghub.com/api.html)
package scrapinghub

import "fmt"

var libversion = "0.1"
var APIURL = "https://dash.scrapinghub.com/api"
var USER_AGENT = fmt.Sprintf("scrapinghub.go/%s (http://github.com/scrapinghub/shubc)", libversion)
