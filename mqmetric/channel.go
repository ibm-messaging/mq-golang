/*
Package mqmetric contains a set of routines common to several
commands used to export MQ metrics to different backend
storage mechanisms including Prometheus and InfluxDB.
*/
package mqmetric

/*
  Copyright (c) IBM Corporation 2016, 2019

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
Functions in this file use the DISPLAY CHSTATUS command to extract metrics
about running MQ channels
*/

import (
	"github.com/ibm-messaging/mq-golang/ibmmq"
	"regexp"
	"strings"
	"time"
)

const (
	ATTR_CHL_NAME     = "name"
	ATTR_CHL_CONNNAME = "connname"
	ATTR_CHL_JOBNAME  = "jobname"
	ATTR_CHL_RQMNAME  = "rqmname"

	ATTR_CHL_MESSAGES      = "messages"
	ATTR_CHL_STATUS        = "status"
	ATTR_CHL_STATUS_SQUASH = ATTR_CHL_STATUS + "_squash"
	ATTR_CHL_SUBSTATE      = "substate"
	ATTR_CHL_TYPE          = "type"
	ATTR_CHL_INSTANCE_TYPE = "instance_type"
	ATTR_CHL_SINCE_MSG     = "time_since_msg"

	ATTR_CHL_NETTIME_SHORT = "nettime_short"
	ATTR_CHL_NETTIME_LONG  = "nettime_long"

	SQUASH_CHL_STATUS_STOPPED    = 0
	SQUASH_CHL_STATUS_TRANSITION = 1
	SQUASH_CHL_STATUS_RUNNING    = 2
)

var ChannelStatus StatusSet
var chlAttrsInit = false
var channelsSeen map[string]bool

/*
Unlike the statistics produced via a topic, there is no discovery
of the attributes available in object STATUS queries. There is also
no discovery of descriptions for them. So this function hardcodes the
attributes we are going to look for and gives the associated descriptive
text. The elements can be expanded later; just trying to give a starting point
for now.
*/
func ChannelInitAttributes() {
	if chlAttrsInit {
		return
	}
	ChannelStatus.Attributes = make(map[string]*StatusAttribute)

	// These fields are used to construct the key to the per-channel map values and
	// as tags to uniquely identify a channel instance
	attr := ATTR_CHL_NAME
	ChannelStatus.Attributes[attr] = newPseudoStatusAttribute(attr, "Channel Name")
	attr = ATTR_CHL_RQMNAME
	ChannelStatus.Attributes[attr] = newPseudoStatusAttribute(attr, "Remote Queue Manager Name")
	attr = ATTR_CHL_JOBNAME
	ChannelStatus.Attributes[attr] = newPseudoStatusAttribute(attr, "MCA Job Name")
	attr = ATTR_CHL_CONNNAME
	ChannelStatus.Attributes[attr] = newPseudoStatusAttribute(attr, "Connection Name")

	// These are the integer status fields that are of interest
	attr = ATTR_CHL_MESSAGES
	ChannelStatus.Attributes[attr] = newStatusAttribute(attr, "Messages (API Calls for SVRCONN)", ibmmq.MQIACH_MSGS)
	ChannelStatus.Attributes[attr].delta = true // We have to manage the differences as MQ reports cumulative values

	// This is decoded by MQCHS_* values
	attr = ATTR_CHL_STATUS
	ChannelStatus.Attributes[attr] = newStatusAttribute(attr, "Channel Status", ibmmq.MQIACH_CHANNEL_STATUS)
	// The next value can be decoded from the MQCHSSTATE_* values
	attr = ATTR_CHL_SUBSTATE
	ChannelStatus.Attributes[attr] = newStatusAttribute(attr, "Channel Substate", ibmmq.MQIACH_CHANNEL_SUBSTATE)
	attr = ATTR_CHL_TYPE
	ChannelStatus.Attributes[attr] = newStatusAttribute(attr, "Channel Type", ibmmq.MQIACH_CHANNEL_TYPE)
	attr = ATTR_CHL_INSTANCE_TYPE
	ChannelStatus.Attributes[attr] = newStatusAttribute(attr, "Channel Instance Type", ibmmq.MQIACH_CHANNEL_INSTANCE_TYPE)

	// This is the same attribute as earlier, except that we indicate the values are to be modified in
	// a special way.
	attr = ATTR_CHL_STATUS_SQUASH
	ChannelStatus.Attributes[attr] = newStatusAttribute(attr, "Channel Status - Simplified", ibmmq.MQIACH_CHANNEL_STATUS)
	ChannelStatus.Attributes[attr].squash = true
	chlAttrsInit = true

	attr = ATTR_CHL_NETTIME_SHORT
	ChannelStatus.Attributes[attr] = newStatusAttribute(attr, "Network Time Short", ibmmq.MQIACH_NETWORK_TIME_INDICATOR)
	ChannelStatus.Attributes[attr].index = 0
	attr = ATTR_CHL_NETTIME_LONG
	ChannelStatus.Attributes[attr] = newStatusAttribute(attr, "Network Time Long", ibmmq.MQIACH_NETWORK_TIME_INDICATOR)
	ChannelStatus.Attributes[attr].index = 1

	attr = ATTR_CHL_SINCE_MSG
	ChannelStatus.Attributes[attr] = newStatusAttribute(attr, "Time Since Msg", -1)
}

