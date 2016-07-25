# mq-golang
This repository demonstrates how you can call IBM MQ from applications written in the Go language.

The repository also includes a program to export MQ statistics to a
Prometheus server.

## Description

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

To use the package, you will need to be able to build Go applications, and have a copy of MQ installed to
build against. It uses cgo to access the MQI C structures and definitions. It assumes that MQ has been
installed in the default location on a Linux platform (/opt/mqm) but you can easily change the
cgo directives in the source files if necessary.

## Limitations

Not all of the MQI verbs are available through this interface. This initial implementation
concentrates on the core API calls needed to put and get messages. Currently unavailable
verbs include:
* MQCONNX
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

## Health Warning

This package is provided as-is with no guarantees of support or updates. There are also no guarantees of compatibility
with any future versions of the package; the API is subject to change based on any feedback.

##Issues and Contributions

For feedback and issues relating specifically to this package, please use the [GitHub issue tracker](https://github.com/ibm-messaging/mq-golang/issues).

Contributions to this package can be accepted under the terms of the IBM Contributor License Agreement,
found in the file CLA.md of this repository. When submitting a pull request, you must include a statement stating
you accept the terms in CLA.md.

