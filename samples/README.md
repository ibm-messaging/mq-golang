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

Some trivial scripts run the sample programs in matching pairs:
* putget.sh  : Run amqsput and then use the generated MsgId to get the same message with amqsget
* pubsub.sh  : Start amqssub and then run the amqspub program immediately

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

### Publish/Subscribe testing
You will probably want to run `amqssub` JUST BEFORE running `amqspub` to ensure
there is something waiting to receive the publications when they are made. The
`pubsub.sh` script executes the two programs appropriately.

## More information
Comments in the programs explain what they are doing. For more detailed information about the
MQ API, the functions, structures, and constants, see the
[MQ Knowledge Center](https://www.ibm.com/support/knowledgecenter/en/SSFKSJ_9.1.0/com.ibm.mq.ref.dev.doc/q089590_.htm).

You can also find general MQ application development advice [here](https://www.ibm.com/support/knowledgecenter/en/SSFKSJ_9.1.0/com.ibm.mq.dev.doc/q022830_.htm).
Information about development for procedural programming languages such as C in that
documentation is most relevant for the interface exported by this Go package.
