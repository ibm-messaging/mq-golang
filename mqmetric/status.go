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

import (
	"fmt"
	"strings"
	"time"
)

var statusDummy = fmt.Sprintf("dummy")

/*
This file defines types and constructors for elements related to status
of MQ objects that are retrieved via polling commands such as DISPLAY CHSTATUS
*/

type StatusAttribute struct {
	Description string
	MetricName  string
	Pseudo      bool
	pcfAttr     int32
	squash      bool
	delta       bool
	index       int
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
	s.index = -1
	s.Values = make(map[string]*StatusValue)
	s.prevValues = make(map[string]int64)
	s.Pseudo = false
	return s
}

func newPseudoStatusAttribute(n string, d string) *StatusAttribute {
	s := newStatusAttribute(n, d, -1)
	s.Pseudo = true
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

// Go uses an example-based method for formatting and parsing timestamps
// This layout matches the MQ PutDate and PutTime strings. An additional TZ
// may eventually have to be turned into a config parm. Note the "15" to indicate
// a 24-hour timestamp. There also seems to be two formats for the time layout comnig
// from MQ - TPSTATUS uses a colon format time, QSTATUS uses the dots.
const timeStampLayoutDot = "2006-01-02 15.04.05"
const timeStampLayoutColon = "2006-01-02 15:04:05"

// Convert the MQ Time and Date formats
func statusTimeDiff(now time.Time, d string, t string) int64 {
	var rc int64
	var err error
	var parsedT time.Time

	// If there's any error in parsing the timestamp - perhaps
	// the value has not been set yet, then just return 0
	rc = 0

	timeStampLayout := timeStampLayoutDot
	if len(d) == 10 && len(t) == 8 {
		if strings.Contains(t, ":") {
			timeStampLayout = timeStampLayoutColon
		}
		parsedT, err = time.ParseInLocation(timeStampLayout, d+" "+t, now.Location())
		if err == nil {
			diff := now.Sub(parsedT).Seconds()

			if diff < 0 { // Cannot have status from the future
				// TODO: Perhaps issue a one-time warning as it might indicate timezone offsets
				// are mismatched between the qmgr and this program
				diff = 0
			}
			rc = int64(diff)
		}
	}
	//fmt.Printf("statusTimeDiff d:%s t:%s diff:%d err:%v\n",d,t,rc,err)
	return rc
}
