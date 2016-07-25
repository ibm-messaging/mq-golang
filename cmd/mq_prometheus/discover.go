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

/*
Functions in this file discover the data available from a queue manager
via the MQ V9 pub/sub monitoring feature. Each metric (element) is
found by discovering the types of metric, and the types are found by first
discovering the classes. Sample program amqsrua is shipped with MQ V9 to
give a good demonstration of the process, which is followed here.
*/

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/prometheus/common/log"
	"ibmmq"
	"strings"
)

type monElement struct {
	parent      *monType
	description string // An English phrase describing the element
	metricName  string // Reformatted description suitable as label
	datatype    int32
	gaugeVec    *prometheus.GaugeVec
	values      map[string]int64
}

type monType struct {
	parent       *monClass
	name         string
	description  string
	objectTopic  string // topic for actual data responses
	elementTopic string // discovery of elements
	elements     map[int]*monElement
	subHobj      map[string]ibmmq.MQObject
}

type monClass struct {
	parent      *allMetrics
	name        string
	description string
	typesTopic  string
	flags       int
	types       map[int]*monType
}

type allMetrics struct {
	classes map[int]*monClass
}

const (
	namespace  = "ibmmq" // All our metrics will show up under this
	qMgrMapKey = "@self" // can never be a real object name
)

var (
	metrics allMetrics
	qList   []string
)

func discoverClasses() error {
	var data []byte
	var sub ibmmq.MQObject
	var err error

	// Have to know the starting point for the topic that tells about classes
	sub, err = subscribe("$SYS/MQ/INFO/QMGR/" + qMgr.Name + "/Monitor/METADATA/CLASSES")
	if err == nil {
		data, err = getMessage(true)
		sub.Close(0)

		elemList, _ := parsePCFResponse(data)

		for i := 0; i < len(elemList); i++ {
			if elemList[i].Type != ibmmq.MQCFT_GROUP {
				continue
			}
			group := elemList[i]
			thisClass := new(monClass)
			classIndex := 0
			thisClass.types = make(map[int]*monType)
			thisClass.parent = &metrics

			for j := 0; j < len(group.GroupList); j++ {
				elem := group.GroupList[j]
				switch elem.Parameter {
				case ibmmq.MQIAMO_MONITOR_CLASS:
					classIndex = int(elem.Int64Value[0])
				case ibmmq.MQIAMO_MONITOR_FLAGS:
					thisClass.flags = int(elem.Int64Value[0])
				case ibmmq.MQCAMO_MONITOR_CLASS:
					thisClass.name = elem.String[0]
				case ibmmq.MQCAMO_MONITOR_DESC:
					thisClass.description = elem.String[0]
				case ibmmq.MQCA_TOPIC_STRING:
					thisClass.typesTopic = elem.String[0]
				default:
					log.Errorf("Unknown parameter %d in class discovery", elem.Parameter)
				}
			}
			metrics.classes[classIndex] = thisClass
		}
	}

	subsOpened = true
	return err
}

func discoverTypes(thisClass *monClass) error {
	var data []byte
	var sub ibmmq.MQObject
	var err error

	//log.Infof("Working on class %s", thisClass.name)
	sub, err = subscribe(thisClass.typesTopic)
	if err == nil {
		data, err = getMessage(true)
		sub.Close(0)

		elemList, _ := parsePCFResponse(data)

		for i := 0; i < len(elemList); i++ {
			if elemList[i].Type != ibmmq.MQCFT_GROUP {
				continue
			}

			group := elemList[i]
			thisType := new(monType)
			thisType.elements = make(map[int]*monElement)
			thisType.subHobj = make(map[string]ibmmq.MQObject)

			typeIndex := 0
			thisType.parent = thisClass

			for j := 0; j < len(group.GroupList); j++ {
				elem := group.GroupList[j]
				switch elem.Parameter {

				case ibmmq.MQIAMO_MONITOR_TYPE:
					typeIndex = int(elem.Int64Value[0])
				case ibmmq.MQCAMO_MONITOR_TYPE:
					thisType.name = elem.String[0]
				case ibmmq.MQCAMO_MONITOR_DESC:
					thisType.description = elem.String[0]
				case ibmmq.MQCA_TOPIC_STRING:
					thisType.elementTopic = elem.String[0]
				default:
					log.Errorf("Unknown parameter %d in type discovery", elem.Parameter)
				}
			}
			thisClass.types[typeIndex] = thisType
		}
	}
	return err
}

