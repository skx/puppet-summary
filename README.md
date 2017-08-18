[![Travis CI](https://img.shields.io/travis/skx/puppet-summary/master.svg?style=flat-square)](https://travis-ci.org/skx/puppet-summary)
[![Go Report Card](https://goreportcard.com/badge/github.com/skx/puppet-summary)](https://goreportcard.com/report/github.com/skx/puppet-summary)
[![license](https://img.shields.io/github/license/skx/puppet-summary.svg)](https://github.com/skx/puppet-summary/blob/master/LICENSE)
[![Release](https://github-release-version.herokuapp.com/github/skx/puppet-summary/release.svg?style=flat)](https://github.com/skx/puppet-summary/releases/latest)

Puppet Summary
==============

This is a simple [golang](https://golang.org/) based project which is designed to offer a dashboard of your current puppet-infrastructure:

* Listing all known-nodes, and their current state.
* Viewing the last few runs of a given system.
* etc.

This project is directly inspired by the [puppet-dashboard](https://github.com/sodabrew/puppet-dashboard) project, reasons why you might prefer _this_ project:

* It is actively maintained.
   * Unlike [puppet-dashboard](https://github.com/sodabrew/puppet-dashboard/issues/341).
* Deployment is significantly simpler.
   * This project only involves deploying a single binary.
* It allows you to submit metrics to a carbon-receiver.
   * The metrics include a distinct count of each state, allowing you to raise alerts when nodes in the failed state are present.
* The output can be used for scripting, and automation.
   * All output is available as [JSON/XML](API.md) in addition to human-viewable HTML.

There are [screenshots included within this repository](screenshots/), and you can view a [sample installation here](https://master.steve.org.uk/).


## Puppet Reporting

The puppet-server has integrated support for submitting reports to
a central location, via HTTP POSTs.   This project is designed to be
a target for such submission:

* Your puppet-master submits reports to this software.
    * The reports are saved locally, as YAML files, beneath `./reports`
    * They are parsed and a simple SQLite database keeps track of them.
* The SQLite database is used to present a visualization layer.
    * Which you can see [in the screenshots](screenshots/).

The reports are expected to be pruned over time, but as the SQLite database
only contains a summary of the available data it will not grow excessively.

> The current software has been tested with over 50,000 reports and performs well at that scale.


## Installation

Providing you have a working go-installation you should be able to
install this software by running:

    go get -u github.com/skx/puppet-summary

> **NOTE**: If you've previously downloaded the code this will update your installation to the most recent available version.

If you don't have a golang environment setup you should be able to download a binary for GNU/Linux from the github release page:

* [Binary Release for GNU/Linux - 64-bit](https://github.com/skx/puppet-summary/releases)


## Execution

Once installed you can launch it directly like so:

    $ puppet-summary serve
    Launching the server on http://127.0.0.1:3001

If you wish to change the host/port you can do so like this:

    $ puppet-summary serve -host 10.10.10.10 -port 4321
    Launching the server on http://10.10.10.10:4321

Other sub-commands are described later, or can be viewed via:

    $ puppet-summary help


## Importing Puppet State

Once you've got an instance of `puppet-summary` installed and running
the next step is to populate it with report data.  The expectation is
that you'll update your puppet server to send the reports to it directly,
by editing `puppet.conf` on your puppet-master:

    [master]
    reports = store, http
    reporturl = http://localhost:3001/upload

* If you're running the dashboard on a different host you'll need to use the external IP/hostname here.
* Once you've changed your master's configuration don't forget to restart the service!

If you __don't__ wish to change your puppet-server initially you can test
what it would look like by importing the existing YAML reports from your
puppet-master.  Something like this should do the job:

    # cd /var/lib/puppet/reports
    # find . -name '*.yaml' -exec \
       curl --data-binary @\{\} http://localhost:3001/upload \;

* That assumes that your reports are located beneath `/var/lib/puppet/reports`,
but that is a reasonable default.
* It also assumes you're running the `puppet-summary` instance upon the puppet-master, if you're on a different host remember to change the URI.


## Maintenance

Over time your reports will start to consuming ever-increasing amounts
of disk-space so they should be pruned.  To prune (read: delete) old reports
run:

    puppet-summary prune -days 15 -prefix ./reports/

That will remove the saved YAML files from disk which are > 15 days old, and
also remove the associated database entries that refer to them.

## Metrics

If you have a carbon-server running locally you can also submit metrics
to it :

    puppet-summary metrics \
      -host carbon.example.com \
      -port 2003 \
      -prefix puppet.example_com  [-nop]

The metrics include the count of nodes in each state, `changed`, `unchanged`, `failed`, and `orphaned` and can be used to raise alerts when things fail.  When running with `-nop` the metrics will be dumped to the console instead of submitted.


## Notes On Deployment

If you can run this software upon your puppet-master then that's the ideal, that way your puppet-master would be configured to uploaded your reports to `127.0.0.1:3001/upload`, and the dashboard itself may be viewed via a reverse-proxy.

The appeal of allowing submissions from the loopback is that your reverse-proxy can deny access to the upload end-point, ensuring nobody else can submit details.  A simple nginx configure might look like this:

     server {
         server_name reports.example.com;
         listen [::]:80  default ipv6only=off;

         ## Puppet-master is the only host that needs access here
         ## it is configured to POST to localhost:3001 directly
         ## so we can disable access here.
         location /upload {
            deny all;
         }

         ## send all traffic to the back-end
         location / {
           proxy_pass  http://127.0.0.1:3001;
           proxy_redirect off;
           proxy_set_header        X-Forwarded-For $remote_addr;
         }
     }

* Please don't run this application as root.
* The defaults are sane, YAML files are stored beneath `./reports`, and the SQLite database is located at "`./ps.db`.
    * Both these values can be changed, but if you change them you'll need to remember to change for all appropriate actions.
      * For example "`puppet-summary serve -db-file ./new.db`",  "`puppet-summary metrics -db-file ./new.db`", and "`puppet-summary prune -db-file ./new.db`".


Steve
 --
