Scrapinghub API Go library
==========================

Go bindings for Scrapinghub HTTP API and a command line tool.

Installation
============

_Requirements_

* Golang >= 1.0 

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
 
### Available Commands

* spiders &lt;project-id&gt; : list the spiders on `project-id`
* jobs &lt;project-id&gt; [filters] : list the last 100 jobs on `project-id` (accept `-count` parameter). Filters are in the form: `state=running`, `spider=spider1`, etc. If `-raw` option is given output the jobs as JsonLines to Stdout.
* jobinfo &lt;job-id&gt; : print information about the job with `job-id`;
* schedule &lt;project-id&gt; &lt;spider-name&gt; [args] : schedule the spider `spider-name` with `args` in project `project-id`
* stop &lt;job-id&gt; : stop the job with `job-id`
* items &lt;job-id&gt; : print to stdout the items for `job-id` (`count` & `offset` available). If `-raw` option is given output the jobs as JsonLines to Stdout.
* project-slybot &lt;project-id&gt; [spiders]: download the zip and write it to Stdout or o.zip if `-o` option is given
* log &lt;job-id&gt;: print to Stdout the log for job `job-id`
