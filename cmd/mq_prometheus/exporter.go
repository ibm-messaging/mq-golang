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
This file provides the main link between the MQ monitoring collection, and
the Prometheus request for data. The Collect() function is the key operation
invoked at the scrape intervals, causing us to read available publications
and update the various Gauges.
*/

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/prometheus/common/log"
	"ibmmq"
	"strings"
	"sync"
)

type exporter struct {
	mutex   sync.RWMutex
	metrics allMetrics
}

func newExporter() *exporter {
	return &exporter{
		metrics: metrics,
	}
}

var (
	first = true
)

/*
fetchPublications has to read all of the messages since the last scrape
and update the values for every relevant gauge.

Because the generation of the messages by the qmgr, and being told to
read them by Prometheus, may not have identical frequencies, there may be
cases where multiple pieces of data have to be collated for the same
gauge. Conversely, there may be times when prometheus calls us but there
are no metrics to update.
*/
func (e *exporter) fetchPublications() {
	var err error
	var data []byte

	var qName string
	var classidx int
	var typeidx int
	var elementidx int
	var value int64

	// Keep reading all available messages until queue is empty. Don't
	// do a GET-WAIT; just immediate removals.
	for err == nil {
		data, err = getMessage(false)
		elemList, _ := parsePCFResponse(data)

		// Most common error will be MQRC_NO_MESSAGE_AVAILABLE
		// which will end the loop.
		if err == nil {
			// A typical publication contains some fixed
			// headers (qmgrName, objectName, class, type etc)
			// followed by a list of index/values.

			// This map contains those element indexes and values from each message
			values := make(map[int]int64)

			qName = ""

			for i := 0; i < len(elemList); i++ {
				switch elemList[i].Parameter {
				case ibmmq.MQCA_Q_MGR_NAME:
					_ = strings.TrimSpace(elemList[i].String[0])
				case ibmmq.MQCA_Q_NAME:
					qName = strings.TrimSpace(elemList[i].String[0])
				case ibmmq.MQIACF_OBJECT_TYPE:
					// Will need to use this as part of the object key and
					// labelling if/when MQ starts to produce stats for other types
					// such as a topic. But for now we can ignore it.
					_ = ibmmq.MQItoString("OT", int(elemList[i].Int64Value[0]))
				case ibmmq.MQIAMO_MONITOR_CLASS:
					classidx = int(elemList[i].Int64Value[0])
				case ibmmq.MQIAMO_MONITOR_TYPE:
					typeidx = int(elemList[i].Int64Value[0])
				case ibmmq.MQIAMO64_MONITOR_INTERVAL:
					_ = elemList[i].Int64Value[0]
				case ibmmq.MQIAMO_MONITOR_FLAGS:
					_ = int(elemList[i].Int64Value[0])
				default:
					value = elemList[i].Int64Value[0]
					elementidx = int(elemList[i].Parameter)
					values[elementidx] = value
				}
			}

			// Now have all the values in this particular message
			// Have to incorporate them into any that already exist
			// For some, that's simply a matter of adding them.
			// For some others, we'll just take the latest.
			//
			// Each element contains a map holding all the objects
			// touched by these messages. The map is referenced by
			// object name if it's a queue; for qmgr-level stats, the
			// map only needs to contain a single entry which I've
			// chosen to reference by "@self" which can never be a
			// real queue name.
			//
			for key, newValue := range values {
				if element, ok := metrics.classes[classidx].types[typeidx].elements[key]; ok {
					objectName := qName
					if objectName == "" {
						objectName = qMgrMapKey
					}

					if oldValue, ok := element.values[objectName]; ok {
						value = updateValue(element, oldValue, newValue)
					} else {
						value = newValue
					}
					element.values[objectName] = value

				}
			}
		}
	}

	// Have now processed all of the publications, and all the MQ-owned
	// value fields and maps have been updated.
	//
	// Now need to set all of the real Gauges with the correct values
	if first {
		// Always ignore the first loop through as there might
		// be accumulated stuff from a while ago, and lead to
		// a misleading range on graphs.
		first = false
	} else {

		for _, thisClass := range e.metrics.classes {
			for _, thisType := range thisClass.types {
				for _, thisElement := range thisType.elements {
					gaugeVec := thisElement.gaugeVec
					for key, value := range thisElement.values {
						// I've  seen negative numbers which are nonsense,
						// possibly 32-bit overflow or uninitialised values
						// in the qmgr. So force data to something sensible
						// just in case those were due to a bug.
						f := float64(value)
						if f < 0 {
							f = 0
						}

						log.Debugf("Pushing Elem %s [%s] Type %d Value %f", thisElement.metricName, key, thisElement.datatype, f)

						// Convert suitable metrics to base units
						if thisElement.datatype == ibmmq.MQIAMO_MONITOR_PERCENT ||
							thisElement.datatype == ibmmq.MQIAMO_MONITOR_HUNDREDTHS {
							f = f / 100
						} else if thisElement.datatype == ibmmq.MQIAMO_MONITOR_MB {
							f = f * 1024 * 1024
						} else if thisElement.datatype == ibmmq.MQIAMO_MONITOR_GB {
							f = f * 1024 * 1024 * 1024
						} else if thisElement.datatype == ibmmq.MQIAMO_MONITOR_MICROSEC {
							f = f / 1000000
						}

						if key == qMgrMapKey {
							gaugeVec.WithLabelValues(config.qMgrName).Set(f)
						} else {
							gaugeVec.WithLabelValues(key, config.qMgrName).Set(f)
						}
					}
				}
			}
		}
	}

}

/*
Describe is called by prometheus on startup of this monitor. It needs to tell
the caller about all of the available metrics.
*/
func (e *exporter) Describe(ch chan<- *prometheus.Desc) {

	log.Infof("IBMMQ Describe started")

	for _, thisClass := range e.metrics.classes {
		for _, thisType := range thisClass.types {
			for _, thisElement := range thisType.elements {
				thisElement.gaugeVec.Describe(ch)
			}
		}
	}
}

/*
Collect is called by prometheus at regular intervals to provide current
data
*/
func (e *exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()

	log.Infof("IBMMQ Collect started")

	// Clear out everything we know so far. In particular, replace
	// the map of values for each object so the collection starts
	// clean
	for _, thisClass := range e.metrics.classes {
		for _, thisType := range thisClass.types {
			for _, thisElement := range thisType.elements {
				thisElement.gaugeVec.Reset()
				thisElement.values = make(map[string]int64)
			}
		}
	}

	// Process all the publications that have arrived
	e.fetchPublications()

	// And now tell prometheus about the data
	for _, thisClass := range e.metrics.classes {
		for _, thisType := range thisClass.types {
			for _, thisElement := range thisType.elements {
				thisElement.gaugeVec.Collect(ch)
			}
		}
	}

}

/*
updateValue calculates whether we need to add the values contained from
multiple publications that might have arrived in the scrape interval
for the same resource, or whether we should just overwrite with the latest.
For example, "RAM total bytes" is useful at its current value, not the
summation of now and 10 seconds ago.

Although there are several monitor datatypes, all of them apart from
explicitly labelled "DELTA" are ones we should just return the latest
value.
*/
func updateValue(elem *monElement, oldValue int64, newValue int64) int64 {

	if elem.datatype == ibmmq.MQIAMO_MONITOR_DELTA {
		return oldValue + newValue
	}

	return newValue
}
