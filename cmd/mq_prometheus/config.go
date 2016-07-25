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
	log "github.com/prometheus/common/log"
	"os"
)

type mqExporterConfig struct {
	qMgrName            string
	replyQ              string
	monitoredQueues     string
	monitoredQueuesFile string
	httpListenPort      string
	httpMetricPath      string
}

const (
	defaultPort = "9157" // reserved in the prometheus wiki
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

	flag.StringVar(&config.httpListenPort, "ibmmq.httpListenPort", defaultPort, "HTTP Listener")
	flag.StringVar(&config.httpMetricPath, "ibmmq.httpMetricPath", "/metrics", "Path to exporter metrics")

	flag.Parse()

	if config.monitoredQueuesFile != "" {
		config.monitoredQueues = readPatterns(config.monitoredQueuesFile)
	}
}

/*
The list of patterns to be monitored can either be provided on the command line
or in an external file, where the patterns appear one-per-line.
*/
func readPatterns(f string) string {
	var s string

	file, err := os.Open(f)
	if err != nil {
		log.Fatalf("Opening file %s: %s", f, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if s != "" {
			s += ","
		}
		s += scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Reading from %s: %s", f, err)
	}
	log.Infof("Read patterns from %s: '%s'", f, s)

	return s
}
