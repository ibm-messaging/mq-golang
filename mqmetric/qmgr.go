/*
Package mqmetric contains a set of routines common to several
commands used to export MQ metrics to different backend
storage mechanisms including Prometheus and InfluxDB.
*/
package mqmetric

/*
  Copyright (c) IBM Corporation 2018,2019

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
Functions in this file use the DISPLAY QMSTATUS command to extract metrics
about the MQ queue manager
*/

import (
	"github.com/ibm-messaging/mq-golang/ibmmq"
	"strings"
	"time"
)

const (
	ATTR_QMGR_NAME              = "name"
	ATTR_QMGR_CONNECTION_COUNT  = "connection_count"
	ATTR_QMGR_CHINIT_STATUS     = "channel_initiator_status"
	ATTR_QMGR_CMD_SERVER_STATUS = "command_server_status"
	ATTR_QMGR_STATUS            = "status"
	ATTR_QMGR_UPTIME            = "uptime"
)

var QueueManagerStatus StatusSet
var qMgrAttrsInit = false

/*
Unlike the statistics produced via a topic, there is no discovery
of the attributes available in object STATUS queries. There is also
no discovery of descriptions for them. So this function hardcodes the
attributes we are going to look for and gives the associated descriptive
text. The elements can be expanded later; just trying to give a starting point
for now.
*/
func QueueManagerInitAttributes() {
	if qMgrAttrsInit {
		return
	}
	QueueManagerStatus.Attributes = make(map[string]*StatusAttribute)

	attr := ATTR_QMGR_NAME
	QueueManagerStatus.Attributes[attr] = newPseudoStatusAttribute(attr, "Queue Manager Name")

	attr = ATTR_QMGR_UPTIME
	QueueManagerStatus.Attributes[attr] = newStatusAttribute(attr, "Up time", -1)

	// These are the integer status fields that are of interest
	attr = ATTR_QMGR_CONNECTION_COUNT
	QueueManagerStatus.Attributes[attr] = newStatusAttribute(attr, "Connection Count", ibmmq.MQIACF_CONNECTION_COUNT)
	attr = ATTR_QMGR_CHINIT_STATUS
	QueueManagerStatus.Attributes[attr] = newStatusAttribute(attr, "Channel Initiator Status", ibmmq.MQIACF_CHINIT_STATUS)
	attr = ATTR_QMGR_CMD_SERVER_STATUS
	QueueManagerStatus.Attributes[attr] = newStatusAttribute(attr, "Command Server Status", ibmmq.MQIACF_CMD_SERVER_STATUS)

	// The qmgr status is pointless - if we can't connect to the qmgr, then we can't report on it. And if we can, it's up.
	// I'll leave this in as a reminder of why it's not being collected.
	//attr = ATTR_QMGR_STATUS
	//QueueManagerStatus.Attributes[attr] = newStatusAttribute(attr, "Queue Manager Status", ibmmq.MQIACF_Q_MGR_STATUS)

	qMgrAttrsInit = true
}

func CollectQueueManagerStatus() error {
	var err error

	QueueManagerInitAttributes()

	// Empty any collected values
	for k := range QueueManagerStatus.Attributes {
		QueueManagerStatus.Attributes[k].Values = make(map[string]*StatusValue)
	}

	err = collectQueueManagerStatus(ibmmq.MQOT_Q_MGR)

	return err

}

// Issue the INQUIRE_Q_MGR_STATUS command for a queue or wildcarded queue name
// Collect the responses and build up the statistics
func collectQueueManagerStatus(instanceType int32) error {
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
	cfh.Command = ibmmq.MQCMD_INQUIRE_Q_MGR_STATUS

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
			// Returned by z/OS qmgrs but are not interesting
			if cfh.Type == ibmmq.MQCFT_XR_SUMMARY || cfh.Type == ibmmq.MQCFT_XR_MSG {
				continue
			}
			parseQMgrData(instanceType, cfh, replyBuf[offset:datalen])

		}
	}

	return err
}

// Given a PCF response message, parse it to extract the desired statistics
func parseQMgrData(instanceType int32, cfh *ibmmq.MQCFH, buf []byte) string {
	var elem *ibmmq.PCFParameter

	qMgrName := ""
	key := ""

	startTime := ""
	startDate := ""

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
		case ibmmq.MQCA_Q_MGR_NAME:
			qMgrName = strings.TrimSpace(elem.String[0])
		}
	}

	// Create a unique key for this instance
	key = qMgrName

	QueueManagerStatus.Attributes[ATTR_QMGR_NAME].Values[key] = newStatusValueString(qMgrName)

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
			for attr, _ := range QueueManagerStatus.Attributes {
				if QueueManagerStatus.Attributes[attr].pcfAttr == elem.Parameter {
					index := QueueManagerStatus.Attributes[attr].index
					if index == -1 {
						v := elem.Int64Value[0]
						if QueueManagerStatus.Attributes[attr].delta {
							// If we have already got a value for this attribute and queue
							// then use it to create the delta. Otherwise make the initial
							// value 0.
							if prevVal, ok := QueueManagerStatus.Attributes[attr].prevValues[key]; ok {
								QueueManagerStatus.Attributes[attr].Values[key] = newStatusValueInt64(v - prevVal)
							} else {
								QueueManagerStatus.Attributes[attr].Values[key] = newStatusValueInt64(0)
							}
							QueueManagerStatus.Attributes[attr].prevValues[key] = v
						} else {
							// Return the actual number
							QueueManagerStatus.Attributes[attr].Values[key] = newStatusValueInt64(v)
						}
					} else {
						v := elem.Int64Value
						if QueueManagerStatus.Attributes[attr].delta {
							// If we have already got a value for this attribute and queue
							// then use it to create the delta. Otherwise make the initial
							// value 0.
							if prevVal, ok := QueueManagerStatus.Attributes[attr].prevValues[key]; ok {
								QueueManagerStatus.Attributes[attr].Values[key] = newStatusValueInt64(v[index] - prevVal)
							} else {
								QueueManagerStatus.Attributes[attr].Values[key] = newStatusValueInt64(0)
							}
							QueueManagerStatus.Attributes[attr].prevValues[key] = v[index]
						} else {
							// Return the actual number
							QueueManagerStatus.Attributes[attr].Values[key] = newStatusValueInt64(v[index])
						}
					}
				}
			}
		} else {
			switch elem.Parameter {
			case ibmmq.MQCACF_Q_MGR_START_TIME:
				startTime = strings.TrimSpace(elem.String[0])
			case ibmmq.MQCACF_Q_MGR_START_DATE:
				startDate = strings.TrimSpace(elem.String[0])
			}
		}
	}

	now := time.Now()
	QueueManagerStatus.Attributes[ATTR_QMGR_UPTIME].Values[key] = newStatusValueInt64(statusTimeDiff(now, startDate, startTime))

	return key
}

// Return a standardised value. If the attribute indicates that something
// special has to be done, then do that. Otherwise just make sure it's a non-negative
// value of the correct datatype
func QueueManagerNormalise(attr *StatusAttribute, v int64) float64 {
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