// If we need to list the channels that match a pattern. Not needed for
// the status queries as they (unlike the pub/sub resource stats) accept
// patterns in the PCF command
func InquireChannels(patterns string) ([]string, error) {
	ChannelInitAttributes()
	return inquireObjects(patterns, ibmmq.MQOT_CHANNEL)
}

func CollectChannelStatus(patterns string) error {
	var err error
	channelsSeen = make(map[string]bool) // Record which channels have been seen in this period
	ChannelInitAttributes()

	// Empty any collected values
	for k := range ChannelStatus.Attributes {
		ChannelStatus.Attributes[k].Values = make(map[string]*StatusValue)
	}

	channelPatterns := strings.Split(patterns, ",")
	if len(channelPatterns) == 0 {
		return nil
	}

	for _, pattern := range channelPatterns {
		pattern = strings.TrimSpace(pattern)
		if len(pattern) == 0 {
			continue
		}

		// This would allow us to extract SAVED information too
		errCurrent := collectChannelStatus(pattern, ibmmq.MQOT_CURRENT_CHANNEL)
		err = errCurrent
		//errSaved := collectChannelStatus(pattern, ibmmq.MQOT_SAVED_CHANNEL)
		//if errCurrent != nil {
		//	err = errCurrent
		//} else {
		//	err = errSaved
		//}

	}

	// Need to clean out the prevValues elements to stop short-lived channels
	// building up in the map
	for a, _ := range ChannelStatus.Attributes {
		if ChannelStatus.Attributes[a].delta {
			m := ChannelStatus.Attributes[a].prevValues
			for key, _ := range m {
				if _, ok := channelsSeen[key]; ok {
					// Leave it in the map
				} else {
					// need to delete it from the map
					delete(m, key)
				}
			}
		}
	}
	return err

}

// Issue the INQUIRE_CHANNEL_STATUS command for a channel or wildcarded channel name
// Collect the responses and build up the statistics
func collectChannelStatus(pattern string, instanceType int32) error {
	var err error
	statusClearReplyQ()

	putmqmd, pmo, cfh, buf := statusSetCommandHeaders()

	// Can allow all the other fields to default
	cfh.Command = ibmmq.MQCMD_INQUIRE_CHANNEL_STATUS

	// Add the parameters one at a time into a buffer
	pcfparm := new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_STRING
	pcfparm.Parameter = ibmmq.MQCACH_CHANNEL_NAME
	pcfparm.String = []string{pattern}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	// Add the parameters one at a time into a buffer
	pcfparm = new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_INTEGER
	pcfparm.Parameter = ibmmq.MQIACH_CHANNEL_INSTANCE_TYPE
	pcfparm.Int64Value = []int64{int64(instanceType)}
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
	// per channel) or we run out of time
	for allReceived := false; !allReceived; {
		cfh, buf, allReceived, err = statusGetReply()
		if buf != nil {
			key := parseChlData(instanceType, cfh, buf)
			if key != "" {
				channelsSeen[key] = true
			}
		}
	}

	return err
}

