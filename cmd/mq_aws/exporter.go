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
This file pushes collected data to the Amazon CloudWatch service.
The Collect() function is the key operation
invoked at the configured intervals, causing us to read available publications
and update the various data points.
*/

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/ibm-messaging/mq-golang/mqmetric"
)

type client struct {
	sess *session.Session
	svc  *cloudwatch.CloudWatch
}

var (
	first      = true
	errorCount = 0
	c          client
)

/*
Collect is called by the main routine at regular intervals to provide current
data
*/
func Collect() error {
	var err error
	var series string
	log.Infof("IBM MQ AWS collection started")

	if c.sess == nil {
		c.sess, err = session.NewSession()
		if err != nil {
			log.Fatal("Cannot create session: ", err)
		}
		c.svc = nil
	}

	if c.svc == nil {
		if config.region == "" {
			c.svc = cloudwatch.New(c.sess, aws.NewConfig())
		} else {
			c.svc = cloudwatch.New(c.sess, aws.NewConfig().WithRegion(config.region))
		}
		if err != nil {
			log.Fatal("Cannot create service: ", err)
		}
	}

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
		bp := newBatchPoints()

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
						pt, _ := newPoint(series+"."+elem.MetricName, t, float64(f), tags)
						bp.addPoint(pt)

						// AWS recommends not sending too many
						// data points in a single request.
						// So we flush after a configurable set has been
						// collected.
						if len(bp.Points) >= config.maxPoints {
							bp = c.Flush(bp)
						}
						//log.Debugf("Adding point %v", pt)
					}
				}
			}
		}

		c.Flush(bp)
	}

	return err

}

func (c client) Flush(bp *BatchPoints) *BatchPoints {
	// This is where real errors might occur, including the inability to
	// contact the server. We will ignore (but log)  these errors
	// up to a threshold, after which it is considered fatal.
	if len(bp.Points) > 0 {
		err := c.Put(bp)
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
	bp = newBatchPoints()
	return bp
}

func (c client) Put(bp *BatchPoints) error {
	if len(bp.Points) == 0 {
		return nil
	}

	log.Infof("Putting %d points", len(bp.Points))
	params := &cloudwatch.PutMetricDataInput{
		MetricData: bp.Points,
		Namespace:  aws.String(config.namespace),
	}

	_, err := c.svc.PutMetricData(params)
	return err
}
