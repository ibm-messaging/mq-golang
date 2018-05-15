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
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/ibm-messaging/mq-golang/mqmetric"
	"github.com/prometheus/client_golang/prometheus"
)

type exporter struct {
	mutex   sync.RWMutex
	metrics mqmetric.AllMetrics
}

func newExporter() *exporter {
	return &exporter{
		metrics: mqmetric.Metrics,
	}
}

var (
	first    = true
	gaugeMap = make(map[string]*prometheus.GaugeVec)
)

/*
Describe is called by Prometheus on startup of this monitor. It needs to tell
the caller about all of the available metrics.
*/
func (e *exporter) Describe(ch chan<- *prometheus.Desc) {

	log.Infof("IBMMQ Describe started")

	for _, cl := range e.metrics.Classes {
		for _, ty := range cl.Types {
			for _, elem := range ty.Elements {
				gaugeMap[makeKey(elem)].Describe(ch)
			}
		}
	}
}

/*
Collect is called by Prometheus at regular intervals to provide current
data
*/
func (e *exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()

	log.Infof("IBMMQ Collect started")

	// Clear out everything we know so far. In particular, replace
	// the map of values for each object so the collection starts
	// clean.
	for _, cl := range e.metrics.Classes {
		for _, ty := range cl.Types {
			for _, elem := range ty.Elements {
				gaugeMap[makeKey(elem)].Reset()
				elem.Values = make(map[string]int64)
			}
		}
	}

	// Deal with all the publications that have arrived
	mqmetric.ProcessPublications()

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

		for _, cl := range e.metrics.Classes {
			for _, ty := range cl.Types {
				for _, elem := range ty.Elements {
					for key, value := range elem.Values {
						f := mqmetric.Normalise(elem, key, value)
						g := gaugeMap[makeKey(elem)]
						if key == mqmetric.QMgrMapKey {
							g.WithLabelValues(config.qMgrName).Set(f)
						} else {
							g.WithLabelValues(key, config.qMgrName).Set(f)
						}
					}
				}
			}
		}
	}

	// And finally tell Prometheus about the data
	for _, cl := range e.metrics.Classes {
		for _, ty := range cl.Types {
			for _, elem := range ty.Elements {
				gaugeMap[makeKey(elem)].Collect(ch)
			}
		}
	}

}

/*
allocateGauges creates a Prometheus gauge for each
resource that we know about. These are stored in a local map keyed
from the resource names.
*/
func allocateGauges() {
	for _, cl := range mqmetric.Metrics.Classes {
		for _, ty := range cl.Types {
			for _, elem := range ty.Elements {
				g := newMqGaugeVec(elem)
				key := makeKey(elem)
				gaugeMap[key] = g
			}
		}
	}
}

/*
makeKey uses the 3 parts of a resource's name to build a unique string.
The "/" character cannot be part of a name, so is a convenient way
to build a unique key. If we ever have metrics for other object
types such as topics, then the object type would be used too.
This key is not used outside of this module, so the format can change.
*/
func makeKey(elem *mqmetric.MonElement) string {
	key := elem.Parent.Parent.Name + "/" +
		elem.Parent.Name + "/" +
		elem.MetricName
	return key
}

/*
newMqGaugeVec returns the structure which will contain the
values and suitable labels. For queues we tag each entry
with both the queue and qmgr name; for the qmgr-wide entries, we
only need the single label.
*/
func newMqGaugeVec(elem *mqmetric.MonElement) *prometheus.GaugeVec {
	queueLabelNames := []string{"object", "qmgr"}
	qmgrLabelNames := []string{"qmgr"}

	labels := qmgrLabelNames
	prefix := "qmgr_"

	if strings.Contains(elem.Parent.ObjectTopic, "%s") {
		labels = queueLabelNames
		prefix = "object_"
	}

	gaugeVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.namespace,
			Name:      prefix + elem.MetricName,
			Help:      elem.Description,
		},
		labels,
	)

	log.Infof("Created gauge for %s", elem.MetricName)
	return gaugeVec
}

/*
newMqGaugeVec returns the structure which will contain the
values and suitable labels. For queues we tag each entry
with both the queue and qmgr name; for the qmgr-wide entries, we
only need the single label.
*/

// TODO: Finish this
/*
func newMqGaugeVecChl(elem *ibmmq.Statistic) *prometheus.GaugeVec {
        prefix := "channel_"

        gaugeVec := prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Namespace: config.namespace,
                        Name:      prefix + elem.MetricName,
                        Help:      elem.Description,
                },
                labels,
        )

        log.Infof("Created gauge for %s", elem.MetricName)
        return gaugeVec
}
*/
