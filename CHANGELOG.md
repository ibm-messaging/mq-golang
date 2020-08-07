# Changelog
Newest updates are at the top of this file.

## Aug 07 2020 - v5.1.1
* ibmmq - Fix STS structure (#146)
* Add flag for Windows build that seems no longer to be automatically set by cgo

## Jul 23 2020 - v5.1.0
* Update for MQ 9.2.0
* mqmetric - Add explicit client configuration options
* mqmetric - Add counter of how many resource publicatins read per scrape

## June 1 2020 - v5.0.0
* Migration for Go modules (requires new major number) (#138)
* ibmmq - Add all string mapping functions from cmqstrc (#142)
* ibmmq - Add AIX platform header
* mqmetric - Permit selection of which statistics to gather for STATQ (ibm-messaging/mq-metric-samples#34)
* mqmetric - Do not try to subscribe to application resource statistics (STATAPP) for now
* mqmetric - Add QFile usage status available from MQ 9.1.5

## Apr 02 2020 - v4.1.4
* Update for MQ 9.1.5
* ibmmq - Add message and header compression for MQCD (#137)
* ibmmq - Set endianness just once (#140)
* mqmetric - Add better diagnostics when running out of object handles
* mqmetric - Make sure strings don't include embedded nulls

## January 09 2020 - v4.1.3
* mqmetric - Discovery of shared queues (ibm-messaging/mq-metric-samples#26)
* mqmetric - Add DESCR attribute from queues and channels to permit labelling in metrics (ibm-messaging/mq-metric-samples#16)

## December 05 2019 - v4.1.2
* Update for MQ 9.1.4 - No new base API function introduced
* Add amqsgbr sample for browse option
* ibmmq - Add qmgr variant of the CB function for event handlers (#128)
* mqmetric - Add MaxChannels/MaxActiveChannels for z/OS (#129)
* mqmetric - Add MaxInst/MaxInst for SVRCONN channels (ibm-messaging/mq-metric-samples#21)

## October 7 2019 - v4.1.1
* ibmmq - Enable use of Context in MQPMO structure (#122)
* ibmmq - Remove unusable fields referring to Distribution List structures

## August 20 2019 - v4.1.0
* Update Docker build scripts for newer Go compiler level
* mqmetric - Issue warning if trying to monitor queues with names containing '/'

## August 2 2019 - unpublished
* ibmmq - Add new verb GetSlice to mirror Get() but which returns ready-sized buffer (#110)
  * See updated sample amqsget.go
* Some comment tidying up. Make CMQC constants constant.

## July 30 2019 - v4.0.10
* ibmmq - Add error checking to some structure fields (#111)

## July 22 2019 - v4.0.9
* mqmetric - Support RESET QSTATS on z/OS queue manager
* mqmetric - Add a Logger class to enable debug output
* mqmetric - Improve some error reports

## July 11 2019 - v4.0.8
* Update for MQ 9.1.3 - No new API function introduced
* mqmetric - Fix leak in subscriptions after rediscovery
* mqmetric - Add USAGE as a queue label for selection by xmitq

## June 25 2019 - v4.0.7
* mqmetric - Allow exclusion patterns for queue names (but not other object types)
  * Use "!" as prefix to a simple pattern in the list of monitored queues
  * For example, "APP.*,S*,!SYSTEM.*"
* mqmetric - Enable re-expansion of monitored queue wildcards while still monitoring
  * See Prometheus monitor sample for configuration
* mqmetric - Added batch size and xmitq time averages to channel metrics
* mqmetric - Enable use of z/OS DISPLAY USAGE for pageset/bufferpool data

## May 31 2019 - v4.0.6
* mqmetric - Allow limited monitoring of V8 Distributed platforms
  * Set `ibmmq.usePublications` to *false* to enable in monitor programs #104
* mqmetric - Added queue_attribute_max_depth to permit %full calculation
  * Set `ibmmq.useStatus` to *true* to enable in monitor programs #105
* samples - Correct use of the new form of the Inq() verb

## April 23 2019
* Fixed memory leak in InqMP
* mqmetric - Added ability to set a timezone offset
* mqmetric - Added fields from SBSTATUS

## April 03 2019
* mqmetric - Added last put/get time metric for queues
* mqmetric - Added last msg time metric for channels
* mqmetric - Added fields from QMSTATUS and TPSTATUS

## April 1 2019
* Added scripts to compile samples inside a container

## March 26 2019 - v4.0.2
* BREAKING API: Add hConn to callback function
* Callbacks not setting hConn correctly (#93)

## March 20 2019 - v4.0.0
* Update for MQ 9.1.2 - ApplName now settable during Connect
* BREAKING API: deprecated Inq()/MQINQ implementation replaced.
* Fixes to callback functions for EVENT processing
* mqmetric - Improve handling of z/OS channel status where multiple instances of the same name
* mqmetric - More accurate testing of model queue default maxdepth for status replies
* mqmetric - Was ignoring an error in subscription processing

## January 24 2019
* Deal with callback functions being called unexpectedly (#75)

## January 2019
* mqmetric - Add some configuration validation
* mqmetric - Make it possible to use CAPEXPRY for statistics subscriptions

## December 2018 - v3.3.0
* All relevant API calls now automatically set FAIL_IF_QUIESCING
* Samples updated to use "defer" instead of just suggesting it
* Add support for MQCB/MQCTL callback functions
* Add support for MQBEGIN transaction management
* Add Dead Letter Header parser

## November 2018 - v3.2.0
* Added GetPlatform to mqmetric so it can be used as a label/tag in collectors
* Added sample programs demonstrating specific operations such as put/get of message
* Fixed conversion of some C strings into Go strings
* Update MQI header files for MQ V9.1.1 and give more platform variations
* Add support for MQSTAT and MQSUBRQ functions
* Add support and sample for Message Property functions
* Add InqMap as alternative (simpler) MQINQ operation. Inq() should be considered deprecated
* Add support for MQSET function
* Add discovery of translated versions of the mqmetric descriptions

## November 2018 - v3.1.0
* Added functions to mqmetric to issue DISPLAY QSTATUS for additional stats
* Added z/OS capability for minimal status

## October 2018
* Allow compilation against MQ v8

## October 2018 - v3.0.0
* Added functions to the mqmetric package to assist with collecting channel status
* Better handle truncated messages when listing the queues that match a pattern

## October 2018
* Corrected heuristic for generating metric names

## August 2018
* Added V9.1 constant definitions
* Updated build comments

## July 2018 - v2.0.0
* Corrected package imports
* Formatted go code with `go fmt`
* Rearranged this file
* Removed logging from golang package `mqmetric`
* Moved samples to a separate repository
* Added build scripts for `ibmmq` and `mqmetric` packages and `ibmmq` samples
* Added unit tests for `ibmmq` and `mqmetric` packages

## March 2018 - v1.0.0
* Added V9.0.5 constant definitions
* Changed #cgo directives for Windows now the compiler supports standard path names
* Added mechanism to set MQ userid and password for Prometheus monitor
* Released v1.0.0 of this repository for use with golang dependency management tools

## October 2017
* Added V9.0.4 constant definitions - now generated from original MQ source code
* Added MQSC script to show how to redefine event queues for pub/sub
* Prometheus collector has new parameter to override the first component of the metric name
* Prometheus collector can now process channel-level statistics

## 18 May 2017
* Added the V9.0.3 constant definitions.
* Reinstated 64-bit structure "length" fields in cmqc.go after fixing a bug in the base product C source code generator.

## 25 Mar 2017
* Added the metaPrefix option to the Prometheus monitor. This allows selection of non-default resources such as the MQ Bridge for Salesforce included in MQ 9.0.2.

## 15 Feb 2017
* API BREAKING CHANGE: The MQI verbs have been changed to return a single error indicator instead of two separate values. See mqitest.go for examples of how MQRC/MQCC codes can now be tested and extracted. This change makes the MQI implementation a bit more natural for Go environments.

## 10 Jan 2017
* Added support for the MQCD and MQSCO structures to allow programmable client connectivity, without requiring a CCDT. See the clientconn sample program for an example of using the MQCD.
* Moved sample programs into subdirectory

## 14 Dec 2016
* Minor updates to this README for formatting
* Removed xxx_CURRENT_LENGTH definitions from cmqc

## 07 Nov 2016
* Added a collector that prints metrics in a simple JSON format. See the [README](cmd/mq_json/README.md) for more details.
* Fixed bug where freespace metrics were showing as non-integer bytes, not percentages

## 17 Oct 2016
* Added some Windows support. An example batch file is included in the mq_influx directory; changes would be needed to the MQSC script to call it. The other monitor programs can be supported with similar modifications.
* Added a "getting started" section to this README.

## 23 Aug 2016
* Added a collector for Amazon AWS CloudWatch monitoring. See the [README](cmd/mq_aws/README.md) for more details.

## 12 Aug 2016
* Added a OpenTSDB monitor. See the [README](cmd/mq_opentsdb/README.md) for more details.
* Added a Collectd monitor. See the [README](cmd/mq_coll/README.md) for more details.
* Added MQI MQCNO/MQCSP structures to support client connections and password authentication with MQCONNX.
* Allow client-mode connections from the monitor programs
* Added Grafana dashboards for the different monitors to show how to query them
* Changed database password mechanism so that "exec" maintains the PID for MQ services

## 04 Aug 2016
* Added a monitor command for exporting MQ data to InfluxDB. See the [README](cmd/mq_influx/README.md) for more details
* Restructured the monitoring code to put common material in the mqmetric package, called from the Influx and Prometheus monitors.

## 25 Jul 2016
* Added functions to handle basic PCF creation and parsing
* Added a monitor command for exporting MQ V9 queue manager data to Prometheus. See the [README](cmd/mq_prometheus/README.md) for more details

## 18 Jul 2016
* Changed structures so that most applications will not need to use cgo to imbed the MQ C headers
  * Go programs will now use int32 where C programs use MQLONG
  * Use of message handles, distribution lists require cgo for now
* Package ibmmq now includes the numeric #defines as a Go file, cmqc.go, for easier use
* Removed "src/" prefix from tree in github repo
* Removed need for buffer length parm on Put/Put1
* Updated comments
* Added MQINQ
* Added MQItoString function for some maps of values to constant names

## 08 Jul 2016
* Initial release
