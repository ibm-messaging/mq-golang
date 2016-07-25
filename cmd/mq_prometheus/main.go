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
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/prometheus/common/log"
	"net/http"
	"os"
)

func main() {
	var err error

	initConfig()
	if config.qMgrName == "" {
		log.Errorln("Must provide a queue manager name to connect to.")
		os.Exit(1)
	}

	log.Infoln("Starting IBM MQ exporter for Prometheus monitoring")

	err = initConnection(config.qMgrName)
	if err == nil {
		log.Infoln("Connected to queue manager ", config.qMgrName)
		defer endConnection()
	}

	// What metrics can the queue manager provide?
	if err == nil {
		err = discoverStats()
	}

	// Which queues have we been asked to monitor? Expand wildcards
	// to explicit names so that subscriptions work.
	if err == nil {
		discoverQueues()
	}

	// Subscribe to all of the various topics
	if err == nil {
		createSubscriptions()
	}

	// Go into main loop for handling requests from prometheus
	if err == nil {
		exporter := newExporter()
		prometheus.MustRegister(exporter)

		http.Handle(config.httpMetricPath, prometheus.Handler())
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write(landingPage())
		})

		log.Infoln("Listening on", config.httpListenPort)
		log.Fatal(http.ListenAndServe(":"+config.httpListenPort, nil))

	}

	if err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}

/*
landingPage gives a very basic response if someone just connects to our port.
The only link on it jumps to the list of available metrics.
*/
func landingPage() []byte {
	return []byte(
		`<html>
<head><title>IBM MQ Exporter</title></head>
<body>
<h1>IBM MQ Exporter</h1>
<p><a href='` + config.httpMetricPath + `'>Metrics</a></p>
</body>
</html>
`)
}
