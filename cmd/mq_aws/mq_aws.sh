#!/bin/sh

# This is used to start the IBM MQ monitoring service for AWS

# The queue manager name comes in from the service definition as the
# only command line parameter
qMgr=$1

# Set the environment to ensure we pick up libmqm.so etc
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
interval="30"

ARGS="-ibmmq.queueManager=$qMgr"
ARGS="$ARGS -ibmmq.monitoredQueues=$queues"
ARGS="$ARGS -log.level=error"
ARGS="$ARGS -ibmmq.maxPoints=20"
ARGS="$ARGS -ibmmq.awsregion=us-west-2"

# Start via "exec" so the pid remains the same. The queue manager can
# then check the existence of the service and use the MQ_SERVER_PID value
# to kill it on shutdown.

# Credentials are expected to be provided in the $HOME/.aws/credentials file
exec /usr/local/bin/mqgo/mq_aws  $ARGS
