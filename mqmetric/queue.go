/*
Package mqmetric contains a set of routines common to several
commands used to export MQ metrics to different backend
storage mechanisms including Prometheus and InfluxDB.
*/
package mqmetric

/*
  Copyright (c) IBM Corporation 2018

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
Functions in this file use the DISPLAY QueueStatus command to extract metrics
about MQ queues
*/

import (
	"github.com/ibm-messaging/mq-golang/ibmmq"
	"strings"
)

const (
	ATTR_Q_NAME        = "name"
	ATTR_Q_MSGAGE      = "oldest_message_age"
	ATTR_Q_IPPROCS     = "input_handles"
	ATTR_Q_OPPROCS     = "output_handles"
	ATTR_Q_QTIME_SHORT = "qtime_short"
	ATTR_Q_QTIME_LONG  = "qtime_long"
	ATTR_Q_DEPTH       = "depth"
)

var QueueStatus StatusSet
var qAttrsInit = false

/*
Unlike the statistics produced via a topic, there is no discovery
of the attributes available in object STATUS queries. There is also
no discovery of descriptions for them. So this function hardcodes the
attributes we are going to look for and gives the associated descriptive
text. The elements can be expanded later; just trying to give a starting point
for now.
*/
func QueueInitAttributes() {
	if qAttrsInit {
		return
	}
	QueueStatus.Attributes = make(map[string]*StatusAttribute)

	attr := ATTR_Q_NAME
	QueueStatus.Attributes[attr] = newStatusAttribute(attr, "Queue Name", -1)

	// These are the integer status fields that are of interest
	attr = ATTR_Q_MSGAGE
	QueueStatus.Attributes[attr] = newStatusAttribute(attr, "Oldest Message", ibmmq.MQIACF_OLDEST_MSG_AGE)
	attr = ATTR_Q_IPPROCS
	QueueStatus.Attributes[attr] = newStatusAttribute(attr, "Input Handles", ibmmq.MQIA_OPEN_INPUT_COUNT)
	attr = ATTR_Q_OPPROCS
	QueueStatus.Attributes[attr] = newStatusAttribute(attr, "Input Handles", ibmmq.MQIA_OPEN_OUTPUT_COUNT)
	// Usually we get the QDepth from published resources, But on z/OS we can get it from the QSTATUS response
	if platform == ibmmq.MQPL_ZOS {
		attr = ATTR_Q_DEPTH
		QueueStatus.Attributes[attr] = newStatusAttribute(attr, "Queue Depth", ibmmq.MQIA_CURRENT_Q_DEPTH)
	}

	attr = ATTR_Q_QTIME_SHORT
	QueueStatus.Attributes[attr] = newStatusAttribute(attr, "Queue Time Short", ibmmq.MQIACF_Q_TIME_INDICATOR)
	QueueStatus.Attributes[attr].index = 0
	attr = ATTR_Q_QTIME_LONG
	QueueStatus.Attributes[attr] = newStatusAttribute(attr, "Queue Time Long", ibmmq.MQIACF_Q_TIME_INDICATOR)
	QueueStatus.Attributes[attr].index = 1

	qAttrsInit = true
}

// If we need to list the queues that match a pattern. Not needed for
// the status queries as they (unlike the pub/sub resource stats) accept
// patterns in the
func InquireQueues(patterns string) ([]string, error) {
	QueueInitAttributes()
	return inquireObjects(patterns, ibmmq.MQOT_Q)
}

func CollectQueueStatus(patterns string) error {
	var err error

	QueueInitAttributes()

	// Empty any collected values
	for k := range QueueStatus.Attributes {
		QueueStatus.Attributes[k].Values = make(map[string]*StatusValue)
	}

	queuePatterns := strings.Split(patterns, ",")
	if len(queuePatterns) == 0 {
		return nil
	}

	for _, pattern := range queuePatterns {
		pattern = strings.TrimSpace(pattern)
		if len(pattern) == 0 {
			continue
		}

		err = collectQueueStatus(pattern, ibmmq.MQOT_Q)

	}

	return err

}

