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

Some Windows capability is also included. One constraint in the cgo package is its (lack of) support
for path names containing spaces and special characters, which makes it tricky to
compile against a copy of MQ installed in the regular location. To build these packages I copied
<mq install>/tools/c/include and <mq install>/bin64 to be under a temporary directory, shown
in the CFLAGS and LDFLAGS directives.

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

08 Jul 2016
* Initial release

18 Jul 2016
* Changed structures so that most applications will not need to use cgo to imbed the MQ C headers
  * Go programs will now use int32 where C programs use MQLONG
  * Use of message handles, distribution lists require cgo for now
* Package ibmmq now includes the numeric #defines as a Go file, cmqc.go, for easier use
* Removed "src/" prefix from tree in github repo
* Removed need for buffer length parm on Put/Put1
* Updated comments
* Added MQINQ
* Added MQItoString function for some maps of values to constant names

25 Jul 2016
* Added functions to handle basic PCF creation and parsing
* Added a monitor command for exporting MQ V9 queue manager data to Prometheus. See
the [README](cmd/mq_prometheus/README.md) for more details

04 Aug 2016
* Added a monitor command for exporting MQ data to InfluxDB. See the [README]
(cmd/mq_influx/README.md) for more details
* Restructured the monitoring code to put common material in the mqmetric
package, called from the Influx and Prometheus monitors.

12 Aug 2016
* Added a OpenTSDB monitor. See the [README](cmd/mq_opentsdb/README.md) for
more details.
* Added a Collectd monitor. See the [README](cmd/mq_coll/README.md) for
more details.
* Added MQI MQCNO/MQCSP structures to support client connections and password authentication
with MQCONNX.
* Allow client-mode connections from the monitor programs
* Added Grafana dashboards for the different monitors to show how to query them
* Changed database password mechanism so that "exec" maintains the PID for MQ services

23 Aug 2016
* Added a collector for Amazon AWS CloudWatch monitoring. See the [README](cmd/mq_aws/README.md)
for more details.

17 Oct 2016
* Added some Windows support. An example batch file is included in the mq_influx directory;
changes would be needed to the MQSC script to call it. The other monitor programs can be
supported with similar modifications.
* Added a "getting started" section to this README.

07 Nov 2016
* Added a collector that prints metrics in a simple JSON format.
See the [README](cmd/mq_json/README.md) for more details.
* Fixed bug where freespace metrics were showing as non-integer bytes, not percentages

14 Dec 2016
* Minor updates to this README for formatting
* Removed xxx_CURRENT_LENGTH definitions from cmqc

10 Jan 2017
* Added support for the MQCD and MQSCO structures to allow programmable client
connectivity, without requiring a CCDT. See the clientconn sample program
for an example of using the MQCD.
* Moved sample programs into subdirectory

15 Feb 2017
* API BREAKING CHANGE: The MQI verbs have been changed to return a single
error indicator instead of two separate values. See mqitest.go for
examples of how MQRC/MQCC codes can now be tested and extracted. This change
makes the MQI implementation a bit more natural for Go environments.

25 Mar 2017
* Added the metaPrefix option to the Prometheus monitor. This allows selection of non-default resources such as the MQ Bridge for Salesforce included in MQ 9.0.2.

18 May 2017
* Added the V9.0.3 constant definitions.
* Reinstated 64-bit structure "length" fields in 
cmqc.go after fixing a bug in the base product C source code generator.

## Health Warning

This package is provided as-is with no guarantees of support or updates. There are also no guarantees of compatibility
with any future versions of the package; the API is subject to change based on any feedback.

## Issues and Contributions

For feedback and issues relating specifically to this package, please use the [GitHub issue tracker](https://github.com/ibm-messaging/mq-golang/issues).

Contributions to this package can be accepted under the terms of the IBM Contributor License Agreement,
found in the file CLA.md of this repository. When submitting a pull request, you must include a statement stating
you accept the terms in CLA.md.

