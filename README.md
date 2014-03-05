Scrapinghub API Go library
==========================

Go bindings for Scrapinghub HTTP API and a command line tool.

Installation
============

_Requirements_

* Golang >= 1.1 

_Steps_

    $ go get github.com/andrix/shubc
    $ go install github.com/andrix/shubc

shubc: a command line tool
--------------------------

Also it's bundled a handy command line to query the API.

Getting help

    % shubc help
    shubc [options] <command> arg1 .. argN

     Commands: 
       spiders <project_id>                       - list the spiders on project_id
       jobs <project_id> [filters]                - list the last 100 jobs on project_id
       jobinfo <job_id>                           - print information about the job with <job_id>
       schedule <project_id> <spider_name> [args] - schedule the spider <spider_name> with [args] in project <project_id>
       stop <job_id>                              - stop the job with <job_id>
       ...

### Configure your APIKEY

You can configure your APIKEY using the .scrapy.cfg file in your home. You can get more information on how to configure it here: http://doc.scrapinghub.com/scrapy-cloud.html#deploying-your-scrapy-spider

### Available Commands

#### Spiders API

* `spiders <project-id>`: list the spiders on `project-id`

#### Jobs API

* `schedule <project-id> <spider-name> [args]`: schedule the spider `spider-name` with `args` in project `project-id`
* `reschedule <job_id>`: re-schedule the job `job_id` with the same arguments and tags
* `jobs <project-id> [filters]`: list the last 100 jobs on `project-id` (accept `-count` parameter). Filters are in the form: `state=running`, `spider=spider1`, etc. If `-raw` option is given output the jobs as JsonLines to Stdout.
* `jobinfo <job-id>`: print information about the job with `job-id`;
* `update <job-id> [args]`: update the job with `job_id` using the `args` given
* `stop <job-id>`: stop the job with `job-id`
* `delete <job-id>`: delete the job with `job- id`

#### Items API

* `items <job-id>`: print to stdout the items for `job-id` (`count` & `offset` available). If `-raw` option is given output the jobs as JsonLines to Stdout.

#### Log API

* `log <job-id>`: print to Stdout the log for job `job-id`

#### Autoscraping API

* `project-slybot <project-id> [spiders]`: download the zip and write it to Stdout or o.zip if `-o` option is given

#### Eggs API

* `eggs-add <project_id> <path> [name=n version=v]`: add the egg in `path` to the project `project_id`. By default it guess the name and version from `path`, but can be given using name=eggname and version=XXX
* `eggs-list <project_id>`: list the eggs in `project_id`
* `eggs-delete <project_id> <egg_name>`: delete the egg `egg_name` in the project `project_id`
