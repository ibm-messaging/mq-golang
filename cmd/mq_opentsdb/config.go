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
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/ibm-messaging/mq-golang/mqmetric"
)

type mqOpenTSDBConfig struct {
	qMgrName            string
	replyQ              string
	monitoredQueues     string
	monitoredQueuesFile string

	cc mqmetric.ConnectionConfig

	databaseName    string
	databaseAddress string
	userid          string
	password        string
	passwordFile    string

	interval  string
	maxErrors int
	maxPoints int

	logLevel string
}

var config mqOpenTSDBConfig

/*
initConfig parses the command line parameters.
*/
func initConfig() {

	flag.StringVar(&config.qMgrName, "ibmmq.queueManager", "", "Queue Manager name")
	flag.StringVar(&config.replyQ, "ibmmq.replyQueue", "SYSTEM.DEFAULT.MODEL.QUEUE", "Reply Queue to collect data")
	flag.StringVar(&config.monitoredQueues, "ibmmq.monitoredQueues", "", "Patterns of queues to monitor")
	flag.StringVar(&config.monitoredQueuesFile, "ibmmq.monitoredQueuesFile", "", "File with patterns of queues to monitor")

	flag.BoolVar(&config.cc.ClientMode, "ibmmq.client", false, "Connect as MQ client")

	flag.StringVar(&config.databaseName, "ibmmq.databaseName", "", "Name of database")
	flag.StringVar(&config.databaseAddress, "ibmmq.databaseAddress", "", "Address of database eg http://example.com:8086")
	flag.StringVar(&config.userid, "ibmmq.databaseUserID", "", "UserID to access the database")
	flag.StringVar(&config.passwordFile, "ibmmq.pwFile", "", "Where is password help temporarily")
	flag.StringVar(&config.interval, "ibmmq.interval", "10", "How many seconds between each collection")
	flag.IntVar(&config.maxErrors, "ibmmq.maxErrors", 10000, "Maximum number of errors communicating with server before considered fatal")
	flag.IntVar(&config.maxPoints, "ibmmq.maxPoints", 30, "Maximum number of points to include in each write to the server")

	flag.StringVar(&config.logLevel, "log.level", "error", "Log level - debug, info, error")

	flag.Parse()

	if config.monitoredQueuesFile != "" {
		config.monitoredQueues = mqmetric.ReadPatterns(config.monitoredQueuesFile)
	}

	// Read password from a file if there is a userid on the command line
	// Delete the file after reading it.
	if config.userid != "" {
		config.userid = strings.TrimSpace(config.userid)

		f, err := os.Open(config.passwordFile)
		if err != nil {
			log.Fatalf("Opening file %s: %s", f, err)
		}

		defer os.Remove(config.passwordFile)
		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Scan()
		p := scanner.Text()
		err = scanner.Err()
		if err != nil {
			log.Fatalf("Reading file %s: %s", f, err)
		}
		config.password = strings.TrimSpace(p)
	}
}
