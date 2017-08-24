
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

To test the coverage of the test-suite you can use the `cover` tool:

     go get golang.org/x/tools/cmd/cover
     go test -coverprofile fmt

Once you've done that you can view the coverage of various functions via:

     go tool cover -func=fmt

If you're looking for suggestions on code-quality you can use the `go vet`
tool:

     go vet .

# Cross compiling puppet-summary

In this example, the compilation is happening on x86_64 Fedora or a Debian 9 amd64 system with a target of Raspbian on ARM (raspberry PI).

## Install the packages you need.

### Fedora
`# dnf install binutils-arm-linux-gnu cross-gcc-common cross-binutils-common gcc-c++-arm-linux-gnu kernel-cross-headers glibc-arm-linux-gnu  glibc-arm-linux-gnu-devel`

### Debian
`# apt-get install  cpp-6-arm-linux-gnueabihf g++-6-arm-linux-gnueabihf gcc-6-arm-linux-gnueabihf gcc-6-arm-linux-gnueabihf-base gccgo-6-arm-linux-gnueabihf`

## Manually fix pthreads

_Note:_ This is only required on Fedora builders.

The way cgo works for cross compiles, it assumes a sysroot, which is normal. However, the way pthreads is called in the github.com/mattn/go-sqlite3 package, it requires and absolute path, but that path is relative to the sysroot provided.

`# pushd /usr/arm-linux-gnu; ln -s /usr .; popd`

## Compile

I use `-v` when cross compiling because it will give much more info if something errors out.

### Fedora

`$ CC=arm-linux-gnu-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm CGO_LDFLAGS=--sysroot=/usr/arm-linux-gnu CGO_CFLAGS=--sysroot=/usr/arm-linux-gnu go build -v .`

### Debian

`$ CC=arm-linux-gnueabihf-gcc-6  CGO_ENABLED=1 GOOS=linux GOARCH=arm CGO_LDFLAGS=--sysroot=/usr/arm-linux-gnu CGO_CFLAGS=--sysroot=/usr/arm-linux-gnu go build -v .`

## Verify build

`$ file puppet-summary

puppet-summary: ELF 32-bit LSB executable, ARM, EABI5 version 1 (SYSV), dynamically linked, interpreter /lib/ld-linux-armhf.so.3, for GNU/Linux 3.2.0, BuildID[sha1]=810382dc0c531df0de230c2f681925d9ebf59fd6, with debug_info, not stripped`
