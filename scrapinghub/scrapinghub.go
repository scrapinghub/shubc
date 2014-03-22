// Go bindings for Scrapinghub API (http://doc.scrapinghub.com/api.html)
package scrapinghub

import "fmt"

var libversion = "0.1"

// The default url for Scrapinghub API
var APIURL = "https://dash.scrapinghub.com/api"

// User-Agent which the library will be identified when querying the API
var USER_AGENT = fmt.Sprintf("scrapinghub.go/%s (http://github.com/scrapinghub/shubc)", libversion)
