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
	"math"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ibm-messaging/mq-golang/mqmetric"
)

var (
	first      = true
	errorCount = 0
)

const (
	blankString = "                                "
)

/*
Collect is called by the main routine at regular intervals to provide current
data
*/
func Collect() error {
	var err error

	log.Infof("IBM MQ JSON collector started")

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
		firstPoint := true
		fmt.Printf("\n{\n")
		fmt.Printf("%s\"collectionTime\" : {\n", blankString[0:2])
		t := time.Now()
		fmt.Printf("%s\"timeStamp\" : \"%s\",\n", blankString[0:4], t.Format(time.RFC3339))
		fmt.Printf("%s\"epoch\" : %d\n", blankString[0:4], t.Unix())
		fmt.Printf("%s},\n", blankString[0:2])

		fmt.Printf("%s\"points\" : [\n", blankString[0:2])
		for _, cl := range mqmetric.Metrics.Classes {
			for _, ty := range cl.Types {
				for _, elem := range ty.Elements {
					for key, value := range elem.Values {
						if !firstPoint {
							fmt.Printf(",\n")
						} else {
							firstPoint = false
						}
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
		fmt.Printf("\n%s]\n}\n", blankString[0:2])

	}

	return err

}

func printPoint(metric string, val float32, tags map[string]string) {
	fmt.Printf("%s{\n", blankString[0:2])
	qmgr := tags["qmgr"]
	fmt.Printf("%s\"queueManager\" : \"%s\",\n", blankString[0:4], qmgr)
	if q, ok := tags["object"]; ok {
		fmt.Printf("%s\"queue\" : \"%s\",\n", blankString[0:4], q)
	}
	if float64(val) == math.Trunc(float64(val)) {
		fmt.Printf("%s\"%s\" : %d\n", blankString[0:4], fixup(metric), int64(val))
	} else {
		fmt.Printf("%s\"%s\" : %f\n", blankString[0:4], fixup(metric), val)
	}
	fmt.Printf("%s}", blankString[0:2])
	return
}

func fixup(s1 string) string {
	// Another reformatting of the metric name - this one converts
	// something like queue_avoided_bytes into queueAvoidedBytes
	s2 := ""
	c := ""
	nextCaseUpper := false

	for i := 0; i < len(s1); i++ {
		if s1[i] != '_' {
			if nextCaseUpper {
				c = strings.ToUpper(s1[i : i+1])
				nextCaseUpper = false
			} else {
				c = strings.ToLower(s1[i : i+1])
			}
			s2 += c
		} else {
			nextCaseUpper = true
		}

	}
	return s2
}