// Issue the INQUIRE_QUEUE_STATUS command for a queue or wildcarded queue name
// Collect the responses and build up the statistics
func collectQueueStatus(pattern string, instanceType int32) error {
	var err error
	var datalen int

	putmqmd := ibmmq.NewMQMD()
	pmo := ibmmq.NewMQPMO()

	pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
	pmo.Options |= ibmmq.MQPMO_NEW_MSG_ID
	pmo.Options |= ibmmq.MQPMO_NEW_CORREL_ID
	pmo.Options |= ibmmq.MQPMO_FAIL_IF_QUIESCING

	putmqmd.Format = "MQADMIN"
	putmqmd.ReplyToQ = statusReplyQObj.Name
	putmqmd.MsgType = ibmmq.MQMT_REQUEST
	putmqmd.Report = ibmmq.MQRO_PASS_DISCARD_AND_EXPIRY

	buf := make([]byte, 0)
	// Empty replyQ in case any left over from previous errors
	for ok := true; ok; {
		getmqmd := ibmmq.NewMQMD()
		gmo := ibmmq.NewMQGMO()
		gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
		gmo.Options |= ibmmq.MQGMO_FAIL_IF_QUIESCING
		gmo.Options |= ibmmq.MQGMO_NO_WAIT
		gmo.Options |= ibmmq.MQGMO_CONVERT
		gmo.Options |= ibmmq.MQGMO_ACCEPT_TRUNCATED_MSG
		_, err = statusReplyQObj.Get(getmqmd, gmo, buf)

		if err != nil && err.(*ibmmq.MQReturn).MQCC == ibmmq.MQCC_FAILED {
			ok = false
		}
	}
	buf = make([]byte, 0)

	cfh := ibmmq.NewMQCFH()
	cfh.Version = ibmmq.MQCFH_VERSION_3
	cfh.Type = ibmmq.MQCFT_COMMAND_XR

	// Can allow all the other fields to default
	cfh.Command = ibmmq.MQCMD_INQUIRE_Q_STATUS

	// Add the parameters one at a time into a buffer
	pcfparm := new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_STRING
	pcfparm.Parameter = ibmmq.MQCA_Q_NAME
	pcfparm.String = []string{pattern}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	pcfparm = new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_INTEGER
	pcfparm.Parameter = ibmmq.MQIACF_Q_STATUS_TYPE
	pcfparm.Int64Value = []int64{int64(ibmmq.MQIACF_Q_STATUS)}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	// Once we know the total number of parameters, put the
	// CFH header on the front of the buffer.
	buf = append(cfh.Bytes(), buf...)

	// And now put the command to the queue
	err = cmdQObj.Put(putmqmd, pmo, buf)
	if err != nil {
		return err

	}

	// Now get the responses - loop until all have been received (one
	// per queue) or we run out of time
	replyBuf := make([]byte, 10240)
	for allReceived := false; !allReceived; {
		getmqmd := ibmmq.NewMQMD()
		gmo := ibmmq.NewMQGMO()
		gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
		gmo.Options |= ibmmq.MQGMO_FAIL_IF_QUIESCING
		gmo.Options |= ibmmq.MQGMO_WAIT
		gmo.Options |= ibmmq.MQGMO_CONVERT
		gmo.WaitInterval = 3 * 1000 // 3 seconds

		datalen, err = statusReplyQObj.Get(getmqmd, gmo, replyBuf)
		if err == nil {
			cfh, offset := ibmmq.ReadPCFHeader(replyBuf)

			if cfh.Control == ibmmq.MQCFC_LAST {
				allReceived = true
			}
			if cfh.Reason != ibmmq.MQRC_NONE {
				continue
			}
			parseQData(instanceType, cfh, replyBuf[offset:datalen])

		}
	}

	return err
}

