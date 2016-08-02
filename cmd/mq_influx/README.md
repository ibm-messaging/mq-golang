# MQ Exporter for InfluxDB monitoring

This directory contains the code for a monitoring solution
that exports queue manager data to am InfluxDB data collection
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
Prometheus-based dashboard, also in this repository.
To use the dashboard,
create a data source in Grafana called "MQ Influx" that points at your
database server, and then import the JSON file.

## Building
* This github repository contains both the monitoring program and
the ibmmq package that links to the core MQ application interface. It
also contains the mqmetric package used as a common component for
supporting alternative database collection protocols.

* You also need access to the InfluxDB Go client interface.

  The command `go get -u github.com/influxdata/influxdb/client/v2` should pull
  down the client code and its dependencies.

* The error logger package may need to be explicitly downloaded

  On my system, I also had to forcibly download the logger package,
  using `go get -u github.com/Sirupsen/logrus`.

Run `go build -o <directory>/mq_influx cmd/mq_influx/*.go` to compile
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
the queue manager publications. Look at the mq_influx.sh script or config.go
to see how to provide these parameters.

In particular, if the database requires password authentication, then the password
is not provided as a command-line parameter, or read from the environment. It needs
to be given via a file named on the command line, which is then deleted
after being read. This indirection is done to make it easy to use the
shell "exec" function, keeping the process id of the real program available
to the queue manager's SERVICE definition.

The queue manager will usually generate its publications every 10 seconds. That is also
the default interval being used in the monitor program to read those publications.

## Configuring InfluxDB
No special configuration is required for InfluxDB. You may want to create an MQ-specific
database, and suitable userids to access that database.

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

The metrics stored in the Influx database are named after the
descriptions that you can see when running the amqsrua sample program, but with some
minor modifications to match a more useful style.
