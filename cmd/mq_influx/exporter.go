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

/*
This file pushes collected data to the InfluxDB.
The Collect() function is the key operation
invoked at the configured intervals, causing us to read available publications
and update the various data points.
*/

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ibm-messaging/mq-golang/mqmetric"
	client "github.com/influxdata/influxdb/client/v2"
)

var (
	first      = true
	errorCount = 0
)

/*
Collect is called by the main routine at regular intervals to provide current
data
*/
func Collect(c client.Client) error {
	var err error
	var series string
	log.Infof("IBMMQ InfluxDB collection started")

	// Clear out everything we know so far. In particular, replace
	// the map of values for each object so the collection starts
	// clean.
	for _, cl := range mqmetric.Metrics.Classes {
		for _, ty := range cl.Types {
			for _, elem := range ty.Elements {
				elem.Values = make(map[string]int64)
			}
		}
	}

	// Process all the publications that have arrived
	mqmetric.ProcessPublications()

	// Have now processed all of the publications, and all the MQ-owned
	// value fields and maps have been updated.
	//
	// Now need to set all of the real items with the correct values
	if first {
		// Always ignore the first loop through as there might
		// be accumulated stuff from a while ago, and lead to
		// a misleading range on graphs.
		first = false
	} else {
		t := time.Now()
		bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
			Database:  config.databaseName,
			Precision: "ms",
		})
		if err != nil {
			// This kind of error is so unlikely, it should be treated as fatal
			log.Fatalln("Error creating batch points: ", err)
		}

		for _, cl := range mqmetric.Metrics.Classes {
			for _, ty := range cl.Types {
				for _, elem := range ty.Elements {
					for key, value := range elem.Values {
						f := mqmetric.Normalise(elem, key, value)
						tags := map[string]string{
							"qmgr": config.qMgrName,
						}

						series = "qmgr"
						if key != mqmetric.QMgrMapKey {
							tags["object"] = key
							series = "queue"
						}
						fields := map[string]interface{}{elem.MetricName: f}
						pt, _ := client.NewPoint(series, tags, fields, t)
						bp.AddPoint(pt)
						log.Debugf("Adding point %v", pt)
					}
				}
			}
		}

		// This is where real errors might occur, including the inability to
		// contact the database server. We will ignore (but log)  these errors
		// up to a threshold, after which it is considered fatal.
		err = c.Write(bp)
		if err != nil {
			log.Error(err)
			errorCount++
			if errorCount >= config.maxErrors {
				log.Fatal("Too many errors communicating with server")
			}
		} else {
			errorCount = 0
		}
	}

	return err

}
