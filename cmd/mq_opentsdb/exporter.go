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
This file pushes collected data to OpenTSDB.
The Collect() function is the key operation
invoked at the configured intervals, causing us to read available publications
and update the various data points.
*/

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ibm-messaging/mq-golang/mqmetric"
)

type client struct {
	url        *url.URL
	httpClient *http.Client
	tr         *http.Transport
}

var (
	first      = true
	errorCount = 0
	c          *client
)

/*
Collect is called by the main routine at regular intervals to provide current
data
*/
func Collect() error {
	var err error
	var series string
	log.Infof("IBM MQ OpenTSDB collection started")

	if c == nil {
		c, err = newClient()
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
		t := time.Now().Unix()
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
						pt, _ := newPoint(series+"."+elem.MetricName, t, float32(f), tags)
						bp.addPoint(pt)

						// OpenTSDB recommends not sending too many
						// data points in a single request. Large requests
						// may require http chunking which is disabled by default
						// in the database. So we flush after a configurable set has been
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

func (c *client) Flush(bp *BatchPoints) *BatchPoints {
	// This is where real errors might occur, including the inability to
	// contact the database server. We will ignore (but log)  these errors
	// up to a threshold, after which it is considered fatal.
	if len(bp.Points) > 0 {
		_, err := c.Put(bp, "details")
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

func newClient() (*client, error) {

	tr := &http.Transport{}
	u, err := url.Parse(config.databaseAddress)
	if err != nil {
		return nil, err
	}

	return &client{
		url: u,
		httpClient: &http.Client{
			Timeout:   0,
			Transport: tr,
		},
		tr: tr,
	}, nil
}

func (c *client) Close() error {
	c.tr.CloseIdleConnections()
	return nil
}

func (c *client) Put(bp *BatchPoints, params string) ([]byte, error) {
	if len(bp.Points) == 0 {
		return nil, nil
	}

	data, err := bp.toJSON()
	if err != nil {
		return nil, err
	}
	log.Debugf("Serialised points are %s", string(data))

	u := c.url
	u.Path = "api/put"
	u.RawQuery = params

	req, err := http.NewRequest("POST", u.String(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	// log.Infof("Request is %v", req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Infoln("Response body: ", string(body))
	return body, nil
}
