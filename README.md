Scrapinghub API Go library
==========================

Go bindings for Scrapinghub HTTP API and a command line tool.

Installation
============

_Requirements_

* Golang >= 1.1 
* go-ini : https://github.com/vaughan0/go-ini

_Steps_

    $ go get github.com/tools/godep
    $ godep get github.com/scrapinghub/shubc
    $ shubc
    Usage: shubc [options] url

scrapinghub.go: the library
---------------------------

Documentation for the library is online at godoc.org:

- [scrapinghub.go documentation](https://godoc.org/github.com/scrapinghub/shubc/scrapinghub)

shubc: a command line tool
--------------------------

Also it's bundled a handy command line to query the API.

Getting help

    % shubc help
    shubc [options] <command> arg1 .. argN

     Options: 
      -apikey="<API KEY>": Scrapinghub api key
      -apiurl="https://dash.scrapinghub.com/api": Scrapinghub API URL (can be changed to another uri for testing).
      -count=0: Count for those commands that need a count limit
      -csv=false: If given, for command items, they will retrieve as CSV writing to os.Stdout
      -fields="": When -csv given, list of comma separated fields to include in the CSV
      -include_headers=false: When -csv given, include the headers of the CSV in the output
      -jl=false: If given, for command items and jobs will retrieve all the data writing to os.Stdout as JsonLines format
      -o="": Write output to a file instead of Stdout
      -offset=0: Number of results to skip from the beginning
      -tail=false: The same that `tail -f` for command `log`

     Commands: 
       Spiders API: 
         spiders <project_id>                       - list the spiders on project_id
       Jobs API: 
         schedule <project_id> <spider_name> [args] - schedule the spider <spider_name> with [args] in project <project_id>
         reschedule <job_id>                        - re-schedule the job `job_id` with the same arguments and tags
         jobs <project_id> [filters]                - list the last 100 jobs on project_id
         jobinfo <job_id>                           - print information about the job with <job_id>
    ...
    ...

### Configure your APIKEY

You can configure your APIKEY using the .scrapy.cfg file in your home. You can get more information on how to configure it here: http://doc.scrapinghub.com/scrapy-cloud.html#deploying-your-scrapy-spider

### Options

* `-apikey` : Scrapinghub api key
* `-apiurl` : Scrapinghub API URL, by default is "https://dash.scrapinghub.com/api" but can be changed to another uri for testing.
* `-count`  : Count for those commands that need a count limit, default=`0` 
* `-csv` : For command `items`, if given, it will retrieve the data as CSV writing to os.Stdout, default=`false`
* `-fields` : For command `items` and when `-csv` option is given, is the list of fields to include in the CSV (e.g: -fields=name,address,etc.)
* `-include_headers` : For command `items` and when `-csv` is given, include the headers of the CSV in the output, default=`false`
* `-jl` : For commands `items` and `jobs`, if given will retrieve all the data writing to os.Stdout as JsonLines format, default=false
* `-o` : Write output to a file instead of Stdout
* `-offset`: Number of results to skip from the beginning, default=`0`
* `-tail` : The same that `tail -f` for command `log`, default=`false`

### Commands

#### Spiders API

* `spiders <project-id>`: list the spiders on `project-id`

#### Jobs API

* `schedule <project-id> <spider-name> [args]`: schedule the spider `spider-name` with `args` in project `project-id`
* `reschedule <job_id>`: re-schedule the job `job_id` with the same arguments and tags
* `jobs <project-id> [filters]`: list the last 100 jobs on `project-id` (accept `-count` parameter). Filters are in the form: `state=running`, `spider=spider1`, etc. Avail. options: `-jl`, `-csv`
* `jobinfo <job-id>`: print information about the job with `job-id`
* `update <job-id> [args]`: update the job with `job_id` using the `args` given
* `stop <job-id>`: stop the job with `job-id`
* `delete <job-id>`: delete the job with `job- id`

#### Items API

* `items <job-id>`: print to stdout the items for `job-id` (`count` & `offset` available). Avail. options: `-jl`, `-csv`

#### Log API

* `log <job-id>`: print to Stdout the log for job `job-id`. Avail. options: `-tail`

#### Autoscraping API

* `project-slybot <project-id> [spiders]`: download the zip and write it to Stdout or o.zip if `-o` option is given

#### Eggs API

* `eggs-add <project_id> <path> [name=n version=v]`: add the egg in `path` to the project `project_id`. By default it guess the name and version from `path`, but can be given using name=eggname and version=XXX
* `eggs-list <project_id>`: list the eggs in `project_id`
* `eggs-delete <project_id> <egg_name>`: delete the egg `egg_name` in the project `project_id`

#### Deploy

* `deploy <target> [project_id=<project_id>] [egg=<egg>] [version=<version>]`: deploy `target` to Scrapy Cloud
* `deploy-list-targets`: list available targets to deploy
* `build-egg`: just build the egg file
