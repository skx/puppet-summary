[![Go Report Card](https://goreportcard.com/badge/github.com/skx/puppet-summary)](https://goreportcard.com/report/github.com/skx/puppet-summary)
[![license](https://img.shields.io/github/license/skx/puppet-summary.svg)](https://github.com/skx/puppet-summary/blob/master/LICENSE)
[![Release](https://img.shields.io/github/release/skx/puppet-summary.svg)](https://github.com/skx/puppet-summary/releases/latest)
[![gocover store](http://gocover.io/_badge/github.com/skx/puppet-summary)](http://gocover.io/github.com/skx/puppet-summary)

Table of Contents
=================

* [Puppet Summary](#puppet-summary)
* [Puppet Reporting](#puppet-reporting)
* [Installation](#installation)
  * [Source Installation](#source-installation)
* [Execution](#execution)
* [Importing Puppet State](#importing-puppet-state)
* [Maintenance](#maintenance)
* [Metrics](#metrics)
* [Notes On Deployment](#notes-on-deployment)
  * [Service file for systemd](#service-file-for-systemd)
* [Github Setup](#github-setup)

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

You can get a good idea of what the project does by looking at the screens:

* [SCREENSHOTS.md](SCREENSHOTS.md)

You can also consult the API documentation:

* [API.md](API.md)



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

> The software has [been reported](https://github.com/skx/puppet-summary/issues/42) to cope with 16k reports per day, archive approximately 27Gb of data over 14 days!



## Installation

Installing the service can be done in one of two ways, depending on whether you have the [go](https://golang.org/) toolchain available:

* Download the appropriate binary from our [project release page](https://github.com/skx/puppet-summary/releases).
* Install from source, as documented below.


### Source Installation

If you're planning to make changes to the code, or examine it, then the obvious approach to installing from source is to clone the code, then build and install it from that local clone:

    git clone https://github.com/skx/puppet-summary
    cd puppet-summary
    go install .

You could install directly from source, without cloning the repository as an interim step, by running:

    go install github.com/skx/puppet-summary@master

In either case you'll find a binary named `puppet-summary` placed inside a directory named `bin` beneath the golang GOPATH directory.  To see exactly where this is please run:

    echo $(go env GOPATH)/bin

(Typically you'd find binaries deployed to the directory `~/go/bin`, however this might vary.)



## Execution

Once installed you can launch it directly like so:

    $ puppet-summary serve
    Launching the server on http://127.0.0.1:3001

If you wish to change the host/port you can do so like this:

    $ puppet-summary serve -host 10.10.10.10 -port 4321
    Launching the server on http://10.10.10.10:4321

To have it listen on any available IP address, use one of these examples:

    $ puppet-summary serve -host "" -port 4321
    $ puppet-summary serve -host 0.0.0.0 -port 4321

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

Over time your reports will start to consuming ever-increasing amounts of disk-space so they should be pruned.  To prune (read: delete) old reports run:

    puppet-summary prune -days 7 -prefix ./reports/

That will remove the saved YAML files from disk which are over 7 days old, and it will _also_ remove the associated database entries that refer to them.

If you're happy with the default pruning behaviour, which is particularly useful when you're running this software in a container, described in [HACKING.md](HACKING.md), you can prune old reports automatically once per week without the need to add a cron-job like so:

    puppet-summary serve  -auto-prune [options..]

If you don't do this you'll need to __add a cronjob__ to ensure that the prune-subcommand runs regularly.

Nodes which had previously submitted updates to your puppet-master, and `puppet-summary` service, but which have failed to do so "recently", will be listed in the web-based user-interface, in the "orphaned" column.  Orphaned nodes will be reaped over time, via the `days` option just discussed.  If you explicitly wish to clean removed-hosts you can do so via:

    puppet-summary prune -verbose -orphaned



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


### Service file for systemd

You can find instructions on how to create a service file for systemd in the [samples](samples) directory.



## Github Setup

This repository is configured to run tests upon every commit, and when
pull-requests are created/updated.  The testing is carried out via
[.github/run-tests.sh](.github/run-tests.sh) which is used by the
[github-action-tester](https://github.com/skx/github-action-tester) action.

Releases are automated in a similar fashion via [.github/build](.github/build),
and the [github-action-publish-binaries](https://github.com/skx/github-action-publish-binaries) action.


Steve
--
