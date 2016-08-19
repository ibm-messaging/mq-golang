# MQ Exporter for Amazon CloudWatch monitoring

This directory contains the code for a monitoring solution
that exports queue manager data to a CloudWatch data collection
system. It also contains configuration files to run the monitor program

The monitor collects metrics published by an MQ V9 queue manager
or the MQ appliance. The monitor program pushes
those metrics into the database, where
they can then be queried directly or used by other packages
such as Grafana.

You can see data such as disk or CPU usage, queue depths, and MQI call
counts.

An example Grafana dashboard is included, to show how queries might
be constructed. The data shown is the same as in the corresponding
Prometheus-based dashboard, also in this repository.
To use the dashboard,
create a data source in Grafana called "MQ CloudWatch" that points at your
AWS server, and then import the JSON file. Grafana does not make it as easy
to handle wildcard queries to CloudWatch, so this dashboard explicitly names
the queues to monitor. There may be better solutions using templates, but
that starts to get more complex than I want to show in this example.

## Building
* This github repository contains both the monitoring program and
the ibmmq package that links to the core MQ application interface. It
also contains the mqmetric package used as a common component for
supporting alternative database collection protocols.

* You also need access to the AWS Go client interfaces.

  The command `go get -u github.com/aws/aws-sdk-go/service` should pull
  down the client code and its dependencies.

* The error logger package may need to be explicitly downloaded

  On my system, I also had to forcibly download the logger package,
  using `go get -u github.com/Sirupsen/logrus`.

Run `go build -o <directory>/mq_aws cmd/mq_aws/*.go` to compile
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

Edit the MQSC script and the shell script to point at appropriate directories
where the program exists, and where you want to put stdout/stderr.
Ensure that the ID running the queue manager has permission to access
the programs and output files.

The monitor always collects all of the available queue manager-wide metrics.
It can also be configured to collect statistics for specific sets of queues.
The sets of queues can be given either directly on the command line with the
`-ibmmq.monitoredQueues` flag, or put into a separate file which is also
named on the command line, with the `ibmmq.monitoredQueuesFile` flag. An
example is included in the startup shell script.

Note that **for now**, the queue patterns are expanded only at startup
of the monitor program. If you want to change the patterns, or new
queues are defined that match an existing pattern, the monitor must be
restarted with a `STOP SERVICE` and `START SERVICE` pair of commands.

There are a number of required parameters to configure the service, including
the queue manager name, how to reach a database, and the frequency of reading
the queue manager publications. Look at the mq_aws.sh script or config.go
to see how to provide these parameters.

For authentication, the
program is expecting to pick up the keys from $HOME/.aws/credentials. That
file is not explicitly referenced in this code; it's used automatically by
the AWS toolkit. You may need to provide a region
name as a command-line option if it is not mentioned in the credentials
file.

The queue manager will usually generate its publications every 10 seconds. However, the
default interval being used by this monitor program is set to 60 seconds for reading those
publications because of the slower rate that CloudWatch monitoring usually works at, and
to reduce the number of calls to CloudWatch which do contribute to AWS charges.

## Configuring CloudWatch
No special configuration is required for CloudWatch. You will see the MQ
data appear in the Custom Metrics options, with a default namespace of "IBM/MQ".
There are two sets of metrics in that namespace, one for the queue manager
(with the qmgr filter) and one for the queues (with the object,qmgr filter).

## Metrics
Once the monitor program has been started,
you will see metrics being available.
console. Two series of metrics are collected, "queue" and "qmgr". All of the queue
manager values are given a tag of the queue manager name; all of the queue-based values
are tagged with both the object and queue manager names.

The example Grafana dashboard shows how queries can be constructed to extract data
about specific queues or the queue manager.

More information on the metrics collected through the publish/subscribe
interface can be found in the [MQ KnowledgeCenter]
(https://www.ibm.com/support/knowledgecenter/SSFKSJ_9.0.0/com.ibm.mq.mon.doc/mo00013_.htm)
with further description in [an MQDev blog entry]
(https://www.ibm.com/developerworks/community/blogs/messaging/entry/Statistics_published_to_the_system_topic_in_MQ_v9?lang=en)

The metrics stored in the database are named after the
descriptions that you can see when running the amqsrua sample program, but with some
minor modifications to match a more useful style.