func discoverElements(thisType *monType) error {
	var err error
	var data []byte
	var sub ibmmq.MQObject
	var thisElement *monElement

	sub, err = subscribe(thisType.elementTopic)
	if err == nil {
		data, err = getMessage(true)
		sub.Close(0)

		elemList, _ := parsePCFResponse(data)

		for i := 0; i < len(elemList); i++ {

			if elemList[i].Type == ibmmq.MQCFT_STRING && elemList[i].Parameter == ibmmq.MQCA_TOPIC_STRING {
				thisType.objectTopic = elemList[i].String[0]
				continue
			}

			if elemList[i].Type != ibmmq.MQCFT_GROUP {
				continue
			}

			group := elemList[i]

			thisElement = new(monElement)
			elementIndex := 0
			thisElement.parent = thisType
			thisElement.values = make(map[string]int64)

			for j := 0; j < len(group.GroupList); j++ {
				elem := group.GroupList[j]
				switch elem.Parameter {

				case ibmmq.MQIAMO_MONITOR_ELEMENT:
					elementIndex = int(elem.Int64Value[0])
				case ibmmq.MQIAMO_MONITOR_DATATYPE:
					thisElement.datatype = int32(elem.Int64Value[0])
				case ibmmq.MQCAMO_MONITOR_DESC:
					thisElement.description = elem.String[0]
				default:
					log.Errorf("Unknown parameter %d in type discovery", elem.Parameter)
				}
			}

			thisElement.metricName = formatDescription(thisElement)
			thisType.elements[elementIndex] = thisElement
		}
	}

	return err
}

/*
Discover the complete set of available statistics in the queue manager
by working through the classes, types and individual elements.

Then discover the list of individual queues we have been asked for.
*/
func discoverStats() error {
	var err error

	// Start with an empty set of information about the available stats
	metrics.classes = make(map[int]*monClass)

	// Then get the list of CLASSES
	err = discoverClasses()

	// For each CLASS, discover the TYPEs of data available
	if err == nil {
		for _, thisClass := range metrics.classes {
			err = discoverTypes(thisClass)
			// And for each CLASS, discover the actual statistics elements
			if err == nil {
				for _, thisType := range thisClass.types {
					err = discoverElements(thisType)
				}
			}
		}
		//

	}

	for _, thisClass := range metrics.classes {
		for _, thisType := range thisClass.types {
			for _, thisElement := range thisType.elements {
				log.Debugf("DUMP Element: Desc = %s ParentType = %s MetaTopic = %s Real Topic = %s Type = %d",
					thisElement.metricName,
					thisElement.parent.name,
					thisElement.parent.elementTopic,
					thisType.objectTopic, thisElement.datatype)
			}
		}
	}

	return err
}

