/*
Package mqmetric contains a set of routines common to several
commands used to export MQ metrics to different backend
storage mechanisms including Prometheus and InfluxDB.
*/
package mqmetric

/*
  Copyright (c) IBM Corporation 2018,2023

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
	"strings"
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
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
	ATTR_QMGR_ACTIVE_LISTENERS    = "active_listeners"
)

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
	ci := getConnection(GetConnectionKey())
	os := &ci.objectStatus[OT_Q_MGR]
	st := GetObjectStatus(GetConnectionKey(), OT_Q_MGR)
	if os.init {
		traceExit("QueueManagerInitAttributes", 1)
		return
	}
	st.Attributes = make(map[string]*StatusAttribute)

	attr := ATTR_QMGR_NAME
	st.Attributes[attr] = newPseudoStatusAttribute(attr, "Queue Manager Name")

	if GetPlatform() != ibmmq.MQPL_ZOS {
		attr = ATTR_QMGR_UPTIME
		st.Attributes[attr] = newStatusAttribute(attr, "Up time", -1)

		// These are the integer status fields that are of interest
		attr = ATTR_QMGR_CONNECTION_COUNT
		st.Attributes[attr] = newStatusAttribute(attr, "Connection Count", ibmmq.MQIACF_CONNECTION_COUNT)
		attr = ATTR_QMGR_CHINIT_STATUS
		st.Attributes[attr] = newStatusAttribute(attr, "Channel Initiator Status", ibmmq.MQIACF_CHINIT_STATUS)
		attr = ATTR_QMGR_CMD_SERVER_STATUS
		st.Attributes[attr] = newStatusAttribute(attr, "Command Server Status", ibmmq.MQIACF_CMD_SERVER_STATUS)
		attr = ATTR_QMGR_ACTIVE_LISTENERS
		st.Attributes[attr] = newStatusAttribute(attr, "Active Listener Count", -1)
	} else {
		attr = ATTR_QMGR_MAX_CHANNELS
		st.Attributes[attr] = newStatusAttribute(attr, "Max Channels", -1)
		attr = ATTR_QMGR_MAX_TCP_CHANNELS
		st.Attributes[attr] = newStatusAttribute(attr, "Max TCP Channels", -1)
		attr = ATTR_QMGR_MAX_ACTIVE_CHANNELS
		st.Attributes[attr] = newStatusAttribute(attr, "Max Active Channels", -1)
	}

	// The qmgr status is reported to Prometheus with some pseudo-values so we can see if
	// we are not actually connected. On other collectors, the whole collection process is
	// halted so this would not be reported.
	attr = ATTR_QMGR_STATUS
	st.Attributes[attr] = newStatusAttribute(attr, "Queue Manager Status", ibmmq.MQIACF_Q_MGR_STATUS)

	os.init = true

	traceExit("QueueManagerInitAttributes", 0)

}

func CollectQueueManagerStatus() error {
	var err error

	traceEntry("CollectQueueManagerStatus")
	//os := &ci.objectStatus[OT_Q_MGR]
	st := GetObjectStatus(GetConnectionKey(), OT_Q_MGR)

	// Empty any collected values
	QueueManagerInitAttributes()
	for k := range st.Attributes {
		st.Attributes[k].Values = make(map[string]*StatusValue)
	}

	if GetPlatform() == ibmmq.MQPL_ZOS {
		err = collectQueueManagerAttrsZOS()
	} else {
		err = collectQueueManagerAttrsDist()
		if err == nil {
			err = collectQueueManagerListeners()
		}
		if err == nil {
			err = collectQueueManagerStatus(ibmmq.MQOT_Q_MGR)
		}
	}

	traceExitErr("CollectQueueManagerStatus", 0, err)

	return err

}

// On z/OS there are a couple of static-ish values that might be helpful.
// They can be obtained via MQINQ and do not need a PCF flow.
// We can't get these on Distributed because equivalents are in qm.ini
func collectQueueManagerAttrsZOS() error {

	traceEntry("collectQueueManagerAttrsZOS")
	ci := getConnection(GetConnectionKey())
	st := GetObjectStatus(GetConnectionKey(), OT_Q_MGR)

	selectors := []int32{ibmmq.MQCA_Q_MGR_NAME,
		ibmmq.MQCA_Q_MGR_DESC,
		ibmmq.MQIA_ACTIVE_CHANNELS,
		ibmmq.MQIA_TCP_CHANNELS,
		ibmmq.MQIA_MAX_CHANNELS}

	v, err := ci.si.qMgrObject.Inq(selectors)
	if err == nil {
		maxchls := v[ibmmq.MQIA_MAX_CHANNELS].(int32)
		maxact := v[ibmmq.MQIA_ACTIVE_CHANNELS].(int32)
		maxtcp := v[ibmmq.MQIA_TCP_CHANNELS].(int32)
		desc := v[ibmmq.MQCA_Q_MGR_DESC].(string)

		key := v[ibmmq.MQCA_Q_MGR_NAME].(string)
		st.Attributes[ATTR_QMGR_MAX_ACTIVE_CHANNELS].Values[key] = newStatusValueInt64(int64(maxact))
		st.Attributes[ATTR_QMGR_MAX_CHANNELS].Values[key] = newStatusValueInt64(int64(maxchls))
		st.Attributes[ATTR_QMGR_MAX_TCP_CHANNELS].Values[key] = newStatusValueInt64(int64(maxtcp))
		st.Attributes[ATTR_QMGR_NAME].Values[key] = newStatusValueString(key)
		// This pseudo-value will always get filled in for a z/OS qmgr - we know it's running because
		// we've been able to connect!
		st.Attributes[ATTR_QMGR_STATUS].Values[key] = newStatusValueInt64(int64(ibmmq.MQQMSTA_RUNNING))
		qMgrInfo.Description = desc
		qMgrInfo.QMgrName = key
	}
	traceExitErr("collectQueueManagerAttrsZOS", 0, err)

	return err
}

func collectQueueManagerAttrsDist() error {

	traceEntry("collectQueueManagerAttrsDist")
	ci := getConnection(GetConnectionKey())
	st := GetObjectStatus(GetConnectionKey(), OT_Q_MGR)

	selectors := []int32{ibmmq.MQCA_Q_MGR_NAME,
		ibmmq.MQCA_Q_MGR_DESC}

	v, err := ci.si.qMgrObject.Inq(selectors)
	desc := DUMMY_STRING
	if err == nil {
		key := v[ibmmq.MQCA_Q_MGR_NAME].(string)
		desc = v[ibmmq.MQCA_Q_MGR_DESC].(string)
		st.Attributes[ATTR_QMGR_NAME].Values[key] = newStatusValueString(key)
		qMgrInfo.Description = desc
		qMgrInfo.QMgrName = key
	}

	traceExitErr("collectQueueManagerAttrsDist", 0, err)

	return err
}

func collectQueueManagerListeners() error {
	var err error

	traceEntry("collectQueueManagerListeners")

	listenerCount := 0

	ci := getConnection(GetConnectionKey())
	st := GetObjectStatus(GetConnectionKey(), OT_Q_MGR)
	statusClearReplyQ()
	putmqmd, pmo, cfh, buf := statusSetCommandHeaders()
	// Can allow all the other fields to default
	cfh.Command = ibmmq.MQCMD_INQUIRE_LISTENER_STATUS

	// Add the parameters one at a time into a buffer
	pcfparm := new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_STRING
	pcfparm.Parameter = ibmmq.MQCACH_LISTENER_NAME
	pcfparm.String = []string{"*"}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	// Once we know the total number of parameters, put the
	// CFH header on the front of the buffer.
	buf = append(cfh.Bytes(), buf...)

	// And now put the command to the queue
	err = ci.si.cmdQObj.Put(putmqmd, pmo, buf)
	if err != nil {
		traceExitErr("collectQueueManagerListeners", 1, err)
		return err
	}

	// Now get the responses - loop until all have been received (one
	// per queue) or we run out of time
	for allReceived := false; !allReceived; {
		cfh, buf, allReceived, err = statusGetReply()
		if buf != nil {
			if parseQMgrListeners(cfh, buf) {
				listenerCount++
			}
		}
	}

	logDebug("Getting listener count for %s as %d", qMgrInfo.QMgrName, listenerCount)

	if qMgrInfo.QMgrName != "" {
		st.Attributes[ATTR_QMGR_ACTIVE_LISTENERS].Values[qMgrInfo.QMgrName] = newStatusValueInt64(int64(listenerCount))
	}

	traceExitErr("collectQueueManagerListeners", 0, err)

	return err
}

// Issue the INQUIRE_Q_MGR_STATUS command for the queue mgr.
// Collect the responses and build up the statistics
func collectQueueManagerStatus(instanceType int32) error {
	var err error

	traceEntry("collectQueueManagerStatus")
	ci := getConnection(GetConnectionKey())

	statusClearReplyQ()
	putmqmd, pmo, cfh, buf := statusSetCommandHeaders()

	// Can allow all the other fields to default
	cfh.Command = ibmmq.MQCMD_INQUIRE_Q_MGR_STATUS

	// Once we know the total number of parameters, put the
	// CFH header on the front of the buffer.
	buf = append(cfh.Bytes(), buf...)

	// And now put the command to the queue
	err = ci.si.cmdQObj.Put(putmqmd, pmo, buf)
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

	st := GetObjectStatus(GetConnectionKey(), OT_Q_MGR)

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

	st.Attributes[ATTR_QMGR_NAME].Values[key] = newStatusValueString(qMgrName)

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

		if !statusGetIntAttributes(GetObjectStatus(GetConnectionKey(), OT_Q_MGR), elem, key) {
			switch elem.Parameter {
			case ibmmq.MQCACF_Q_MGR_START_TIME:
				startTime = strings.TrimSpace(elem.String[0])
			case ibmmq.MQCACF_Q_MGR_START_DATE:
				startDate = strings.TrimSpace(elem.String[0])
			}
		}
	}

	now := time.Now()
	st.Attributes[ATTR_QMGR_UPTIME].Values[key] = newStatusValueInt64(statusTimeDiff(now, startDate, startTime))

	traceExitF("parseQMgrData", 0, "Key: %s", key)
	return key
}

// Given a PCF response message, parse it to extract the desired statistics
func parseQMgrListeners(cfh *ibmmq.MQCFH, buf []byte) bool {
	//var elem *ibmmq.PCFParameter

	traceEntry("parseQMgrListeners")
	listener := false

	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh == nil || cfh.ParameterCount == 0 {
		traceExit("parseQMgrListeners", 1)
		return false
	}

	// Parse it to look for successful queries
	for parmAvail && cfh.CompCode != ibmmq.MQCC_FAILED {
		_, bytesRead = ibmmq.ReadPCFParameter(buf[offset:])
		offset += bytesRead
		// Have we now reached the end of the message
		if offset >= datalen {
			parmAvail = false
		}
		listener = true
	}

	traceExitF("parseQMgrListeners", 0, "active: %v", listener)
	return listener
}

// Return a standardised value. If the attribute indicates that something
// special has to be done, then do that. Otherwise just make sure it's a non-negative
// value of the correct datatype
func QueueManagerNormalise(attr *StatusAttribute, v int64) float64 {
	return statusNormalise(attr, v)
}
