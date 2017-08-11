
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

In addition to that there is a simple end-point which is designed to
return a list of all the nodes in the given state:

* `GET /api/state/$state`

Only valid states are permitted (`changed`, `failed`, & `unchanged`).



## Code Quality

Test our the test-suite coverage :

     go get golang.org/x/tools/cmd/cover
     go test -coverprofile fmt

Look for advice on code:

     go vet .