// Given a PCF response message, parse it to extract the desired statistics
func parseChlData(instanceType int32, cfh *ibmmq.MQCFH, buf []byte) string {
	var elem *ibmmq.PCFParameter

	chlName := ""
	connName := ""
	jobName := ""
	rqmName := ""
	startDate := ""
	startTime := ""
	key := ""

	lastMsgDate := ""
	lastMsgTime := ""

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
		case ibmmq.MQCACH_CHANNEL_NAME:
			chlName = strings.TrimSpace(elem.String[0])
		case ibmmq.MQCACH_CONNECTION_NAME:
			connName = strings.TrimSpace(elem.String[0])
		case ibmmq.MQCACH_MCA_JOB_NAME:
			jobName = strings.TrimSpace(elem.String[0])
		case ibmmq.MQCA_REMOTE_Q_MGR_NAME:
			rqmName = strings.TrimSpace(elem.String[0])
		case ibmmq.MQCACH_CHANNEL_START_TIME:
			startTime = strings.TrimSpace(elem.String[0])
		case ibmmq.MQCACH_CHANNEL_START_DATE:
			startDate = strings.TrimSpace(elem.String[0])
		}
	}

	// Create a unique key for this channel instance
	//
	// The jobName does not exist on z/OS so it cannot be used to distinguish
	// unique instances of the same channel name. Instead, we try to fake it with
	// the channel start timestamp. That may still be wrong if lots of channel
	// instances start at the same time, but it's a lot better than combining the
	// instances badly.
	if jobName == "" && platform == ibmmq.MQPL_ZOS {
		jobName = startDate + ":" + startTime
	}
	key = chlName + "/" + connName + "/" + rqmName + "/" + jobName

	// Look to see if we've already seen a Current channel status that matches
	// the Saved version. If so, then don't bother with the Saved status
	if instanceType == ibmmq.MQOT_SAVED_CHANNEL {
		subKey := chlName + "/" + connName + "/" + rqmName + "/.*"
		for k, _ := range channelsSeen {
			re := regexp.MustCompile(subKey)
			if re.MatchString(k) {
				return ""
			}
		}
	}

	ChannelStatus.Attributes[ATTR_CHL_NAME].Values[key] = newStatusValueString(chlName)
	ChannelStatus.Attributes[ATTR_CHL_CONNNAME].Values[key] = newStatusValueString(connName)
	ChannelStatus.Attributes[ATTR_CHL_RQMNAME].Values[key] = newStatusValueString(rqmName)
	ChannelStatus.Attributes[ATTR_CHL_JOBNAME].Values[key] = newStatusValueString(jobName)

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

		if !statusGetIntAttributes(ChannelStatus, elem, key) {
			switch elem.Parameter {
			case ibmmq.MQCACH_LAST_MSG_TIME:
				lastMsgTime = strings.TrimSpace(elem.String[0])
			case ibmmq.MQCACH_LAST_MSG_DATE:
				lastMsgDate = strings.TrimSpace(elem.String[0])
			}
		}
	}

	now := time.Now()
	diff := statusTimeDiff(now, lastMsgDate, lastMsgTime)
	ChannelStatus.Attributes[ATTR_CHL_SINCE_MSG].Values[key] = newStatusValueInt64(diff)

	return key
}

// Return a standardised value. If the attribute indicates that something
// special has to be done, then do that. Otherwise just make sure it's a non-negative
// value of the correct datatype
func ChannelNormalise(attr *StatusAttribute, v int64) float64 {
	var f float64

	if attr.squash {
		switch attr.pcfAttr {

		case ibmmq.MQIACH_CHANNEL_STATUS:
			v32 := int32(v)
			switch v32 {
			case ibmmq.MQCHS_INACTIVE,
				ibmmq.MQCHS_DISCONNECTED,
				ibmmq.MQCHS_STOPPED,
				ibmmq.MQCHS_PAUSED:
				f = float64(SQUASH_CHL_STATUS_STOPPED)

			case ibmmq.MQCHS_BINDING,
				ibmmq.MQCHS_STARTING,
				ibmmq.MQCHS_STOPPING,
				ibmmq.MQCHS_RETRYING,
				ibmmq.MQCHS_REQUESTING,
				ibmmq.MQCHS_INITIALIZING,
				ibmmq.MQCHS_SWITCHING:
				f = float64(SQUASH_CHL_STATUS_TRANSITION)
			case ibmmq.MQCHS_RUNNING:
				f = float64(SQUASH_CHL_STATUS_RUNNING)
			default:
				f = float64(SQUASH_CHL_STATUS_STOPPED)
			}
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
