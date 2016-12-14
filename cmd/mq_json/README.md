# MQ Exporter for JSON-based monitoring

This directory contains the code for a monitoring solution
that prints queue manager data in JSON format.
It also contains configuration files to run the monitor program

The monitor collects metrics published by an MQ V9 queue manager
or the MQ appliance. The monitor program prints
these metrics to stdout.

You can see data such as disk or CPU usage, queue depths, and MQI call
counts.

## Building
* This github repository contains both the monitoring program and
the ibmmq package that links to the core MQ application interface. It
also contains the mqmetric package used as a common component for
supporting alternative database collection protocols.

* Get the error logger package used by all of these monitors
using `go get -u github.com/Sirupsen/logrus`.

Run `go build -o <directory>/mq_json cmd/mq_json/*.go` to compile
the program and put it to a specific directory.

## Configuring MQ
It is convenient to run the monitor program as a queue manager service.
This directory contains an MQSC script to define the service. In fact, the
service definition points at a simple script which sets up any
necessary environment and builds the command line parameters for the
real monitor program. As the last line of the script is "exec", the
process id of the script is inherited by the monitor program, and the
queue manager can then check on the status, and can drive a suitable
`STOP SERVICE` operation during queue manager shutdown.

Edit the MQSC script to point at appropriate directories
where the program exists, and where you want to put stdout/stderr.
Ensure that the ID running the queue manager has permission to access
the programs and output files.

Since the output from the monitor is always sent to stdout, you will
probably want to modify the script to pipe the output to a processing
program that works with JSON data, or to a program that automatically
creates and manages multiple log files.

The monitor always collects all of the available queue manager-wide metrics.
It can also be configured to collect statistics for specific sets of queues.
The sets of queues can be given either directly on the command line with the
`-ibmmq.monitoredQueues` flag, or put into a separate file which is also
named on the command line, with the `ibmmq.monitoredQueuesFile` flag. An
example is included in the startup shell script.

At each collection interval, a JSON object is printed, consisting of
a timestamp followed by an array of "points" which contain the
metric and the resource it refers to.

For example,
    {
       "collectionTime" : {
          "timeStamp" : "2016-11-07-T15:00:55Z"
          "epoch" : 1478527255
       },
       "points" : [
          { "queueManager" : "QM1", "ramTotalBytes" : 15515735206 },
          { "queueManager" : "QM1", "userCpuTimePercentage" : 1.33 }
       ]
    }


## Metrics
Once the monitor program has been started,
you will see metrics being available.
More information on the metrics collected through the publish/subscribe
interface can be found in the [MQ KnowledgeCenter]
(https://www.ibm.com/support/knowledgecenter/SSFKSJ_9.0.0/com.ibm.mq.mon.doc/mo00013_.htm)
with further description in [an MQDev blog entry]
(https://www.ibm.com/developerworks/community/blogs/messaging/entry/Statistics_published_to_the_system_topic_in_MQ_v9?lang=en)

The metrics printed are named after the
descriptions that you can see when running the amqsrua sample program, but with some
minor modifications to match a more useful style.