/*
discoverQueues lists the queues that match all of the configured
patterns.

The patterns must match the MQ rule - asterisk on the end of the
string only.

If a bad pattern is used, or no queues exist that match the pattern
then an error is reported but we continue processing other patterns.

An alternative would be to list ALL the queues (though that could be a
long list, and we would really have to worry about TRUNCATED message retrieval),
and then use a more general regexp match. Something for a later update
perhaps.
*/
func discoverQueues() error {
	var err error
	var elem *ibmmq.PCFParameter
	var datalen int

	queues := strings.Split(config.monitoredQueues, ",")
	for i := 0; i < len(queues) && err == nil; i++ {
		pattern := queues[i]

		if strings.Count(pattern, "*") > 1 ||
			(strings.Count(pattern, "*") == 1 && !strings.HasSuffix(pattern, "*")) {
			log.Errorf("Queue pattern '%s' is not valid", pattern)
			continue
		}

		putmqmd := ibmmq.NewMQMD()
		pmo := ibmmq.NewMQPMO()

		pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
		pmo.Options |= ibmmq.MQPMO_NEW_MSG_ID
		pmo.Options |= ibmmq.MQPMO_NEW_CORREL_ID
		pmo.Options |= ibmmq.MQPMO_FAIL_IF_QUIESCING

		putmqmd.Format = "MQADMIN"
		putmqmd.ReplyToQ = replyQObj.Name
		putmqmd.MsgType = ibmmq.MQMT_REQUEST
		putmqmd.Report = ibmmq.MQRO_PASS_DISCARD_AND_EXPIRY

		cfh := ibmmq.NewMQCFH()

		// Can allow all the other fields to default
		cfh.Command = ibmmq.MQCMD_INQUIRE_Q_NAMES

		// Add the parameters one at a time into a buffer
		buf := make([]byte, 0)
		pcfparm := new(ibmmq.PCFParameter)
		pcfparm.Type = ibmmq.MQCFT_STRING
		pcfparm.Parameter = ibmmq.MQCA_Q_NAME
		pcfparm.String = []string{pattern}
		cfh.ParameterCount++
		buf = append(buf, pcfparm.Bytes()...)

		pcfparm = new(ibmmq.PCFParameter)
		pcfparm.Type = ibmmq.MQCFT_INTEGER
		pcfparm.Parameter = ibmmq.MQIA_Q_TYPE
		pcfparm.Int64Value = []int64{int64(ibmmq.MQQT_LOCAL)}
		cfh.ParameterCount++
		buf = append(buf, pcfparm.Bytes()...)

		// Once we know the total number of parameters, put the
		// CFH header on the front of the buffer.
		buf = append(cfh.Bytes(), buf...)

		// And put the command to the queue
		_, err = cmdQObj.Put(putmqmd, pmo, buf)

		if err != nil {
			log.Error(err)
		}

		// Now get the response
		getmqmd := ibmmq.NewMQMD()
		gmo := ibmmq.NewMQGMO()
		gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
		gmo.Options |= ibmmq.MQGMO_FAIL_IF_QUIESCING
		gmo.Options |= ibmmq.MQGMO_WAIT
		gmo.Options |= ibmmq.MQGMO_CONVERT
		gmo.WaitInterval = 30 * 1000

		// Ought to add a loop here in case we get truncated data
		buf = make([]byte, 32768)

		datalen, _, err = replyQObj.Get(getmqmd, gmo, buf)
		if err == nil {
			cfh, offset := ibmmq.ReadPCFHeader(buf)
			if cfh.CompCode != ibmmq.MQCC_OK {
				log.Errorf("PCF command failed with CC %d RC %d",
					cfh.CompCode,
					cfh.Reason)
			} else {
				parmAvail := true
				bytesRead := 0
				for parmAvail && cfh.CompCode != ibmmq.MQCC_FAILED {
					elem, bytesRead = ibmmq.ReadPCFParameter(buf[offset:])
					offset += bytesRead
					// Have we now reached the end of the message
					if offset >= datalen {
						parmAvail = false
					}

					switch elem.Parameter {
					case ibmmq.MQCACF_Q_NAMES:
						if len(elem.String) == 0 {
							log.Errorf("No queues matching '%s' exist", pattern)
						}
						for i := 0; i < len(elem.String); i++ {
							qList = append(qList, strings.TrimSpace(elem.String[i]))
						}
					}
				}
			}
		} else {
			log.Error(err)
		}
	}

	log.Infof("Discovered queues = %v", qList)

	return err
}

/*
Now that we know which topics can return data, need to
create all the subscriptions. We don't keep track of
all the subscription handles, but can assume they will be
removed on exit. Will need to revisit this if/when a dynamic "refresh
config" is implemented.

As subscriptions are created, also create the corresponding
Gauge for prometheus monitoring
*/
func createSubscriptions() error {
	var err error
	var sub ibmmq.MQObject

loop:
	for _, thisClass := range metrics.classes {
		for _, thisType := range thisClass.types {

			if strings.Contains(thisType.objectTopic, "%s") {
				for i := 0; i < len(qList); i++ {
					topic := fmt.Sprintf(thisType.objectTopic, qList[i])
					sub, err = subscribe(topic)
					thisType.subHobj[qList[i]] = sub
				}
			} else {
				sub, err = subscribe(thisType.objectTopic)
				thisType.subHobj[qMgrMapKey] = sub
			}

			for _, thisElement := range thisType.elements {
				thisElement.gaugeVec = newMqGaugeVec(thisElement)
			}
			if err != nil {
				log.Error("Error subscribing: ", err)
				break loop
			}
		}
	}

	return err
}

