/*
Package mqmetric contains a set of routines common to several
commands used to export MQ metrics to different backend
storage mechanisms including Prometheus and InfluxDB.
*/
package mqmetric

/*
  Copyright (c) IBM Corporation 2018,2020

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
	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
	"strings"
	"time"
)

const (
	ATTR_QMGR_NAME                = "name"
	ATTR_QMGR_CONNECTION_COUNT    = "connection_count"
	ATTR_QMGR_CHINIT_STATUS       = "channel_initiator_status"
	ATTR_QMGR_CMD_SERVER_STATUS   = "command_server_status"
	ATTR_QMGR_STATUS              = "status"
	ATTR_QMGR_UPTIME              = "uptime"
	ATTR_QMGR_MAX_CHANNELS        = "max_channels"
	ATTR_QMGR_MAX_ACTIVE_CHANNELS = "max_active_channels"
	ATTR_QMGR_MAX_TCP_CHANNELS    = "max_tcp_channels"
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

	traceEntry("QueueManagerInitAttributes")
	if qMgrAttrsInit {
		traceExit("QueueManagerInitAttributes", 1)
		return
	}
	QueueManagerStatus.Attributes = make(map[string]*StatusAttribute)

	attr := ATTR_QMGR_NAME
	QueueManagerStatus.Attributes[attr] = newPseudoStatusAttribute(attr, "Queue Manager Name")

	if GetPlatform() != ibmmq.MQPL_ZOS {
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
	} else {
		attr = ATTR_QMGR_MAX_CHANNELS
		QueueManagerStatus.Attributes[attr] = newStatusAttribute(attr, "Max Channels", -1)
		attr = ATTR_QMGR_MAX_TCP_CHANNELS
		QueueManagerStatus.Attributes[attr] = newStatusAttribute(attr, "Max TCP Channels", -1)
		attr = ATTR_QMGR_MAX_ACTIVE_CHANNELS
		QueueManagerStatus.Attributes[attr] = newStatusAttribute(attr, "Max Active Channels", -1)
	}

	qMgrAttrsInit = true

	traceExit("QueueManagerInitAttributes", 0)

}

func CollectQueueManagerStatus() error {
	var err error

	traceEntry("CollectQueueManagerStatus")
	QueueManagerInitAttributes()
	for k := range QueueManagerStatus.Attributes {
		QueueManagerStatus.Attributes[k].Values = make(map[string]*StatusValue)
	}

	// Empty any collected values
	if GetPlatform() == ibmmq.MQPL_ZOS {
		err = collectQueueManagerAttrs()
	} else {
		err = collectQueueManagerStatus(ibmmq.MQOT_Q_MGR)
	}

	traceExitErr("CollectQueueManagerStatus", 0, err)

	return err

}

// On z/OS there are a couple of static-ish values that might be helpful.
// They can be obtained via MQINQ and do not need a PCF flow.
// We can't get these on Distributed because equivalents are in qm.ini
func collectQueueManagerAttrs() error {

	traceEntry("collectQueueManagerAttrs")

	selectors := []int32{ibmmq.MQCA_Q_MGR_NAME,
		ibmmq.MQIA_ACTIVE_CHANNELS,
		ibmmq.MQIA_TCP_CHANNELS,
		ibmmq.MQIA_MAX_CHANNELS}

	v, err := qMgrObject.Inq(selectors)
	if err == nil {
		maxchls := v[ibmmq.MQIA_MAX_CHANNELS].(int32)
		maxact := v[ibmmq.MQIA_ACTIVE_CHANNELS].(int32)
		maxtcp := v[ibmmq.MQIA_TCP_CHANNELS].(int32)
		key := v[ibmmq.MQCA_Q_MGR_NAME].(string)
		QueueManagerStatus.Attributes[ATTR_QMGR_MAX_ACTIVE_CHANNELS].Values[key] = newStatusValueInt64(int64(maxact))
		QueueManagerStatus.Attributes[ATTR_QMGR_MAX_CHANNELS].Values[key] = newStatusValueInt64(int64(maxchls))
		QueueManagerStatus.Attributes[ATTR_QMGR_MAX_TCP_CHANNELS].Values[key] = newStatusValueInt64(int64(maxtcp))
		QueueManagerStatus.Attributes[ATTR_QMGR_NAME].Values[key] = newStatusValueString(key)

	}
	traceExitErr("collectQueueManagerAttrs", 0, err)

	return err
}

// Issue the INQUIRE_Q_MGR_STATUS command for the queue mgr.
// Collect the responses and build up the statistics
func collectQueueManagerStatus(instanceType int32) error {
	var err error

	traceEntry("collectQueueManagerStatus")

	statusClearReplyQ()
	putmqmd, pmo, cfh, buf := statusSetCommandHeaders()

	// Can allow all the other fields to default
	cfh.Command = ibmmq.MQCMD_INQUIRE_Q_MGR_STATUS

	// Once we know the total number of parameters, put the
	// CFH header on the front of the buffer.
	buf = append(cfh.Bytes(), buf...)

	// And now put the command to the queue
	err = cmdQObj.Put(putmqmd, pmo, buf)
	if err != nil {
		traceExitErr("collectQueueManagerStatus", 1, err)
		return err
	}

	// Now get the responses - loop until all have been received (one
	// per queue) or we run out of time
	for allReceived := false; !allReceived; {
		cfh, buf, allReceived, err = statusGetReply()
		if buf != nil {
			parseQMgrData(instanceType, cfh, buf)
		}
	}

	traceExitErr("collectQueueManagerStatus", 0, err)
	return err
}

// Given a PCF response message, parse it to extract the desired statistics
func parseQMgrData(instanceType int32, cfh *ibmmq.MQCFH, buf []byte) string {
	var elem *ibmmq.PCFParameter

	traceEntry("parseQMgrData")
	qMgrName := ""
	key := ""

	startTime := ""
	startDate := ""

	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh == nil || cfh.ParameterCount == 0 {
		traceExit("parseQMgrData", 1)
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

		if !statusGetIntAttributes(QueueManagerStatus, elem, key) {
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

	traceExitF("parseQMgrData", 0, "Key: %s", key)
	return key
}

// Return a standardised value. If the attribute indicates that something
// special has to be done, then do that. Otherwise just make sure it's a non-negative
// value of the correct datatype
func QueueManagerNormalise(attr *StatusAttribute, v int64) float64 {
	return statusNormalise(attr, v)
}
