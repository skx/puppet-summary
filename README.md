Puppet Summary
==============

This is a simple [golang](https://golang.org/) based project which is designed to offer a dashboard of your current puppet-infrastructure:

* Listing all known-nodes, and their current state.
* Viewing the last few runs of a given system.
* etc.

There are [screenshots included within this repository](screenshots/).

In short:

* Your puppet-server submits reports to this software.
    * The reports are saved locally, as YAML files, beneath `./reports`
    * They are parsed and a simple SQLite database keeps track of data.
* The SQLite database is used to present a visualisation layer.

The reports are expected to be pruned over time, but as the SQLite database
only contains a summary of the available data it will not grow excessively.

> The current software has been tested with over 50,000 reports and performs well at that scale.


## Installation

To install this software it should be sufficient to run:

    go get github.com/skx/puppet-summary

Once installed you need to launch it like so:

    puppet-summary

Then configure your puppet-server to send its reports to the host.
Edit `puppet.conf` on your server:

    [master]
    reports = store, http
    reporturl = http://localhost:3001/upload

**NOTE**: Once configured don't forget to restart your puppet service!



## Testing

If you don't wish to install it for real, updating your puppet-server,
and running in production, you can instead instead just pretend you're
running it!  Assuming you have a bunch of YAML files stored upon your
puppet-server, probably beneath `/var/lib/puppet/reports`, you can copy
them to your local system, then submit them to the server running locally:

    find . -name '*.yaml' -exec curl --data-binary @\{\} http://localhost:3001/upload \;



## Editing the HTML Templates

The generated HTML views are stored inside the compiled binary to ease
deployment.  If you wish to tweak the look & feel by editing them then
you're more then welcome.

The raw HTML-templates are located beneath `data/`, and you can edit them
then rebuild the compiled versions via `go-bindata`.

Install `go-bindata` like this, if you don't already have it present:

     go get -u github.com/jteeuwen/go-bindata/...

Now regenerate the compiled version(s) of the templates and rebuild the
binary to make your changes:

    go-bindata -nomemcopy data/
    go build .


## End-Points

There are four main end-points:

* `GET /`
  * Show all known-nodes and their current status.
* `GET /node/${fqdn}`
   * Shows the last N (max 50) runs of puppet against the given node.
   * This includes a graph of run-time.
* `GET /report/${n}`
   * This shows useful output of a given run.
* `POST /upload`
   * Store a report, this is expected to be invoked from the puppet-master.


## Notes On Deployment

* Received YAML files are stored beneath `./reports`
* The SQLite database is `./foo.db`.
  * These will become more flexible in the future.

Obviously don't run this as root.

 Steve
 --
