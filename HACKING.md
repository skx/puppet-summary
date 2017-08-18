
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


## Code Quality

Test our the test-suite coverage :

     go get golang.org/x/tools/cmd/cover
     go test -coverprofile fmt

Look for advice on code:

     go vet .
