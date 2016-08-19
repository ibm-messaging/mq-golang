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
	"errors"
	_ "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"strings"
	"time"
)

func newPoint(metric string, timestamp time.Time, value float64, tags map[string]string) (*cloudwatch.MetricDatum, error) {
	var dl []*cloudwatch.Dimension

	if metric == "" {
		return nil, errors.New("PointError: Metric can not be empty")
	}

	d1 := cloudwatch.Dimension{
		Name:  aws.String("qmgr"),
		Value: aws.String(tags["qmgr"]),
	}

	dl = append(dl, &d1)

	if qName, ok := tags["object"]; ok {
		d2 := cloudwatch.Dimension{
			Name:  aws.String("object"),
			Value: aws.String(qName),
		}
		dl = append(dl, &d2)
	}

	unit := aws.String("None")
	if strings.HasSuffix(metric, "_seconds") {
		unit = aws.String("Seconds")
	} else if strings.HasSuffix(metric, "_bytes") {
		unit = aws.String("Bytes")
	}

	return &cloudwatch.MetricDatum{
		Dimensions: dl,
		MetricName: aws.String(metric),
		Timestamp:  aws.Time(timestamp),
		Unit:       unit,
		Value:      aws.Float64(value),
	}, nil
}

/*
BatchPoints is a set of points collected in one iteration.
*/
type BatchPoints struct {
	Points []*cloudwatch.MetricDatum
}

func newBatchPoints() *BatchPoints {
	return &BatchPoints{}
}

func (bp *BatchPoints) addPoint(p *cloudwatch.MetricDatum) {
	bp.Points = append(bp.Points, p)
}
