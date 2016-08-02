#!/bin/sh

# This is used to start the IBM MQ monitoring service for collectd
# via the 'Exec' interface

# The queue manager name comes in from the collectd configuration as
# the only command line parameter.
qMgr=$1

# Set the environment to ensure we pick up libmqm.so etc
# This assumes there is an MQ installation in the default location, even
# if it is not the one associated with the queue manager
. /opt/mqm/bin/setmqenv -m $qMgr -k

# A list of queues to be monitored is given here.
# It is a set of names or patterns ('*' only at the end, to match how MQ works),
# separated by commas. When no queues match a pattern, it is reported but
# is not fatal.
queues="APP.*,MYQ.*"

# An alternative is to have a file containing the patterns, and named
# via the ibmmq.monitoredQueuesFile option.

# And other parameters that may be needed
# See config.go for all recognised flags
interval="5"

ARGS="-ibmmq.queueManager=$qMgr"
ARGS="$ARGS -ibmmq.monitoredQueues=$queues"
ARGS="$ARGS -ibmmq.interval=$interval"
ARGS="$ARGS -log.level=error"

# Start via "exec" so the pid remains the same.
#
# The collectd system reads everything sent to stdout
# This program will send error log info to stderr; that is also
# picked up by collectd.
#
# Change this line to match wherever you have installed the MQ monitor program
exec /usr/local/bin/mqgo/mq_coll $ARGS
