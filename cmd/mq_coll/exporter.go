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
The Collect() function is the key operation
invoked at the configured intervals, causing us to read available publications
and update the various data points.
*/

import (
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/ibm-messaging/mq-golang/mqmetric"
)

var (
	first      = true
	errorCount = 0
)

/*
Collect is called by the main routine at regular intervals to provide current
data
*/
func Collect() error {
	var err error

	log.Infof("IBM MQ stdout collector started")

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

		for _, cl := range mqmetric.Metrics.Classes {
			for _, ty := range cl.Types {
				for _, elem := range ty.Elements {
					for key, value := range elem.Values {
						f := mqmetric.Normalise(elem, key, value)
						tags := map[string]string{
							"qmgr": config.qMgrName,
						}

						if key != mqmetric.QMgrMapKey {
							tags["object"] = key
						}
						printPoint(elem.MetricName, float32(f), tags)

					}
				}
			}
		}

	}

	return err

}

func printPoint(metric string, val float32, tags map[string]string) {
	qmgr := tags["qmgr"]
	if q, ok := tags["object"]; ok {
		if !strings.HasPrefix(metric, "queue") {
			metric = "queue_" + metric
		}
		metric += "-" + fixup(q)
	}
	fmt.Printf("PUTVAL %s/%s-%s/%s interval=%s N:%f\n",
		config.hostlabel, "qmgr", fixup(qmgr), metric, config.interval, val)
	return
}

func fixup(s1 string) string {
	s2 := strings.Replace(s1, ".", "_", -1)
	return s2
}
