# mq-golang
This repository demonstrates how you can call IBM MQ from applications written in the Go language.

The repository also includes programs to export MQ statistics to some monitoring
packages including Prometheus, InfluxDB and OpenTSDB.

A minimum level of MQ V9 is required to build this package.
The monitoring data published by the queue manager is not available before
that version; the interface also assumes availability of
MQI structures from that level of MQ.

## MQI Description

The ibmmq directory contains a Go package, exposing an MQI-like interface.

The intention is to give an API that is more natural for Go programmers than the common
procedural MQI. For example, fixed length string arrays from the C API such as MQCHAR48 are
represented by the native Go string type. Conversion between these types is handled within the ibmmq
package itself, removing the need for Go programmers to know about it.

A short program in the mqitest directory gives an example of using this interface, to put and get messages
and to subscribe to a topic.

Feedback on the utility of this package, thoughts about whether it should be changed or extended are
welcomed.

## Using the package

To use code in this repository, you will need to be able to build Go applications, and
have a copy of MQ installed to build against. It uses cgo to access the MQI C structures and definitions. It assumes that MQ has been
installed in the default location on a Linux platform (/opt/mqm) but you can easily change the
cgo directives in the source files if necessary.

Some Windows capability is also included. This has been tested with Go 1.10
compiler, which now permits standard Windows paths (eg including spaces)
so the CGO directives can point at the normal MQ install path.

## Getting started

If you are unfamiliar with Go, the following steps can help create a
working environment with source code in a suitable tree. Initial setup
tends to be platform-specific, but subsequent steps are independent of the
platform.

### Linux

* Install the Go runtime and compiler. On Linux, the packaging may vary
but a typical directory for the code is /usr/lib/golang.
* Create a working directory. For example, mkdir $HOME/gowork
* Set environment variables. Based on the previous lines,

  export GOROOT=/usr/lib/golang

  export GOPATH=$HOME/gowork

* If using a version of Go from after 2017, you must set environment variables
to permit some compile/link flags. This is due to a security fix in the compiler.
  export CGO_LDFLAGS_ALLOW="-Wl,-rpath.*"

* Install the git client

### Windows

* Install the Go runtime and compiler. On Windows, the
common directory is c:\Go
* Ensure you have a gcc-based compiler, for example from the Cygwin
distribution. I use the mingw variation, to ensure compiled code can
be used on systems without Cygwin installed
* Create a working directory. For example, mkdir c:\Gowork
* Set environment variables. Based on the previous lines,

  set GOROOT=c:\Go

  set GOPATH=c:\Gowork

  set CC=x86_64-w64-mingw32-gcc.exe

* The CGO_LDFLAGS_ALLOW variable is not needed on Windows
* Install the git client
* Make sure the MQ include files and libraries are in a path that does
not include spaces or other special characters, as discussed above.

### Common

* Make sure your PATH includes routes to the Go compiler ($GOROOT/bin),
the Git client, and the C compiler.
* Change directory to the workspace you created earlier. (cd $GOPATH)
* Use git to get a copy of the MQ components into a new directory in the
workspace. Use "src" as the destination, to get the directory created
automatically; this path will then be searched by the Go compiler.

  git clone http://github.com/ibm-messaging/mq-golang src

* Use Go to download prerequisite components for any monitors you are interested
in running. The logrus package is required for all of the monitors; but not
all of the monitors require further downloads.

  go get -u github.com/Sirupsen/logrus

  go get -u github.com/prometheus/client_golang/prometheus

  go get -u github.com/influxdata/influxdb/client/v2

  go get -u github.com/aws/aws-sdk-go/service

* Compile the components you are interested in. For example

  go install ./src/cmd/mq_prometheus

At this point, you should have a compiled copy of the code in $GOPATH/bin.

## Limitations

Not all of the MQI verbs are available through the ibmmq package. This
implementation concentrates on the core API calls needed to put and get messages.
Currently unavailable verbs include:
* MQSET
* All of the message property manipulators
* MQCB

There are also no structure handlers for message headers such as MQRFH2 or MQDLH.

## History

See [CHANGES](https://github.com/ibm-messaging/mq-golang/CHANGES.md).

## Health Warning

This package is provided as-is with no guarantees of support or updates. There are also no guarantees of compatibility
with any future versions of the package; the API is subject to change based on any feedback.

## Issues and Contributions

For feedback and issues relating specifically to this package, please use the [GitHub issue tracker](https://github.com/ibm-messaging/mq-golang/issues).

Contributions to this package can be accepted under the terms of the IBM Contributor License Agreement,
found in the file CLA.md of this repository. When submitting a pull request, you must include a statement stating
you accept the terms in CLA.md.

