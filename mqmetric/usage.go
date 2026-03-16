package mqmetric

/*
  Copyright (c) IBM Corporation 2016, 2021

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
Functions in this file use the DISPLAY USAGE    command to extract metrics
about MQ on z/OS pageset and bufferpool use.
*/

import (
	//	"fmt"
	"strconv"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

const (
	ATTR_BP_ID           = "id"
	ATTR_BP_LOCATION     = "location"
	ATTR_BP_CLASS        = "pageclass"
	ATTR_BP_FREE         = "buffers_free"
	ATTR_BP_FREE_PERCENT = "buffers_free_percent"
	ATTR_BP_TOTAL        = "buffers_total"

	ATTR_PS_ID           = "id"
	ATTR_PS_BPID         = "bufferpool"
	ATTR_PS_TOTAL        = "pages_total"
	ATTR_PS_UNUSED       = "pages_unused"
	ATTR_PS_NP_PAGES     = "pages_nonpersistent"
	ATTR_PS_P_PAGES      = "pages_persistent"
	ATTR_PS_STATUS       = "status"
	ATTR_PS_EXPAND_COUNT = "expansion_count"
)

func UsageInitAttributes() {
	traceEntry("usageInitAttributes")

	otbp := OT_BP
	otps := OT_PS

	ci := getConnection(GetConnectionKey())
	osbp := &ci.objectStatus[otbp]
	osps := &ci.objectStatus[otps]
	stbp := GetObjectStatus(GetConnectionKey(), otbp)
	stps := GetObjectStatus(GetConnectionKey(), otps)
	if osbp.init && osps.init {
		traceExit("usageInitAttributes", 1)
		return
	}
	stbp.Attributes = make(map[string]*StatusAttribute)
	stps.Attributes = make(map[string]*StatusAttribute)

	// The Bufferpool-related fields that are of interest
	newPseudoStatusMapEntryRequired(stbp, otbp, ATTR_BP_ID, "Buffer Pool ID")
	newPseudoStatusMapEntryRequired(stbp, otbp, ATTR_BP_LOCATION, "Buffer Pool Location")
	newPseudoStatusMapEntryRequired(stbp, otbp, ATTR_BP_CLASS, "Buffer Pool Class")

	newStatusMapEntry(stbp, otbp, ATTR_BP_FREE, "Free buffers", ibmmq.MQIACF_USAGE_FREE_BUFF, false)
	newStatusMapEntry(stbp, otbp, ATTR_BP_FREE_PERCENT, "Free buffers percent", ibmmq.MQIACF_USAGE_FREE_BUFF_PERC, false)
	newStatusMapEntry(stbp, otbp, ATTR_BP_TOTAL, "Total buffers", ibmmq.MQIACF_USAGE_TOTAL_BUFFERS, false)

	// And the Pageset-related metrics
	newPseudoStatusMapEntryRequired(stps, otps, ATTR_PS_ID, "Pageset ID")
	newPseudoStatusMapEntryRequired(stps, otps, ATTR_PS_BPID, "Buffer Pool ID")
	newStatusMapEntry(stps, otps, ATTR_PS_TOTAL, "Total pages", ibmmq.MQIACF_USAGE_TOTAL_PAGES, false)
	newStatusMapEntry(stps, otps, ATTR_PS_UNUSED, "Unused pages", ibmmq.MQIACF_USAGE_UNUSED_PAGES, false)
	newStatusMapEntry(stps, otps, ATTR_PS_NP_PAGES, "Non-persistent pages", ibmmq.MQIACF_USAGE_NONPERSIST_PAGES, false)
	newStatusMapEntry(stps, otps, ATTR_PS_P_PAGES, "Persistent pages", ibmmq.MQIACF_USAGE_PERSIST_PAGES, false)
	newStatusMapEntry(stps, otps, ATTR_PS_STATUS, "Status", ibmmq.MQIACF_PAGESET_STATUS, false)
	newStatusMapEntry(stps, otps, ATTR_PS_EXPAND_COUNT, "Expansion Count", ibmmq.MQIACF_USAGE_EXPAND_COUNT, false)

	osbp.init = true
	osps.init = true

	traceExit("usageInitAttributes", 0)

}

func CollectUsageStatus() error {
	var err error
	traceEntry("CollectUsageStatus")

	stbp := GetObjectStatus(GetConnectionKey(), OT_BP)
	stps := GetObjectStatus(GetConnectionKey(), OT_PS)

	UsageInitAttributes()

	// Empty any collected values
	for k := range stbp.Attributes {
		stbp.Attributes[k].Values = make(map[string]*StatusValue)
	}
	for k := range stps.Attributes {
		stps.Attributes[k].Values = make(map[string]*StatusValue)
	}
	err = collectUsageStatus()
	traceExitErr("CollectUsageStatus", 0, err)
	return err
}

func collectUsageStatus() error {
	var err error
	traceEntry("collectUsageStatus")
	ci := getConnection(GetConnectionKey())

	statusClearReplyQ()

	putmqmd, pmo, cfh, buf := statusSetCommandHeaders()
	// Can allow all the other fields to default
	cfh.Command = ibmmq.MQCMD_INQUIRE_USAGE

	// There are no additional parameters required as the
	// default behaviour of the command returns what we need

	// Once we know the total number of parameters, put the
	// CFH header on the front of the buffer.
	buf = append(cfh.Bytes(), buf...)

	// And now put the command to the queue
	err = ci.si.cmdQObj.Put(putmqmd, pmo, buf)
	if err != nil {
		traceExitErr("collectUsageStatus", 1, err)
		return err

	}

	for allReceived := false; !allReceived; {
		cfh, buf, allReceived, err = statusGetReply(putmqmd.MsgId)
		if buf != nil {
			//	fmt.Printf("UsageBP Data received. cfh %v err %v\n",cfh,err)
			parseUsageData(cfh, buf)
		}

	}

	traceExitErr("collectUsageStatus", 0, err)

	return err
}

// Given a PCF response message, parse it to extract the desired statistics
func parseUsageData(cfh *ibmmq.MQCFH, buf []byte) string {
	var elem *ibmmq.PCFParameter
	var responseType int32

	traceEntry("parseUsageData")

	stbp := GetObjectStatus(GetConnectionKey(), OT_BP)
	stps := GetObjectStatus(GetConnectionKey(), OT_PS)

	bpId := ""
	bpLocation := ""
	bpClass := ""
	psId := ""

	key := ""
	parmAvail := true
	bytesRead := 0
	offset := 0
	datalen := len(buf)
	if cfh == nil || cfh.ParameterCount == 0 {
		traceExit("parseUsageData", 1)
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
		case ibmmq.MQIACF_USAGE_TYPE:
			v := int32(elem.Int64Value[0])
			switch v {
			case ibmmq.MQIACF_USAGE_BUFFER_POOL, ibmmq.MQIACF_USAGE_PAGESET:
				responseType = v
			default:
				traceExit("parseUsageData", 2)
				return ""
			}

		case ibmmq.MQIACF_BUFFER_POOL_ID:
			bpId = strconv.FormatInt(elem.Int64Value[0], 10)
		case ibmmq.MQIA_PAGESET_ID:
			psId = strconv.FormatInt(elem.Int64Value[0], 10)
		case ibmmq.MQIACF_BUFFER_POOL_LOCATION:
			v := elem.Int64Value[0]
			switch int32(v) {
			case ibmmq.MQBPLOCATION_ABOVE:
				bpLocation = "Above"
			case ibmmq.MQBPLOCATION_BELOW:
				bpLocation = "Below"
			case ibmmq.MQBPLOCATION_SWITCHING_ABOVE:
				bpLocation = "Switching Above"
			case ibmmq.MQBPLOCATION_SWITCHING_BELOW:
				bpLocation = "Switching Below"
			}

		case ibmmq.MQIACF_PAGECLAS:
			v := elem.Int64Value[0]
			switch int32(v) {
			case ibmmq.MQPAGECLAS_4KB:
				bpClass = "4KB"
			case ibmmq.MQPAGECLAS_FIXED4KB:
				bpClass = "Fixed4KB"
			}
		}
	}

	// The DISPLAY USAGE command (with no qualifiers) returns two types of response.
	// Buffer pool usage and pageset usage are both reported. We can use the responseType
	// to work with both in a single pass and update separate blocks of data.
	if responseType == ibmmq.MQIACF_USAGE_BUFFER_POOL {

		// Create a unique key for this instance
		key = bpId

		v, ok := stbp.Attributes[ATTR_BP_ID]
		if ok {
			v.Values[key] = newStatusValueString(bpId)
		}
		v, ok = stbp.Attributes[ATTR_BP_LOCATION]
		if ok {
			v.Values[key] = newStatusValueString(bpLocation)
		}
		v, ok = stbp.Attributes[ATTR_BP_CLASS]
		if ok {
			v.Values[key] = newStatusValueString(bpClass)
		}

		parmAvail = true
		// And then re-parse the message so we can store the metrics now knowing the map key
		offset = 0
		for parmAvail && cfh.CompCode != ibmmq.MQCC_FAILED {
			elem, bytesRead = ibmmq.ReadPCFParameter(buf[offset:])
			offset += bytesRead
			// Have we now reached the end of the message
			if offset >= datalen {
				parmAvail = false
			}

			statusGetIntAttributes(GetObjectStatus(GetConnectionKey(), OT_BP), elem, key)
		}
	} else {
		// Create a unique key for this instance
		key = psId

		v, ok := stps.Attributes[ATTR_PS_ID]
		if ok {
			v.Values[key] = newStatusValueString(psId)
		}
		v, ok = stps.Attributes[ATTR_PS_BPID]
		if ok {
			v.Values[key] = newStatusValueString(bpId)
		}

		parmAvail = true
		// And then re-parse the message so we can store the metrics now knowing the map key
		offset = 0
		for parmAvail && cfh.CompCode != ibmmq.MQCC_FAILED {
			elem, bytesRead = ibmmq.ReadPCFParameter(buf[offset:])
			offset += bytesRead
			// Have we now reached the end of the message
			if offset >= datalen {
				parmAvail = false
			}

			statusGetIntAttributes(GetObjectStatus(GetConnectionKey(), OT_PS), elem, key)
		}
	}
	traceExitF("parseUsageData", 0, "Key: %s", key)
	return key
}

// Return a standardised value. If the attribute indicates that something
// special has to be done, then do that. Otherwise just make sure it's a non-negative
// value of the correct datatype
func UsageNormalise(attr *StatusAttribute, v int64) float64 {
	return statusNormalise(attr, v)
}
