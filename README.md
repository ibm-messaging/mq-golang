# mq-golang

This repository demonstrates how you can call IBM MQ from applications written in the Go language.

> **NOTICE**: Please ensure that you use a dependency management tool such as [dep](https://github.com/golang/dep) or [Glide](http://glide.sh/), and add a specific version dependency.

This repository previously contained sample programs that exported MQ statistics to
some monitoring packages. These have now been moved to a
new [GitHub repository called mq-metric-samples](https://github.com/ibm-messaging/mq-metric-samples).

A minimum level of MQ V8 is required to build these packages. However, note that
the monitoring data published by the queue manager is not available before MQ V9.

## Health Warning

This package is provided as-is with no guarantees of support or updates. There are
also no guarantees of compatibility with any future versions of the package; the API
is subject to change based on any feedback. Versioned releases are made in this repository
to assist with using stable APIs.

## MQI Description

The ibmmq directory contains a Go package, exposing an MQI-like interface.

The intention is to give an API that is more natural for Go programmers than the
common procedural MQI. For example, fixed length string arrays from the C API such
as MQCHAR48 are represented by the native Go string type. Conversion between these
types is handled within the ibmmq package itself, removing the need for Go programmers
to know about it.

Sample programs are provided to demonstrate various features of using the MQI. See the
README in the `samples` directory for more information about those programs.

The mqmetric directory contains functions to help monitoring programs access MQ status and
statistics. This package is not needed for general application programs.

## Using the package

To use code in this repository, you will need to be able to build Go applications, and
have a copy of MQ installed to build against. It uses cgo to access the MQI C
structures and definitions. It assumes that MQ has been installed in the default
location (on a Linux platform this would be `/opt/mqm`) but this can be changed
with environment variables if necessary.

Windows compatibility is also included. This has been tested with Go 1.10 compiler,
which now permits standard Windows paths (eg including spaces) so the CGO directives
can point at the normal MQ install path.

## Getting started

If you are unfamiliar with Go, the following steps can help create a working environment
with source code in a suitable tree. Initial setup tends to be platform-specific,
but subsequent steps are independent of the platform.

### Linux

* Install the Go runtime and compiler. On Linux, the packaging may vary but a typical
directory for the code is `/usr/lib/golang`.

* Create a working directory. For example, ```mkdir $HOME/gowork```

* Set environment variables. Based on the previous lines,

```
  export GOROOT=/usr/lib/golang
  export GOPATH=$HOME/gowork
```

* If using a version of Go from after 2017, you must set environment variables to permit some compile/link flags. This is due to a security fix in the compiler.

```
export CGO_LDFLAGS_ALLOW="-Wl,-rpath.*"
```

* Install the git client

### Windows

* Install the Go runtime and compiler. On Windows, the common directory is `c:\Go`
* Ensure you have a gcc-based compiler, for example from the Cygwin distribution. I use the mingw variation, to ensure compiled code can be used on systems without Cygwin installed
* Create a working directory. For example, `mkdir c:\Gowork`
* Set environment variables. Based on the previous lines,

```
set GOROOT=c:\Go
set GOPATH=c:\Gowork
set CC=x86_64-w64-mingw32-gcc.exe
```

* The `CGO_LDFLAGS_ALLOW` variable is not needed on Windows
* Install the git client

### Common

* Make sure your PATH includes routes to the Go compiler (`$GOROOT/bin`), the Git client, and the C compiler.
* Change directory to the workspace you created earlier. (`cd $GOPATH`)
* Use git to get a copy of the MQ components into a new directory in the workspace.

  `git clone https://github.com/ibm-messaging/mq-golang.git src/github.com/ibm-messaging/mq-golang`

* If you have not installed MQ libraries into the default location, then set environment variables
for the C compiler to recognise those directories. For example,

```
   export CGO_CFLAGS="-I/my/mq/dir/inc"
   export CGO_LDFLAGS="-I/my/mq/dir/lib64"
```

* Compile the `ibmmq` component:

  `go install ./src/github.com/ibm-messaging/mq-golang/ibmmq`

* If you plan to use monitoring functions, then compile the `mqmetric` component:

  `go install ./src/github.com/ibm-messaging/mq-golang/mqmetric`

* Sample programs can be compiled in this way

  `go build -o bin/mqitest ./src/github.com/ibm-messaging/mq-golang/samples/mqitest/*.go`

At this point, you should have a compiled copy of the program in `$GOPATH/bin`. See the
`samples` directory for more sample programs.

## Limitations

All regular MQI verbs are now available through the `ibmmq` package.

There are no structure handlers for message headers such as MQRFH2 or MQDLH.

## History

See [CHANGELOG](CHANGELOG.md) in this directory.

## Issues and Contributions

Feedback on the utility of this package, thoughts about whether it should be changed
or extended are welcomed.

For feedback and issues relating specifically to this package, please use
the [GitHub issue tracker](https://github.com/ibm-messaging/mq-golang/issues).

Contributions to this package can be accepted under the terms of the IBM Contributor
License Agreement, found in the [CLA file](CLA.md) of this repository. When
submitting a pull request, you must include a statement stating you accept the terms
in the CLA.

## Copyright

© Copyright IBM Corporation 2016, 2018
