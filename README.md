# mq-golang

This repository demonstrates how you can call IBM MQ from applications written in the Go language.

This repository previously contained programs that exported MQ statistics to
some monitoring packages. These have now been moved to a
[GitHub repository called mq-metric-samples](https://github.com/ibm-messaging/mq-metric-samples).

A minimum level of MQ V8 is required to build these packages. However, note that
the monitoring data published by the queue manager and exploited in the mqmetric package
is not available before MQ V9. A limited set of metrics can be monitored for MQ V8 instances by setting `ibmmq.usePublications=false`.

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
directory for the code is `/usr/lib/golang`. If you see an error similar to "ld: NULL not defined"
when building a program then it is likely you need to upgrade your compiler.


* Create a working directory. For example, ```mkdir $HOME/gowork```

* Set environment variables. Based on the previous lines,

```
  export GOROOT=/usr/lib/golang
  export GOPATH=$HOME/gowork
```

* On Linux, some versions of the compiler have required that you set environment variables to permit some compile/link flags. Recent versions of Go seem to effectively include this fix in the compiler so that the export is no longer necessary.

```
export CGO_LDFLAGS_ALLOW="-Wl,-rpath.*"
```

* Install the git client

### Windows

* Install the Go runtime and compiler. On Windows, the common directory is `c:\Go`
* Ensure you have a gcc-based compiler. The variant that now seems to be recommended for cgo is
the [tdm-gcc-64](https://jmeubank.github.io/tdm-gcc/download/) 64-bit compiler suite.
The default `gcc` compiler from Cygwin does not work because it tries to build a
Cygwin-enabled executable but the MQ libraries do not work in that model;
the `mingw` versions build Windows-native programs.
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
for the C compiler to recognise those directories. You may get messages from the compiler
saying that the default MQ directories cannot be found, but those warnings can be ignored.
The exact values for these environment variables will vary by platform, but follow the
corresponding CFLAGS/LDFLAGS values in `mqi.go`

For example,

```
   export MQ_INSTALLATION_PATH=/my/mq/dir  # This will also be set from the setmqenv command

   export CGO_CFLAGS="-I$MQ_INSTALLATION_PATH/inc"

   export CGO_LDFLAGS="-L$MQ_INSTALLATION_PATH/lib64 -Wl,-rpath,$MQ_INSTALLATION_PATH/lib64"
```

* Compile the `ibmmq` component:
*
  `go install ./src/github.com/ibm-messaging/mq-golang/ibmmq`

* If you plan to use monitoring functions, then compile the `mqmetric` component:

  `go install ./src/github.com/ibm-messaging/mq-golang/mqmetric`

* Sample programs can be compiled in this way

  `go build -o bin/mqitest ./src/github.com/ibm-messaging/mq-golang/samples/mqitest/*.go`

At this point, you should have a compiled copy of the program in `$GOPATH/bin`. See the
`samples` directory for more sample programs.

## Building in a container
The `buildSamples.sh` script in this directory can also be used to create a container which will
compile the samples and copy them to a local directory. If you use this approach, you do not need
to install a local copy of the compiler and associated toold, though you will still need a copy of
the MQ runtime libraries for wherever you execute the programs.

## Go Modules
The packages in this repository are now set up to be used as Go modules. See the `go.mod` file in
the root of the repository. This required a major version bump in the release stream.

Support for modules started to be introduced around Go 1.11 and has been firmed up in various
modification level updates in each of the compiler levels since then. The module changes for this
package were developed and tested with Go 1.13.6.

To use the MQ module in your application, your `go.mod` file contains

```
  require (
    github.com/ibm-messaging/mq-golang/v5 v5.0.0
  )
```

and your application code will include

```
  import ibmmq "github.com/ibm-messaging/mq-golang/v5/ibmmq"
```

If you have not moved to using modules in your application, you should continue using the older levels
of these packages. For example, you can continue to use `dep` with `Gopkg.toml` referring to

```
[[constraint]]
  name = "github.com/ibm-messaging/mq-golang"
  version = "4.1.4"
```

## Related Projects

These GitHub-hosted projects are related to or derived from this one. This is not a complete list. Please
let me know, via an issue, if you have another project that might be suitable for inclusion here.

| Repository                           | Description   |
|--------------------------------------|---------------|
|[ibm-messaging/mq-metric-samples](https://github.com/ibm-messaging/mq-metric-samples)| Extracts metrics for use in Prometheus, Influx<br>JSON consumers etc.|
|[ibm-messaging/mq-golang-jms20](https://github.com/ibm-messaging/mq-golang-jms20)   | JMS-style messaging interface for Go applications|
|[ibm-messaging/mq-container](https://github.com/ibm-messaging/mq-container)         | Building MQ into containers. Uses features from this package<br>for configuration and monitoring  |
|[felix-lessoer/qbeat](https://github.com/felix-lessoer/qbeat)                       | Extract monitoring and statstics from MQ for use in Elasticsearch|
|[ibm-messaging/mq-mqi-nodejs](https://github.com/ibm-messaging/mq-mqi-nodejs)       | A similar MQI interface for Node.js applications|

## Limitations

### Package 'ibmmq'
All regular MQI verbs are now available through the `ibmmq` package.

The only unimplemented area of MQI function is the use of Distribution Lists: they were
rarely used, and the Publish/Subscribe operations provide similar capability.

### Package 'mqmetric'
* There is currently a queue manager limitation which does not permit resource publications to
be made about queues whose name includes '/'. Attempting to monitor such a queue will result in a warning
logged by the mqmetric package.

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

© Copyright IBM Corporation 2016, 2020
