# mq-golang

This repository demonstrates how you can call IBM MQ from applications written in the Go language.

The repository originally also contained programs that exported MQ statistics to
monitoring systems. These programs have been moved to a GitHub repository called [mq-metric-samples](https://github.com/ibm-messaging/mq-metric-samples).

A minimum level of MQ V8 is required to build these packages, although it should be possible to connect as a client to even older versions of queue manager.

## Health Warning

This package is provided as-is with no guarantees of support or updates. You cannot use
IBM formal support channels (Cases/PMRs) for assistance with material in this repository.

There are also no guarantees of compatibility with any future versions of the package; the API
is subject to change based on any feedback. Versioned releases are made in this repository
to assist with using stable APIs. Future versions will follow semver guidance so that breaking changes
will only be done with a new major version number on the module.

See the [DEPRECATIONS](DEPRECATIONS.md) file for any planned changes to the API.

## MQI Description

The `ibmmq` directory contains a Go package, exposing an MQI-like interface.

The intention is to give an API that is more natural for Go programmers than the
common procedural MQI. For example, fixed length string arrays from the C API such
as MQCHAR48 are represented by the native Go string type. Conversion between these
types is handled within the `ibmmq` package itself, removing the need for Go programmers
to know about it.

Sample programs are provided to demonstrate various features of using the MQI. See the
README in the `samples` directory for more information about those programs. Detailed information about the MQI and application design can be found in the MQ product
documentation. Although that doesn't mention Go as a language, the principles for all
applications apply.

The `mqmetric` directory contains functions to help monitoring programs access MQ status and statistics. This package is not needed for general application programs.

## Using the package

To use code in this repository, you will need to be able to build Go applications, and
have a copy of MQ installed to build against. It uses `cgo` to access the MQI C
structures and definitions. It assumes that MQ has been installed in the default
location (on a Linux platform this would be `/opt/mqm`) but this can be changed
with environment variables if necessary.

Windows compatibility is also included. Current versions of the Go compiler
permit standard Windows paths (eg including spaces) so the CGO directives
can point at the normal MQ install path.

## Getting started

If you are unfamiliar with Go, the following steps can help create a working environment
with source code in a suitable tree. Initial setup tends to be platform-specific,
but subsequent steps are independent of the platform.

### MQ Client SDK
The MQ Client SDK for C programs is required in order to compile and run Go programs. You may have this from an MQ Client installation image (eg rpm, dep for Linux; msi for
Windows). 

For Linux x64 and Windows systems, you may also choose to use the 
MQ Redistributable Client package which is a simple zip/tar file that does not need
any privileges to install:

* Download [IBM MQ redistributable client](https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqdev/redist)
* Unpack archive to fixed directory. E.g. `c:\IBM-MQC-Redist-Win64`  or `/opt/mqm`.

### Linux

* Install the Go runtime and compiler. On Linux, the packaging may vary but a typical
directory for the code is `/usr/lib/golang`. 
* Create a working directory. For example, ```mkdir $HOME/gowork```
* Install the git client and the gcc C compiler

### Windows

* Install the Go runtime and compiler. On Windows, the common directory is `c:\Go`
* Ensure you have a gcc-based compiler. The variant that seems to be recommended for cgo is
the [tdm-gcc-64](https://jmeubank.github.io/tdm-gcc/download/) 64-bit compiler suite.
The default `gcc` compiler from Cygwin does not work because it tries to build a
Cygwin-enabled executable but the MQ libraries do not work in that model;
the `mingw` versions build Windows-native programs.
* Create a working directory. For example, `mkdir c:\Gowork`
* Set an environment variable for the compiler
```
set CC=x86_64-w64-mingw32-gcc.exe
```

### Common

* Make sure your PATH includes routes to the Go compiler, the Git client, and the C compiler.
* Change to the directory you created earlier.
* Use git to get a copy of the MQ components into a new directory in the workspace.

  `git clone git@github.com:ibm-messaging/mq-golang.git src/github.com/ibm-messaging/mq-golang`

* If you have not installed MQ libraries into the default location, then set environment variables
for the C compiler to recognise those directories. You may then get messages from the compiler
saying that the default MQ directories cannot be found, but those warnings can be ignored.
The exact values for these environment variables will vary by platform, but follow the
corresponding CFLAGS/LDFLAGS values in `mqi.go`

For example, on Linux:

```
   export MQ_INSTALLATION_PATH=/my/mq/dir  # This will also be set from the setmqenv command
   export CGO_CFLAGS="-I$MQ_INSTALLATION_PATH/inc"
   export CGO_LDFLAGS="-L$MQ_INSTALLATION_PATH/lib64 -Wl,-rpath,$MQ_INSTALLATION_PATH/lib64"
```

Or on Windows:

```
  set CGO_CFLAGS=-Ic:\IBM-MQC-Redist-Win64\tools\c\include -D_WIN64
  set CGO_LDFLAGS=-L c:\IBM-MQC-Redist-Win64\bin64 -lmqm
```

* Sample programs can be compiled directly:

 ```
 cd src/github.com/ibm-messaging/mq-golang/samples
 go build -o /tmp/mqitest mqitest/*.go
 ```

At this point, you should have a compiled copy of the program in `/tmp`. See the
`samples` directory for more sample programs.


## Building in a container
The `buildSamples.sh` script in this directory can also be used to create a container which will
install the MQ Client SDK, compile the samples and copy them to a local directory. If you use this approach, you do not need
to install a local copy of the compiler and associated tools, though you will still need a copy of
the MQ C client runtime libraries for wherever you execute the programs.

## Go Modules
The packages in this repository are set up to be used as Go modules. See the `go.mod` file in
the root of the repository.

Support for modules started to be introduced around Go 1.11 and has been firmed up in various
modification level updates in each of the compiler levels since then. It is now recommended to 
use at least version 1.17 of the compiler.

Use of modules means that packages do not need to be independently compiled or
installed. Environment variables such as `GOROOT` and `GOPATH` that were previously required are
now redundant in module mode.

To use the MQ module in your application, your `go.mod` file contains

```
  require (
    github.com/ibm-messaging/mq-golang/v5 v5.x.y
  )
```

and your application code includes

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
Those older versions are not maintained, so it is strongly recommended you do move to using modules.

## Related Projects

These GitHub-hosted projects are related to or derived from this one. This is not a complete list. Please
let me know, via an issue, if you have another project that might be suitable for inclusion here.

| Repository                           | Description   |
|--------------------------------------|---------------|
|[ibm-messaging/mq-metric-samples](https://github.com/ibm-messaging/mq-metric-samples)| Extracts metrics for use in Prometheus, Influx<br>JSON consumers etc.|
|[ibm-messaging/mq-golang-jms20](https://github.com/ibm-messaging/mq-golang-jms20)   | JMS-style messaging interface for Go applications|
|[ibm-messaging/mq-container](https://github.com/ibm-messaging/mq-container)         | Building MQ into containers. Uses features from this package<br>for configuration and monitoring  |
|[felix-lessoer/qbeat](https://github.com/felix-lessoer/qbeat)                       | Extract monitoring and statistics from MQ for use in Elasticsearch|
|[ibm-messaging/mq-mqi-nodejs](https://github.com/ibm-messaging/mq-mqi-nodejs)       | A similar MQI interface for Node.js applications|

## Limitations

### Package 'ibmmq'
* All regular MQI verbs are available through the `ibmmq` package.
* The only unimplemented area of MQI function is the use of Distribution Lists: they were
rarely used, and the Publish/Subscribe operations provide similar capability.
* Go is not supported for writing MQ Exits, so structures and methods for those features
are not included.

### Package 'mqmetric'
* The monitoring data published by the queue manager and exploited in the mqmetric package is not available before MQ V9. A limited set of metrics can be monitored for MQ V8 instances by setting the `ConnectionConfig.UsePublications` configuration option to `false`.
* There is currently a queue manager limitation which does not permit resource publications to be made about queues whose name includes '/'. Attempting to monitor such a queue will result in a warning logged by the mqmetric package.

## History

See [CHANGELOG](CHANGELOG.md) in this directory.

## Issues and Contributions

Feedback on the utility of this package, thoughts about whether it should be changed
or extended are welcomed.

For feedback and issues relating specifically to this package, please use
the [GitHub issue tracker](https://github.com/ibm-messaging/mq-golang/issues).

Contributions to this package can be accepted under the terms of the Developer's Certificate
of Origin, found in the [DCO file](DCO1.1.txt) of this repository. When
submitting a pull request, you must include a statement stating you accept the terms
in the DCO.


## Copyright

© Copyright IBM Corporation 2016, 2023