// Given a PCF response message, parse it to extract the desired statistics
func parseQData(instanceType int32, cfh *ibmmq.MQCFH, buf []byte) string {
	var elem *ibmmq.PCFParameter

	qName := ""
	key := ""

	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh.ParameterCount == 0 {
		return ""
	}

	// Parse it once to extract the fields that are needed for the map key
	for parmAvail && cfh.CompCode != ibmmq.MQCC_FAILED {
		elem, bytesRead = ibmmq.ReadPCFParameter(buf[offset:])
		offset += bytesRead
		// Have we now reached the end of the message
		if offset >= datalen {
			parmAvail = false
		}

		switch elem.Parameter {
		case ibmmq.MQCA_Q_NAME:
			qName = strings.TrimSpace(elem.String[0])
		}
	}

	// Create a unique key for this instance
	key = qName

	QueueStatus.Attributes[ATTR_Q_NAME].Values[key] = newStatusValueString(qName)

	// And then re-parse the message so we can store the metrics now knowing the map key
	parmAvail = true
	offset = 0
	for parmAvail && cfh.CompCode != ibmmq.MQCC_FAILED {
		elem, bytesRead = ibmmq.ReadPCFParameter(buf[offset:])
		offset += bytesRead
		// Have we now reached the end of the message
		if offset >= datalen {
			parmAvail = false
		}

		// Look at the Parameter and loop through all the possible status
		// attributes to find it.We don't break from the loop after finding a match
		// because there might be more than one attribute associated with the
		// attribute (in particular status/status_squash)
		usableValue := false
		if elem.Type == ibmmq.MQCFT_INTEGER || elem.Type == ibmmq.MQCFT_INTEGER64 {
			usableValue = true
		} else if elem.Type == ibmmq.MQCFT_INTEGER_LIST || elem.Type == ibmmq.MQCFT_INTEGER64_LIST {
			usableValue = true
		}

		if usableValue {
			for attr, _ := range QueueStatus.Attributes {
				if QueueStatus.Attributes[attr].pcfAttr == elem.Parameter {
					index := QueueStatus.Attributes[attr].index
					if index == -1 {
						v := elem.Int64Value[0]
						if QueueStatus.Attributes[attr].delta {
							// If we have already got a value for this attribute and queue
							// then use it to create the delta. Otherwise make the initial
							// value 0.
							if prevVal, ok := QueueStatus.Attributes[attr].prevValues[key]; ok {
								QueueStatus.Attributes[attr].Values[key] = newStatusValueInt64(v - prevVal)
							} else {
								QueueStatus.Attributes[attr].Values[key] = newStatusValueInt64(0)
							}
							QueueStatus.Attributes[attr].prevValues[key] = v
						} else {
							// Return the actual number
							QueueStatus.Attributes[attr].Values[key] = newStatusValueInt64(v)
						}
					} else {
						v := elem.Int64Value
						if QueueStatus.Attributes[attr].delta {
							// If we have already got a value for this attribute and queue
							// then use it to create the delta. Otherwise make the initial
							// value 0.
							if prevVal, ok := QueueStatus.Attributes[attr].prevValues[key]; ok {
								QueueStatus.Attributes[attr].Values[key] = newStatusValueInt64(v[index] - prevVal)
							} else {
								QueueStatus.Attributes[attr].Values[key] = newStatusValueInt64(0)
							}
							QueueStatus.Attributes[attr].prevValues[key] = v[index]
						} else {
							// Return the actual number
							QueueStatus.Attributes[attr].Values[key] = newStatusValueInt64(v[index])
						}
					}
				}
			}
		}
	}

	return key
}

// Return a standardised value. If the attribute indicates that something
// special has to be done, then do that. Otherwise just make sure it's a non-negative
// value of the correct datatype
func QueueNormalise(attr *StatusAttribute, v int64) float64 {
	var f float64

	if attr.squash {
		switch attr.pcfAttr {
		// No queue status values need squashing right now
		default:
			f = float64(v)
			if f < 0 {
				f = 0
			}
		}
	} else {
		f = float64(v)
		if f < 0 {
			f = 0
		}
	}
	return f
}
