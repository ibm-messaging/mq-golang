package mqmetric

/*
  Copyright (c) IBM Corporation 2018,2026

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
	"strconv"
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
	ATTR_QMGR_ACTIVE_SERVICES     = "active_services"

	// Some of the log-related metrics are effectively duplicated between QMSTATUS and
	// published resources eg LOGUTIL. We prefer the publication versions so do not
	// explicitly call them out here. We also do not collect "static" logger configuration
	// values such as LOGEXTSZ, LOGPRIM or LOGTYPE.
	ATTR_QMGR_LOG_CURRENT_EXTENT = "log_extent_current"
	ATTR_QMGR_LOG_MEDIA_EXTENT   = "log_extent_media"
	ATTR_QMGR_LOG_ARCHIVE_EXTENT = "log_extent_archive"
	ATTR_QMGR_LOG_RESTART_EXTENT = "log_extent_restart"

	ATTR_QMGR_LOG_MEDIA_SIZE    = "log_size_media"
	ATTR_QMGR_LOG_ARCHIVE_SIZE  = "log_size_archive"
	ATTR_QMGR_LOG_RESTART_SIZE  = "log_size_restart"
	ATTR_QMGR_LOG_REUSABLE_SIZE = "log_size_reusable"
	ATTR_QMGR_LOG_START         = "log_start_epoch"
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
	ot := OT_Q_MGR
	ci := getConnection(GetConnectionKey())
	os := &ci.objectStatus[ot]
	st := GetObjectStatus(GetConnectionKey(), ot)
	if os.init {
		traceExit("QueueManagerInitAttributes", 1)
		return
	}

	st.Attributes = make(map[string]*StatusAttribute)

	newPseudoStatusMapEntryRequired(st, ot, ATTR_QMGR_NAME, "Queue Manager Name")

	if GetPlatform() != ibmmq.MQPL_ZOS {
		newStatusMapEntry(st, ot, ATTR_QMGR_UPTIME, "Up time", DUMMY_PCFATTR, false)

		// These are the integer status fields that are of interest
		newStatusMapEntry(st, ot, ATTR_QMGR_CONNECTION_COUNT, "Connection Count", ibmmq.MQIACF_CONNECTION_COUNT, false)
		newStatusMapEntry(st, ot, ATTR_QMGR_CHINIT_STATUS, "Channel Initiator Status", ibmmq.MQIACF_CHINIT_STATUS, false)
		newStatusMapEntry(st, ot, ATTR_QMGR_CMD_SERVER_STATUS, "Command Server Status", ibmmq.MQIACF_CMD_SERVER_STATUS, false)
		newStatusMapEntry(st, ot, ATTR_QMGR_ACTIVE_LISTENERS, "Active Listener Count", DUMMY_PCFATTR, false)
		newStatusMapEntry(st, ot, ATTR_QMGR_ACTIVE_SERVICES, "Active Service Count", DUMMY_PCFATTR, false)

		// Log-related metrics
		newStatusMapEntry(st, ot, ATTR_QMGR_LOG_CURRENT_EXTENT, "Log Current Extent", DUMMY_PCFATTR, false)
		newStatusMapEntry(st, ot, ATTR_QMGR_LOG_MEDIA_EXTENT, "Log Media Extent", DUMMY_PCFATTR, false)
		newStatusMapEntry(st, ot, ATTR_QMGR_LOG_ARCHIVE_EXTENT, "Log Archive Extent", DUMMY_PCFATTR, false)
		newStatusMapEntry(st, ot, ATTR_QMGR_LOG_RESTART_EXTENT, "Log Restart Recovery Extent", DUMMY_PCFATTR, false)

		newStatusMapEntry(st, ot, ATTR_QMGR_LOG_MEDIA_SIZE, "Log Media Size", ibmmq.MQIACF_MEDIA_LOG_SIZE, false)
		newStatusMapEntry(st, ot, ATTR_QMGR_LOG_ARCHIVE_SIZE, "Log Archive Size", ibmmq.MQIACF_ARCHIVE_LOG_SIZE, false)
		newStatusMapEntry(st, ot, ATTR_QMGR_LOG_RESTART_SIZE, "Log Restart Recovery Size", ibmmq.MQIACF_RESTART_LOG_SIZE, false)
		newStatusMapEntry(st, ot, ATTR_QMGR_LOG_REUSABLE_SIZE, "Log Reusable Size", ibmmq.MQIACF_REUSABLE_LOG_SIZE, false)

		newStatusMapEntry(st, ot, ATTR_QMGR_LOG_START, "Log Start Time (epoch ms)", DUMMY_PCFATTR, false)

	} else {
		newStatusMapEntry(st, ot, ATTR_QMGR_MAX_CHANNELS, "Max Channels", DUMMY_PCFATTR, false)
		newStatusMapEntry(st, ot, ATTR_QMGR_MAX_TCP_CHANNELS, "Max TCP Channels", DUMMY_PCFATTR, false)
		newStatusMapEntry(st, ot, ATTR_QMGR_MAX_ACTIVE_CHANNELS, "Max Active Channels", DUMMY_PCFATTR, false)
	}

	// The qmgr status is reported to Prometheus with some pseudo-values so we can see if
	// we are not actually connected. On other collectors, the whole collection process is
	// halted so this would not be reported.
	newStatusMapEntryRequired(st, ot, ATTR_QMGR_STATUS, "Queue Manager Status", ibmmq.MQIACF_Q_MGR_STATUS)

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
			err = collectQueueManagerServices()
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

	if ci.showCustomAttribute {
		selectors = append(selectors, ibmmq.MQCA_CUSTOM)
	}

	v, err := ci.si.qMgrObject.Inq(selectors)
	if err == nil {
		maxchls := v[ibmmq.MQIA_MAX_CHANNELS].(int32)
		maxact := v[ibmmq.MQIA_ACTIVE_CHANNELS].(int32)
		maxtcp := v[ibmmq.MQIA_TCP_CHANNELS].(int32)
		desc := v[ibmmq.MQCA_Q_MGR_DESC].(string)

		key := v[ibmmq.MQCA_Q_MGR_NAME].(string)

		a, ok := st.Attributes[ATTR_QMGR_MAX_ACTIVE_CHANNELS]
		if ok {
			a.Values[key] = newStatusValueInt64(int64(maxact))
		}
		a, ok = st.Attributes[ATTR_QMGR_MAX_CHANNELS]
		if ok {
			a.Values[key] = newStatusValueInt64(int64(maxchls))
		}
		a, ok = st.Attributes[ATTR_QMGR_MAX_TCP_CHANNELS]
		if ok {
			a.Values[key] = newStatusValueInt64(int64(maxtcp))
		}
		a, ok = st.Attributes[ATTR_QMGR_NAME]
		if ok {
			a.Values[key] = newStatusValueString(key)
		}
		// This pseudo-value will always get filled in for a z/OS qmgr - we know it's running because
		// we've been able to connect!
		a, ok = st.Attributes[ATTR_QMGR_STATUS]
		if ok {
			a.Values[key] = newStatusValueInt64(int64(ibmmq.MQQMSTA_RUNNING))
		}
		qMgrInfo.Description = desc
		qMgrInfo.QMgrName = key
		if ci.showCustomAttribute {
			qMgrInfo.Custom = v[ibmmq.MQCA_CUSTOM].(string)
		}
	}
	traceExitErr("collectQueueManagerAttrsZOS", 0, err)

	return err
}

func collectQueueManagerAttrsDist() error {

	traceEntry("collectQueueManagerAttrsDist")
	ci := getConnection(GetConnectionKey())
	st := GetObjectStatus(GetConnectionKey(), OT_Q_MGR)

	selectors := []int32{ibmmq.MQCA_Q_MGR_NAME,
		ibmmq.MQCA_Q_MGR_DESC, ibmmq.MQCA_CUSTOM}

	v, err := ci.si.qMgrObject.Inq(selectors)
	desc := DUMMY_STRING
	custom := DUMMY_STRING

	if err == nil {
		key := v[ibmmq.MQCA_Q_MGR_NAME].(string)
		desc = v[ibmmq.MQCA_Q_MGR_DESC].(string)
		custom = v[ibmmq.MQCA_CUSTOM].(string)

		st.Attributes[ATTR_QMGR_NAME].Values[key] = newStatusValueString(key)
		qMgrInfo.Description = desc
		qMgrInfo.QMgrName = key
		qMgrInfo.Custom = custom
	}

	traceExitErr("collectQueueManagerAttrsDist", 0, err)

	return err
}

// We collect the number of active listeners, rather than
// enumerating the status of all of the configured objects. In most
// systems, the listener count will be "1". And getting all of the information
// about all objects is probably overkill. This does assume that
// listeners are managed through the listener objects, rather than
// being started independently eg by direct use of the runmqlsr command.
func collectQueueManagerListeners() error {
	var err error

	traceEntry("collectQueueManagerListeners")

	listenerCount := 0

	ci := getConnection(GetConnectionKey())
	st := GetObjectStatus(GetConnectionKey(), OT_Q_MGR)
	statusClearReplyQ()
	putmqmd, pmo, cfh, buf := statusSetCommandHeaders()
	// Can allow all the other fields to default
	// Only active or transitioning listeners return a response.
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
		cfh, buf, allReceived, err = statusGetReply(putmqmd.MsgId)
		if buf != nil {
			if parseQMgrActiveProcesses(cfh, buf) {
				listenerCount++
			}
		}
	}

	logDebug("Getting listener count for %s as %d", qMgrInfo.QMgrName, listenerCount)

	if qMgrInfo.QMgrName != "" {
		v, ok := st.Attributes[ATTR_QMGR_ACTIVE_LISTENERS]
		if ok {
			v.Values[qMgrInfo.QMgrName] = newStatusValueInt64(int64(listenerCount))
		}
	}

	traceExitErr("collectQueueManagerListeners", 0, err)

	return err
}

// We collect the number of active services. The details of
// the services are not suitable for metrics, but the total number might be interesting.
// "Active" includes the starting/stopping states that might be reported.
func collectQueueManagerServices() error {
	var err error

	traceEntry("collectQueueManagerServices")

	serviceCount := 0

	ci := getConnection(GetConnectionKey())
	st := GetObjectStatus(GetConnectionKey(), OT_Q_MGR)
	statusClearReplyQ()
	putmqmd, pmo, cfh, buf := statusSetCommandHeaders()
	// Can allow all the other fields to default
	// Only active or transitioning listeners return a response.
	cfh.Command = ibmmq.MQCMD_INQUIRE_SERVICE_STATUS

	// Add the parameters one at a time into a buffer
	pcfparm := new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_STRING
	pcfparm.Parameter = ibmmq.MQCA_SERVICE_NAME
	pcfparm.String = []string{"*"}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	// Once we know the total number of parameters, put the
	// CFH header on the front of the buffer.
	buf = append(cfh.Bytes(), buf...)

	// And now put the command to the queue
	err = ci.si.cmdQObj.Put(putmqmd, pmo, buf)
	if err != nil {
		traceExitErr("collectQueueManagerServices", 1, err)
		return err
	}

	// Now get the responses - loop until all have been received (one
	// per queue) or we run out of time
	for allReceived := false; !allReceived; {
		cfh, buf, allReceived, err = statusGetReply(putmqmd.MsgId)
		if buf != nil {
			if parseQMgrActiveProcesses(cfh, buf) {
				serviceCount++
			}
		}
	}

	logDebug("Getting service count for %s as %d", qMgrInfo.QMgrName, serviceCount)

	if qMgrInfo.QMgrName != "" {
		v, ok := st.Attributes[ATTR_QMGR_ACTIVE_SERVICES]
		if ok {
			v.Values[qMgrInfo.QMgrName] = newStatusValueInt64(int64(serviceCount))
		}
	}

	traceExitErr("collectQueueManagerServices", 0, err)

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
		cfh, buf, allReceived, err = statusGetReply(putmqmd.MsgId)
		if buf != nil {
			parseQMgrStatusData(instanceType, cfh, buf)
		}
	}

	traceExitErr("collectQueueManagerStatus", 0, err)
	return err
}

// Given a PCF response message, parse it to extract the desired statistics
func parseQMgrStatusData(instanceType int32, cfh *ibmmq.MQCFH, buf []byte) string {
	var elem *ibmmq.PCFParameter

	traceEntry("parseQMgrStatusData")

	st := GetObjectStatus(GetConnectionKey(), OT_Q_MGR)

	qMgrName := ""
	key := ""

	startTime := ""
	startDate := ""
	logStartTime := ""
	logStartDate := ""

	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh == nil || cfh.ParameterCount == 0 {
		traceExit("parseQMgrStatusData", 1)
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
	hostname := DUMMY_STRING
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
			case ibmmq.MQCACF_HOST_NAME: // This started to be available from 9.3.2
				hostname = strings.TrimSpace(elem.String[0])

			// Log-related attributes naming an extent will need conversion from a string to an integer
			case ibmmq.MQCACF_CURRENT_LOG_EXTENT_NAME:
				v, ok := st.Attributes[ATTR_QMGR_LOG_CURRENT_EXTENT]
				if ok {
					v.Values[key] = newStatusValueInt64(logExtent(elem.String[0]))
				}
			case ibmmq.MQCACF_MEDIA_LOG_EXTENT_NAME:
				v, ok := st.Attributes[ATTR_QMGR_LOG_MEDIA_EXTENT]
				if ok {
					v.Values[key] = newStatusValueInt64(logExtent(elem.String[0]))
				}
			case ibmmq.MQCACF_ARCHIVE_LOG_EXTENT_NAME:
				v, ok := st.Attributes[ATTR_QMGR_LOG_ARCHIVE_EXTENT]
				if ok {
					v.Values[key] = newStatusValueInt64(logExtent(elem.String[0]))
				}
			case ibmmq.MQCACF_RESTART_LOG_EXTENT_NAME:
				v, ok := st.Attributes[ATTR_QMGR_LOG_RESTART_EXTENT]
				if ok {
					v.Values[key] = newStatusValueInt64(logExtent(elem.String[0]))
				}
			case ibmmq.MQCACF_LOG_START_TIME:
				logStartTime = strings.TrimSpace(elem.String[0])
			case ibmmq.MQCACF_LOG_START_DATE:
				logStartDate = strings.TrimSpace(elem.String[0])
			}
		}
	}

	v, ok := st.Attributes[ATTR_QMGR_UPTIME]
	if ok {
		now := time.Now()
		v.Values[key] = newStatusValueInt64(statusTimeDiff(now, startDate, startTime))
	}
	qMgrInfo.HostName = hostname

	v, ok = st.Attributes[ATTR_QMGR_LOG_START]
	if ok {
		epoch := statusTimeEpoch(logStartDate, logStartTime)
		v.Values[key] = newStatusValueInt64(epoch)
	}

	traceExitF("parseQMgrStatusData", 0, "Key: %s", key)
	return key
}

// Given a PCF response message, parse it to extract the desired statistics
func parseQMgrActiveProcesses(cfh *ibmmq.MQCFH, buf []byte) bool {
	//var elem *ibmmq.PCFParameter

	traceEntry("parseQMgrActiveProcesses")
	process := false

	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh == nil || cfh.ParameterCount == 0 {
		traceExit("parseQMgrActiveProcesses", 1)
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
		process = true
	}

	traceExitF("parseQMgrActiveProcesses", 0, "active: %v", process)
	return process
}

// A log extent is reported by the qmgr with a name like "S001234.LOG". We
// extract the numeric part here so it can be returned like a regular metric.
// If the extent doesn't match that format (likely an empty string for CIRCULAR logging
// systems) then just return 0.
func logExtent(l string) int64 {
	l = strings.ToUpper(l)
	if strings.HasPrefix(l, "S") && strings.HasSuffix(l, ".LOG") {
		l = strings.Replace(strings.Replace(l, "S", "", -1), ".LOG", "", -1)
		v, err := strconv.Atoi(l)
		if err == nil {
			return int64(v)
		}
	}
	return 0
}

// Return a standardised value. If the attribute indicates that something
// special has to be done, then do that. Otherwise just make sure it's a non-negative
// value of the correct datatype
func QueueManagerNormalise(attr *StatusAttribute, v int64) float64 {
	switch attr.pcfAttr {
	// The logger size values are reported in MB by the qmgr to keep them in MQCFIN range. We normalise them to bytes here
	case ibmmq.MQIACF_MEDIA_LOG_SIZE,
		ibmmq.MQIACF_RESTART_LOG_SIZE,
		ibmmq.MQIACF_ARCHIVE_LOG_SIZE,
		ibmmq.MQIACF_REUSABLE_LOG_SIZE:
		f := float64(v) * 1024 * 1024
		if f < 0 {
			f = 0
		}
		return f
	default:
		return statusNormalise(attr, v)
	}
}

// Return the nominated MQCA* attribute from the object's attributes
// stored in the map. The "key" is unused for now, but might be useful
// if we do a version that supports connections to multiple qmgrs. And it keeps
// the function looking like the equivalent for the Queue query.
func GetQueueManagerAttribute(key string, attribute int32) string {
	v := DUMMY_STRING

	switch attribute {
	case ibmmq.MQCACF_HOST_NAME:
		v = qMgrInfo.HostName
		v = strings.ReplaceAll(v, "-", ".")
	default:
		v = DUMMY_STRING
	}
	v = strings.TrimSpace(v)

	if v == "" {
		v = DUMMY_STRING
	}
	return v
}
