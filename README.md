[![Travis CI](https://img.shields.io/travis/skx/puppet-summary/master.svg?style=flat-square)](https://travis-ci.org/skx/puppet-summary)
[![license](https://img.shields.io/github/license/skx/puppet-summary.svg)](https://github.com/skx/puppet-summary/blob/master/LICENSE)


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
   * The metrics include the puppet-runtime of each node too.

There are [screenshots included within this repository](screenshots/).


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


## Maintenance & Metrics

Over time your reports will start to consuming ever-increasing amounts
of disk-space so they should be pruned.  To prune (read: delete) old reports
run:

    puppet-summary prune -days 15

That will remove the reports from disk which are > 15 days old, and
also remove the associated SQLite entries that refer to them.

If you have a carbon-server running locally you can also submit metrics
to it :

    puppet-summary metrics \
      -host carbon.example.com \
      -port 2003 \
      -prefix puppet.example_com  [-nop]

The metrics will include:

* A count of nodes in each state, for example:
  * `puppet.example_com.state.failed 3`
  * `puppet.example_com.state.changed 1`
  * `puppet.example_com.state.unchanged 297`
  * `..`
* The runtime of each node, for example:
  * `puppet.example_com.www1_example_com.runtime 23.21`
  * `puppet.example_com.www2_example_com.runtime 23.21`
  * `..`

When running with `-nop` the metrics will be dumped to the console instead
of submitted.  For example:

     $ puppet-summary metrics -nop
     ..
     puppet.rsync_io.runtime 2.1318351220327783
     puppet.www_steve_fi.runtime 3.1109390222183892
     ..
     puppet.state.unchanged 1
     puppet.state.changed 1
     puppet.state.failed  0


## Notes On Deployment

* Please don't run this application as root.
* Rather than exposing it publicly you should prefer to run behind an `nginx`/`apache2` reverse-proxy.
* Received YAML files are stored beneath `./reports`
    * This can be changed with the `-prefix` argument to `puppet-summary serve`
* The default SQLite database is located at `./ps.db`, but it can be changed via the command-line, for example:
    * `puppet-summary serve -db-file local.sqlite3`
    * **NOTE** If you change the default you'll need to specify this for all commands, e.g. `serve`, `prune`, and `metrics`.

 Steve
 --
