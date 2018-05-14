package main

/*
  Copyright (c) IBM Corporation 2016

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific

   Contributors:
     Mark Taylor - Initial Contribution
*/

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/ibm-messaging/mq-golang/mqmetric"
)

type mqExporterConfig struct {
	qMgrName            string
	replyQ              string
	monitoredQueues     string
	monitoredQueuesFile string

	// TODO: enable these
	monitorChannelStatistics bool
	statisticsQueueName      string

	metaPrefix string

	cc mqmetric.ConnectionConfig

	httpListenPort string
	httpMetricPath string
	logLevel       string
	namespace      string
}

const (
	defaultPort      = "9157" // reserved in the prometheus wiki
	defaultNamespace = "ibmmq"
)

var config mqExporterConfig

/*
initConfig parses the command line parameters. Note that the logging
package requires flag.Parse to be called before we can do things like
info/error logging

The default IP port for this monitor is registered with prometheus so
does not have to be provided.
*/
func initConfig() {

	flag.StringVar(&config.qMgrName, "ibmmq.queueManager", "", "Queue Manager name")
	flag.StringVar(&config.replyQ, "ibmmq.replyQueue", "SYSTEM.DEFAULT.MODEL.QUEUE", "Reply Queue to collect data")
	flag.StringVar(&config.monitoredQueues, "ibmmq.monitoredQueues", "", "Patterns of queues to monitor")
	flag.StringVar(&config.monitoredQueuesFile, "ibmmq.monitoredQueuesFile", "", "File with patterns of queues to monitor")
	flag.StringVar(&config.metaPrefix, "metaPrefix", "", "Override path to monitoring resource topic")

	flag.BoolVar(&config.cc.ClientMode, "ibmmq.client", false, "Connect as MQ client")
	flag.StringVar(&config.cc.UserId, "ibmmq.userid", "", "UserId for MQ connection")
	// TODO: Also enable a different mechanism to read the password
	flag.StringVar(&config.cc.Password, "ibmmq.password", "", "Password for MQ connection")

	// TODO: turn these on
	//flag.BoolVar(&config.monitorChannelStatistics, "ibmmq.monitorChannelStatistics", false, "Whether to collect channel stats")
	//flag.StringVar(&config.statisticsQueueName, "ibmmq.statisticsQueueName", "SYSTEM.ADMIN.STATISTICS.QUEUE", "Which queue holds channel stats")
	config.monitorChannelStatistics = false
	config.statisticsQueueName = ""

	flag.StringVar(&config.httpListenPort, "ibmmq.httpListenPort", defaultPort, "HTTP Listener")
	flag.StringVar(&config.httpMetricPath, "ibmmq.httpMetricPath", "/metrics", "Path to exporter metrics")

	flag.StringVar(&config.logLevel, "log.level", "error", "Log level - debug, info, error")
	flag.StringVar(&config.namespace, "namespace", defaultNamespace, "Namespace for metrics")

	flag.Parse()

	if config.monitoredQueuesFile != "" {
		config.monitoredQueues = mqmetric.ReadPatterns(config.monitoredQueuesFile)
	}

	if config.cc.UserId != "" && config.cc.Password == "" {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Printf("Enter password: \n")
		scanner.Scan()
		config.cc.Password = scanner.Text()
	}
}
