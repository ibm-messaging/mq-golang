# Sample program information
Files in this directory are samples to demonstrate use of the Go interface to IBM MQ.

You can run them individually using `go run <program>.go` with any additional
required or optional parameters on the command line. Look at the source code to see
which configuration values can be changed.

Make sure you first read the README in the root of this repository to set up an environment
where Go programs can be compiled, and how the packages refer to the MQ interfaces.

The `buildSamples.sh` script in the root directory can be used to create a container to
compile the samples and copy them to a local directory.

## Default values
Where needed for the sample programs:

* the default queue manager is "QM1"
* the default queue is "DEV.QUEUE.1"
* the default topic is based on "DEV.BASE.TOPIC" (topic string is under dev/... tree)

## Description of sample programs
Current samples in this directory include

* amqsput.go : Put a single message to a queue
* amqsget.go : Get all the messages from a queue. Optionally get a specific message by its id
* amqspub.go : Publish to a topic
* amqssub.go : Subscribe to a topic and receive publications
* amqsconn.go: How to programmatically connect as an MQ client to a remote queue manager.
Allow use of a userid/password for authentication. There are no default values for this sample.
* amqsprop.go: Set and extract message properties
* amqsinq.go : Demonstrate the Inq API for inquiring about object attributes
* amqsset.go : Demonstrate how to set attributes of an MQ object using the MQSET verb
* amqscb.go  : Demonstrate use of the CALLBACK capability for asynchronous consumption of messages
* amqsbo.go  : Show how to deal with poison messages by putting them to a configured backout queue
* amqsdlh.go : Putting a message to a DLQ with a dead-letter header
* amqspcf.go : Demonstrate use of the PCF functions to create a command and parse a response
* amqsjwt.go : Demonstrate retrieving a JWT token from a server and using that to connect

Some trivial scripts run the sample programs in matching pairs:
* putget.sh  : Run amqsput and then use the generated MsgId to get the same message with amqsget
* pubsub.sh  : Start amqssub and then run the amqspub program immediately

Building a container:
* runSample.sh           : Drives the process to get `amqsput` into the container
* runSample.bud.sh       : Drives the process to get `amqsput` into the container using podman/buildah as an alternative approach
* runSample.*.Dockerfile : Instructions to create containers with runtime dependencies
* runSample.gomod        : Copied into the container as `go.mod`
Two variants of the Dockerfile are provided. Set the `FROM` environment variable to "UBI"
to use Red Hat Universal Base Images as the starting points for building and runtime; set it to
"DEP" to use a Ubuntu/Debian combination.

The `mqitest` sample program in its own subdirectory is a more general demonstration
of many of the features available from the MQI rather than focussed on a specific
aspect.

## Running the programs
Apart from the `amqsconn.go` program, the other samples are designed to either connect
to a local queue manager (on the same machine) or for the client configuration to be
provided externally such as by the MQSERVER environment variable or the
MQ Client Channel Definition Table (CCDT) file. The MQ_CONNECT_TYPE environment
variable can be used to force client connections to be made, even if you have
installed the full server product; that variable is not needed if you have
only installed the MQ client libraries.

For example

```
  export MQSERVER="SYSTEM.DEF.SVRCONN/TCP/localhost(1414)"
  export MQ_CONNECT_TYPE=CLIENT
  go run amqsput.go DEV.QUEUE.1 QM1
```

The amqsput.go program also allows the queue and queue manager names to
be provided by environment variables, to show another configuration
mechanism. That approach will often be used in container deployments,
and is demonstrated in the runSample set of files.

### Publish/Subscribe testing
You will probably want to run `amqssub` JUST BEFORE running `amqspub` to ensure
there is something waiting to receive the publications when they are made. The
`pubsub.sh` script executes the two programs appropriately.

## Building a container for running samples
There is an set of files in here that will show how to create a container that runs
the `amqsput` program. The `runSample*.sh` scripts drive the process. It will try to
connect to a queue manager running on the host machine.

The process is split into two pieces - the first is used to compile the program, and
the second creates a (hopefully) smaller container with just the components needed
to run the program.

## More information
Comments in the programs explain what they are doing. For more detailed information about the MQ API, the functions,
structures, and constants, see the [MQ Documentation](https://www.ibm.com/docs/en/ibm-mq/latest).

You can also find general MQ application development advice
[here](https://www.ibm.com/docs/en/ibm-mq/latest?topic=mq-developing-applications). Information about development for
procedural programming languages such as C in that documentation is most relevant for the interface exported by this
package.
