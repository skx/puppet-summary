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
    * They are parsed and a simple SQLite database keeps track of data.
* The SQLite database is used to present a visualisation layer.
    * Which you can see [in the screenshots](screenshots/).

The reports are expected to be pruned over time, but as the SQLite database
only contains a summary of the available data it will not grow excessively.

> The current software has been tested with over 50,000 reports and performs well at that scale.


## Installation & Execution

To install this software it should be sufficient to run the following:

    go get -u github.com/skx/puppet-summary

Once installed you can launch it like so:

    $ puppet-summary serve
    Launching the server on http://127.0.0.1:3001

If you wish to change the host/port you can do so like this:

    $ puppet-summary -port 4321 -host 10.10.10.10 serve
    Launching the server on http://10.10.10.10:4321


## Configuring Your Puppet Server

Once you've got an instance of `puppet-summary` installed and running
the next step is to populate it with report data.  The general way to
do that is to update your puppet server to send the reports to it as
they are received.

Edit `puppet.conf` on your puppet-master:

    [master]
    reports = store, http
    reporturl = http://localhost:3001/upload

* If you're running the dashboard on a different host you'll need to use the external IP/hostname here.
* Once configured don't forget to restart your puppet service!

That should be sufficient to make puppet submit reports, where they
can be stored and displayed.

If you don't wish to change your puppet-server initially you can test
what it would look like by importing the existing YAML reports
that are almost certainly present upon your puppet server, adding them
by-hand.

Something like this should do the job:

    # cd /var/lib/puppet/reports
    # find . -name '*.yaml' -exec \
       curl --data-binary @\{\} http://localhost:3001/upload \;

* That assumes that your reports are located beneath `/var/lib/puppet/reports`,
but that is a reasonable default.


## Maintenance

Over time your reports will grow excessively large.  We only display
the most recent 50 upon the per-node page so you might not notice.

To prune (read: delete) old reports run:

    puppet-summary -days 15 prune

That will remove the reports from disk which are > 15 days old, and
also remove the associated SQLite entries that refer to them.


## Notes On Deployment

* Please don't run this application as root.
* Received YAML files are stored beneath `./reports`
* The default SQLite database is `./ps.db`.
    * This can be changed via the command-line, for example:
    * `puppet-summary -db-file local.sqlite3 -port 4323 serve`



 Steve
 --
