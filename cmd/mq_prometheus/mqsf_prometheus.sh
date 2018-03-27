#!/bin/sh

# This is used to start the IBM MQ monitoring service for Prometheus
# to collect data from the MQ Bridge for Salesforce.

# The queue manager name comes in from the service definition as the
# only command line parameter
qMgr=$1

# Set the environment to ensure we pick up libmqm.so etc
. /opt/mqm/bin/setmqenv -m $qMgr -k

# A list of topics to be monitored is given here. Can use specific topics
# or MQ wildcards such as "#". We are still using a "queues" command line
# parameter, and ought to change it, but this works.
queues="#,/topic/PT1,/topic/PT2,/event/PE2__e,/event/PE1__e"

# See config.go for all recognised flags

# Start via "exec" so the pid remains the same. The queue manager can
# then check the existence of the service and use the MQ_SERVER_PID value
# to kill it on shutdown.
# Need to specify an HTTP port that is not currently used. The default 9157 may
# be used by the collector accessing queue manager resource statistics.
exec /usr/local/bin/mqgo/mq_prometheus -ibmmq.queueManager=$qMgr -ibmmq.monitoredQueues="$queues" -log.level=error -metaPrefix="\$SYS/Application/runmqsfb" -ibmmq.httpListenPort=9158 -namespace=ibmmqsf