/*
newMqGaugeVec returns the structure which will contain the
value and suitable labels. For queues we tag each entry
with both the queue and qmgr name; for the qmgr-wide entries, we
only need the single label.
*/
func newMqGaugeVec(thisElement *monElement) *prometheus.GaugeVec {
	queueLabelNames := []string{"object", "qmgr"}
	qmgrLabelNames := []string{"qmgr"}

	labels := qmgrLabelNames
	prefix := "qmgr_"

	if strings.Contains(thisElement.parent.objectTopic, "%s") {
		labels = queueLabelNames
		prefix = "object_"
	}

	gaugeVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      prefix + thisElement.metricName,
			Help:      thisElement.description,
		},
		labels,
	)

	// log.Infof("Created gauge for %s", thisElement.metricName)
	return gaugeVec
}

/*
Parse a PCF response message, printing the
elements.

Returns TRUE if this is the last response in a
set, based on the MQCFH.Control value.
*/
func parsePCFResponse(buf []byte) ([]*ibmmq.PCFParameter, bool) {
	var elem *ibmmq.PCFParameter
	var elemList []*ibmmq.PCFParameter
	var bytesRead int

	rc := false

	// First get the MQCFH structure. This also returns
	// the number of bytes read so we know where to start
	// looking for the next element
	cfh, offset := ibmmq.ReadPCFHeader(buf)

	// If the command succeeded, loop through the remainder of the
	// message to decode each parameter.
	for i := 0; i < int(cfh.ParameterCount); i++ {
		// We don't know how long the parameter is, so we just
		// pass in "from here to the end" and let the parser
		// tell us how far it got.
		elem, bytesRead = ibmmq.ReadPCFParameter(buf[offset:])
		offset += bytesRead
		// Have we now reached the end of the message
		elemList = append(elemList, elem)
		if elem.Type == ibmmq.MQCFT_GROUP {
			groupElem := elem
			for j := 0; j < int(groupElem.ParameterCount); j++ {
				elem, bytesRead = ibmmq.ReadPCFParameter(buf[offset:])
				offset += bytesRead
				groupElem.GroupList = append(groupElem.GroupList, elem)
			}
		}

	}

	if cfh.Control == ibmmq.MQCFC_LAST {
		rc = true
	}
	return elemList, rc
}

/*
Need to turn the "friendly" name of each element into something
that is suitable for Prometheus metric names. The rules for that are
very limited (basically a-z, A-Z, 0-9, _).

The guidelines also discuss unit consistency (always use seconds,
bytes etc), and organisation of the elements of the name (units last)

While we can't change the MQ-generated descriptions for its statistics,
we can reformat most of them heuristically here.
*/
func formatDescription(elem *monElement) string {
	s := elem.description

	s = strings.Replace(s, " ", "_", -1)
	s = strings.Replace(s, "/", "_", -1)
	s = strings.Replace(s, "-", "_", -1)

	/* common pattern is "xxx - yyy" leading to 3 ugly adjacent underscores */
	s = strings.Replace(s, "___", "_", -1)
	s = strings.Replace(s, "__", "_", -1)

	/* make it all lowercase. Not essential, but looks better */
	s = strings.ToLower(s)

	// Do not use _count
	s = strings.Replace(s, "_count", "", -1)

	// Switch round a couple of specific names
	s = strings.Replace(s, "bytes_written", "written_bytes", -1)
	s = strings.Replace(s, "bytes_max", "max_bytes", -1)
	s = strings.Replace(s, "bytes_in_use", "in_use_bytes", -1)
	s = strings.Replace(s, "messages_expired", "expired_messages", -1)

	if strings.HasSuffix(s, "free_space") {
		s = s + "_bytes"
	}

	// Make "byte", "file" and "message" units plural
	if strings.HasSuffix(s, "byte") ||
		strings.HasSuffix(s, "message") ||
		strings.HasSuffix(s, "file") {
		s = s + "s"
	}

	// Move % to the end
	if strings.Contains(s, "_percentage_") {
		s = strings.Replace(s, "_percentage_", "_", -1)
		s += "_percentage"
	}

	unit := ""
	switch elem.datatype {
	case ibmmq.MQIAMO_MONITOR_MICROSEC:
		// Although the qmgr captures in us, we convert when
		// pushing out to prometheus, so this label needs to match
		unit = "_seconds"
	}

	s += unit

	return s
}
