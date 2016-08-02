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
	"encoding/json"
	"errors"
	_ "github.com/Sirupsen/logrus"
)

/*
Point contains the elements needed for a single entry in the
database.
*/
type Point struct {
	// Opentsdb metric name
	Metric string `json:"metric"`

	// Timestamp unix time e.g.: time.Now().Unix()
	Timestamp int64 `json:"timestamp"`

	Value float32 `json:"value"`

	// Map of tags, example: {"host": "desktop"}
	Tags map[string]string `json:"tags"`
}

func newPoint(metric string, timestamp int64, value float32, tags map[string]string) (*Point, error) {
	if metric == "" {
		return nil, errors.New("PointError: Metric can not be empty")
	}

	return &Point{
		Metric:    metric,
		Timestamp: timestamp,
		Value:     value,
		Tags:      tags,
	}, nil
}

/*
BatchPoints is the set of points collected in one iteration.
*/
type BatchPoints struct {
	Points []*Point `json:""`
}

func newBatchPoints() *BatchPoints {
	return &BatchPoints{}
}

func (bp *BatchPoints) addPoint(p *Point) {
	bp.Points = append(bp.Points, p)
}

func (bp *BatchPoints) toJSON() ([]byte, error) {
	j, err := json.Marshal(bp.Points)
	//log.Debug("Points set = ", string(j))
	return j, err
}
