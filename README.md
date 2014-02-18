shubc
=====

Go bindings for Scrapinghub HTTP API and a command line tool.

scrapinghub package
===================

It has the binding in Go for the HTTP API of Scrapy Cloud

Installation
------------

    $ go get github.com/andrix/shubc
    $ cd shubc
    $ go install shubc

Command line tool
=================

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
 
Available Commands
------------------

* spiders <project_id> : list the spiders on project_id
* jobs <project_id> [filters] : list the last 100 jobs on project_id (accept -count parameter). Filters are in the form: state=running, spider=spider1, etc
* jobinfo <job_id> : print information about the job with <job_id>
* schedule <project_id> <spider_name> [args] : schedule the spider <spider_name> with [args] in project <project_id>
* stop <job_id> : stop the job with <job_id>
* items <job_id> : print to stdout the items for <job_id> (count & offset available)

