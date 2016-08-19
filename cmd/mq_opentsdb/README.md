# MQ Exporter for OpenTSDB monitoring

This directory contains the code for a monitoring solution
that exports queue manager data to an OpenTSDB data collection
system. It also contains configuration files to run the monitor program

The monitor collects metrics published by an MQ V9 queue manager
or the MQ appliance. The monitor program pushes
those metrics into the database, over an HTTP connection, where
they can then be queried directly or used by other packages
such as Grafana.

You can see data such as disk or CPU usage, queue depths, and MQI call
counts.

An example Grafana dashboard is included, to show how queries might
be constructed. The data shown is the same as in the corresponding
Prometheus and InfluxDB-based dashboards, also in this repository.
To use the dashboard,
create a data source in Grafana called "MQ OpenTSDB" that points at your
database server, and then import the JSON file.

## Building
* This github repository contains both the monitoring program and
the ibmmq package that links to the core MQ application interface. It
also contains the mqmetric package used as a common component for
supporting alternative database collection protocols.

* The error logger package may need to be explicitly downloaded

  On my system, I also had to forcibly download the logger package,
  using `go get -u github.com/Sirupsen/logrus`.

Run `go build -o <directory>/mq_opentsdb cmd/mq_opentsdb/*.go` to compile
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
the queue manager publications. Look at the mq_opentsdb.sh script or config.go
to see how to provide these parameters.

In particular, if the database requires password authentication, then the password
is not provided as a command-line parameter, or read from the environment. It needs
to be given to the command via a file; while the shell script has a hardcoded password
which is sent to the real command, you may prefer to have an alternative mechanism to
discover that password.

The queue manager will usually generate its publications every 10 seconds. That is also
the default interval being used in the monitor program to read those publications.

## Configuring OpenTSDB
No special configuration is required for the database.

## Metrics
Once the monitor program has been started,
you will see metrics being available.
console. Two series of metrics are collected, "queue" and "qmgr". All of the queue
manager values are given a tag of the queue manager name; all of the queue-based values
are tagged with both the queue and queue manager names. The queue name is actually
given by the "object" tag; that is to simplify things if the queue manager
ever generates this kind of statistics for other object types such as topics.

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
