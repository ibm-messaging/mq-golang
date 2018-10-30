/*
Package mqmetric contains a set of routines common to several
commands used to export MQ metrics to different backend
storage mechanisms including Prometheus and InfluxDB.
*/
package mqmetric

/*
  Copyright (c) IBM Corporation 2016, 2018

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

   Contributors:
     Mark Taylor - Initial Contribution
*/

/*
This file defines types and constructors for elements related to status
of MQ objects that are retrieved via polling commands such as DISPLAY CHSTATUS
*/

type StatusAttribute struct {
	Description string
	MetricName  string
	pcfAttr     int32
	squash      bool
	delta       bool
	Values      map[string]*StatusValue
	prevValues  map[string]int64
}

type StatusSet struct {
	Attributes map[string]*StatusAttribute
}

// All we care about for attributes are ints and strings. Other complex
// PCF datatypes are not currently going to be returned through this mechanism
type StatusValue struct {
	IsInt64     bool
	ValueInt64  int64
	ValueString string
}

// Initialise with default values.
func newStatusAttribute(n string, d string, p int32) *StatusAttribute {
	s := new(StatusAttribute)
	s.MetricName = n
	s.Description = d
	s.pcfAttr = p
	s.squash = false
	s.delta = false
	s.Values = make(map[string]*StatusValue)
	s.prevValues = make(map[string]int64)
	return s
}

func newStatusValueInt64(v int64) *StatusValue {
	s := new(StatusValue)
	s.ValueInt64 = v
	s.IsInt64 = true
	return s
}

func newStatusValueString(v string) *StatusValue {
	s := new(StatusValue)
	s.ValueString = v
	s.IsInt64 = false
	return s
}
