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
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ibm-messaging/mq-golang/mqmetric"
)

func initLog() {
	level, err := log.ParseLevel(config.logLevel)
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)
}

func main() {
	var err error
	var c client.Client

	initConfig()
	if config.qMgrName == "" {
		log.Errorln("Must provide a queue manager name to connect to.")
		os.Exit(1)
	}
	d, err := time.ParseDuration(config.interval + "s")
	if err != nil {
		log.Errorln("Invalid value for interval parameter: ", err)
		os.Exit(1)
	}

	initLog()
	log.Infoln("Starting IBM MQ metrics exporter for InfluxDB monitoring")

	// Connect and open standard queues
	err = mqmetric.InitConnection(config.qMgrName, config.replyQ, &config.cc)
	if err == nil {
		log.Infoln("Connected to queue manager ", config.qMgrName)
		defer mqmetric.EndConnection()
	}

	// What metrics can the queue manager provide? Find out, and
	// subscribe.
	if err == nil {
		err = mqmetric.DiscoverAndSubscribe(config.monitoredQueues, true, "")
	}

	// Go into main loop for sending data to database
	// Creating the client is not likely to have an error; the error will
	// come during the write of the data.
	if err == nil {
		c, err = client.NewHTTPClient(client.HTTPConfig{
			Addr:     config.databaseAddress,
			Username: config.userid,
			Password: config.password,
		})

		if err != nil {
			log.Error(err)
		} else {
			for {
				Collect(c)
				time.Sleep(d)
			}
		}

	}

	if err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}
