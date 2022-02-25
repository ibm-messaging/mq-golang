package mqmetric

/*
  Copyright (c) IBM Corporation 2016, 2020

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
This file holds most of the calls to the MQI, so we
don't need to repeat common setups eg of MQMD or MQSD structures.
*/

import (
	"fmt"
	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

var (
	getBuffer = make([]byte, 32768)
)

type ConnectionConfig struct {
	ClientMode    bool
	UserId        string
	Password      string
	TZOffsetSecs  float64
	SingleConnect bool

	UsePublications      bool
	UseStatus            bool
	UseResetQStats       bool
	ShowInactiveChannels bool

	CcdtUrl  string
	ConnName string
	Channel  string
}

// Which objects are available for subscription. How
// do we define which ones to subscribe to and filter the
// specific subscriptions.

type DiscoverObject struct {
	ObjectNames          string
	UseWildcard          bool
	SubscriptionSelector string
}

// For now, only queues are subscribable through this interface
// but there are now Application resources that might be relevant
// at some time.
type DiscoverConfig struct {
	MetaPrefix      string // Root of all meta-data discovery
	MonitoredQueues DiscoverObject
}

type MQMetricError struct {
	Err      string
	MQReturn *ibmmq.MQReturn
}

func (e MQMetricError) Error() string { return e.Err + " : " + e.MQReturn.Error() }
func (e MQMetricError) Unwrap() error { return e.MQReturn }

/*
InitConnection connects to the queue manager, and then
opens both the command queue and a dynamic reply queue
to be used for all responses including the publications
*/
func InitConnection(qMgrName string, replyQ string, replyQ2 string, cc *ConnectionConfig) error {
	return initConnectionKey("", qMgrName, replyQ, replyQ2, cc)
}
func InitConnectionKey(key string, qMgrName string, replyQ string, replyQ2 string, cc *ConnectionConfig) error {
	return initConnectionKey(key, qMgrName, replyQ, replyQ2, cc)
}
func initConnectionKey(key string, qMgrName string, replyQ string, replyQ2 string, cc *ConnectionConfig) error {
	var err error
	var gocd *ibmmq.MQCD
	var mqreturn *ibmmq.MQReturn
	var errorString = ""

	traceEntryF("InitConnection", "QMgrName %s", qMgrName)

	initConnection(key)

	gocno := ibmmq.NewMQCNO()
	gocsp := ibmmq.NewMQCSP()

	// Copy initialisation configuraton information to local structure
	ci := getConnection(GetConnectionKey())

	ci.tzOffsetSecs = cc.TZOffsetSecs
	ci.showInactiveChannels = cc.ShowInactiveChannels

	// Explicitly force client mode if requested. Otherwise use the "default"
	// Client mode can be come from a simple boolean, or from having
	// common configurations with the CCDT or ConnName/Channel being set.
	if cc.CcdtUrl != "" {
		cc.ClientMode = true
	} else if cc.ConnName != "" || cc.Channel != "" {
		cc.ClientMode = true
		gocd = ibmmq.NewMQCD()
		gocd.ChannelName = cc.Channel
		gocd.ConnectionName = cc.ConnName
	}

	// connection mechanism depending on what is installed or configured.
	if cc.ClientMode {
		gocno.Options = ibmmq.MQCNO_CLIENT_BINDING
		// Force reconnection to only be to the same qmgr. Cannot do this with externally
		// configured (eg MQ_CONNECT_TYPE or client-only installation) connections. But
		// it is a bad idea to try to reconnect to a different queue manager.
		// If the collector is managing its own reconnect, then don't use the MQ automatic mode
		if cc.SingleConnect {
			gocno.Options |= ibmmq.MQCNO_RECONNECT_DISABLED
		} else {
			gocno.Options |= ibmmq.MQCNO_RECONNECT_Q_MGR
		}
		if cc.CcdtUrl != "" {
			gocno.CCDTUrl = cc.CcdtUrl
			logInfo("Trying to connect as client using CCDT: %s", gocno.CCDTUrl)
		} else if gocd != nil {
			gocno.ClientConn = gocd
			logInfo("Trying to connect as client using ConnName: %s, Channel: %s", gocd.ConnectionName, gocd.ChannelName)
		} else {
			logInfo("Trying to connect as client with external configuration")
		}
	}
	gocno.Options |= ibmmq.MQCNO_HANDLE_SHARE_BLOCK

	if cc.Password != "" {
		gocsp.Password = cc.Password
	}
	if cc.UserId != "" {
		gocsp.UserId = cc.UserId
		gocno.SecurityParms = gocsp
	}

	logDebug("Connecting to queue manager %s", qMgrName)
	ci.si.qMgr, err = ibmmq.Connx(qMgrName, gocno)
	if err == nil {
		ci.si.qmgrConnected = true
	} else {
		errorString = "Cannot connect to queue manager " + qMgrName
		mqreturn = err.(*ibmmq.MQReturn)
	}

	// Discover important information about the qmgr - its real name
	// and the platform type. Also check if it is at least V9 (on Distributed platforms)
	// so that monitoring will work.
	if err == nil {
		var v map[int32]interface{}

		ci.useStatus = cc.UseStatus

		mqod := ibmmq.NewMQOD()
		openOptions := ibmmq.MQOO_INQUIRE + ibmmq.MQOO_FAIL_IF_QUIESCING

		mqod.ObjectType = ibmmq.MQOT_Q_MGR
		mqod.ObjectName = ""

		ci.si.qMgrObject, err = ci.si.qMgr.Open(mqod, openOptions)

		if err == nil {
			selectors := []int32{ibmmq.MQCA_Q_MGR_NAME,
				ibmmq.MQIA_COMMAND_LEVEL,
				ibmmq.MQIA_PERFORMANCE_EVENT,
				ibmmq.MQIA_MAX_HANDLES,
				ibmmq.MQIA_PLATFORM}

			v, err = ci.si.qMgrObject.InqMap(selectors)
			if err == nil {
				ci.si.resolvedQMgrName = v[ibmmq.MQCA_Q_MGR_NAME].(string)
				ci.si.platform = v[ibmmq.MQIA_PLATFORM].(int32)
				ci.si.commandLevel = v[ibmmq.MQIA_COMMAND_LEVEL].(int32)
				ci.si.maxHandles = v[ibmmq.MQIA_MAX_HANDLES].(int32)
				if ci.si.platform == ibmmq.MQPL_ZOS {
					ci.usePublications = false
					ci.useResetQStats = cc.UseResetQStats
					evEnabled := v[ibmmq.MQIA_PERFORMANCE_EVENT].(int32)
					if ci.useResetQStats && evEnabled == 0 {
						errorString = "Requested use of RESET QSTATS but queue manager has PERFMEV(DISABLED)"
						//err = fmt.Errorf(errorString) // Bypass the error for a while
					}
				} else {
					if cc.UsePublications == true {
						if ci.si.commandLevel < 900 && ci.si.platform != ibmmq.MQPL_APPLIANCE {
							//err = fmt.Errorf("Queue manager must be at least V9.0 for full monitoring. The ibmmq.usePublications configuration parameter can be used to permit limited monitoring.")
							errorString = "Unsupported system"
						} else {
							ci.usePublications = cc.UsePublications
						}
					} else {
						ci.usePublications = false
					}
				}
			}
		} else {
			errorString = "Cannot open queue manager object"
			mqreturn = err.(*ibmmq.MQReturn)
		}
	}

	// MQOPEN of the COMMAND QUEUE
	if err == nil {
		mqod := ibmmq.NewMQOD()

		openOptions := ibmmq.MQOO_OUTPUT | ibmmq.MQOO_FAIL_IF_QUIESCING

		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = "SYSTEM.ADMIN.COMMAND.QUEUE"
		if ci.si.platform == ibmmq.MQPL_ZOS {
			mqod.ObjectName = "SYSTEM.COMMAND.INPUT"
		}

		ci.si.cmdQObj, err = ci.si.qMgr.Open(mqod, openOptions)
		if err != nil {
			errorString = "Cannot open queue " + mqod.ObjectName
			mqreturn = err.(*ibmmq.MQReturn)
		}

	}

	// MQOPEN of a reply queue also used for subscription delivery
	if err == nil {
		mqod := ibmmq.NewMQOD()
		openOptions := ibmmq.MQOO_INPUT_EXCLUSIVE | ibmmq.MQOO_FAIL_IF_QUIESCING
		openOptions |= ibmmq.MQOO_INQUIRE
		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = replyQ
		ci.si.replyQObj, err = ci.si.qMgr.Open(mqod, openOptions)
		ci.si.replyQBaseName = replyQ
		if err == nil {
			ci.si.queuesOpened = true
			clearQ(ci.si.replyQObj)
		} else {
			errorString = "Cannot open queue " + mqod.ObjectName
			mqreturn = err.(*ibmmq.MQReturn)
		}
	}

	// MQOPEN of a second reply queue used for status polling
	if err == nil {
		mqod := ibmmq.NewMQOD()
		openOptions := ibmmq.MQOO_INPUT_EXCLUSIVE | ibmmq.MQOO_FAIL_IF_QUIESCING
		mqod.ObjectType = ibmmq.MQOT_Q
		ci.si.replyQ2BaseName = replyQ2
		if replyQ2 != "" {
			mqod.ObjectName = replyQ2
		} else {
			mqod.ObjectName = replyQ
		}
		ci.si.statusReplyQObj, err = ci.si.qMgr.Open(mqod, openOptions)
		if err != nil {
			errorString = "Cannot open queue " + mqod.ObjectName
			mqreturn = err.(*ibmmq.MQReturn)
		} else {
			clearQ(ci.si.statusReplyQObj)
		}
	}

	if err != nil {
		if mqreturn == nil {
			mqreturn = &ibmmq.MQReturn{MQCC: ibmmq.MQCC_WARNING, MQRC: ibmmq.MQRC_ENVIRONMENT_ERROR}
		}
		traceExitErr("InitConnection", 1, mqreturn)
		return MQMetricError{Err: errorString, MQReturn: mqreturn}
	}

	logTrace("initConnection: Queue manager resolved info - %+v", ci.si)
	traceExitErr("InitConnection", 0, mqreturn)

	return err
}

/*
EndConnection tidies up by closing the queues and disconnecting.
*/
func EndConnection() {
	traceEntry("EndConnection")

	ci := getConnection(GetConnectionKey())
	if ci == nil {
		traceExit("EndConnection", 1)
		return
	}
	m := GetPublishedMetrics(GetConnectionKey())
	// MQCLOSE all subscriptions
	if ci.si.subsOpened {
		for _, cl := range m.Classes {
			for _, ty := range cl.Types {
				for _, hObj := range ty.subHobj {
					hObj.Close(0)
				}
			}
		}
	}

	// MQCLOSE the queues
	if ci.si.queuesOpened {
		ci.si.cmdQObj.Close(0)
		ci.si.replyQObj.Close(0)
		ci.si.statusReplyQObj.Close(0)
		ci.si.qMgrObject.Close(0)
	}

	// MQDISC regardless of other errors
	if ci.si.qmgrConnected {
		ci.si.qMgr.Disc()
	}

	traceExit("EndConnection", 0)
}

/*
getMessage returns a message from the replyQ. The only
parameter to the function says whether this should block
for 30 seconds or return immediately if there is no message
available. When working with the command queue, blocking is
required; when getting publications, non-blocking is better.

A 32K buffer was created at the top of this file, and should always
be big enough for what we are expecting.
*/
func getMessage(wait bool) ([]byte, error) {
	traceEntry("getMessage")
	ci := getConnection(GetConnectionKey())

	rc, err := getMessageWithHObj(wait, ci.si.replyQObj)
	traceExitErr("getMessage", 0, err)
	return rc, err
}

func getMessageWithHObj(wait bool, hObj ibmmq.MQObject) ([]byte, error) {
	var err error
	var datalen int

	traceEntry("getMessageWithHObj")
	getmqmd := ibmmq.NewMQMD()
	gmo := ibmmq.NewMQGMO()
	gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT
	gmo.Options |= ibmmq.MQGMO_FAIL_IF_QUIESCING
	gmo.Options |= ibmmq.MQGMO_CONVERT

	gmo.MatchOptions = ibmmq.MQMO_NONE

	if wait {
		gmo.Options |= ibmmq.MQGMO_WAIT
		gmo.WaitInterval = 30 * 1000
	}

	datalen, err = hObj.Get(getmqmd, gmo, getBuffer)

	traceExitErr("getMessageWithHObj", 0, err)

	return getBuffer[0:datalen], err
}

/*
subscribe to the nominated topic. The previously-opened
replyQ is used for publications; we do not use a managed queue here,
so that everything can be read from one queue. The object handle for the
subscription is returned so we can close it when it's no longer needed.
*/
func subscribe(topic string, pubQObj *ibmmq.MQObject) (ibmmq.MQObject, error) {
	return subscribeWithOptions(topic, pubQObj, false)
}

/*
subscribe to the nominated topic, but ask the queue manager to
allocate the replyQ for us
*/
func subscribeManaged(topic string, pubQObj *ibmmq.MQObject) (ibmmq.MQObject, error) {
	return subscribeWithOptions(topic, pubQObj, true)
}

func subscribeWithOptions(topic string, pubQObj *ibmmq.MQObject, managed bool) (ibmmq.MQObject, error) {
	var err error

	traceEntry("subscribeWithOptions")
	ci := getConnection(GetConnectionKey())

	mqsd := ibmmq.NewMQSD()
	mqsd.Options = ibmmq.MQSO_CREATE
	mqsd.Options |= ibmmq.MQSO_NON_DURABLE
	mqsd.Options |= ibmmq.MQSO_FAIL_IF_QUIESCING
	if managed {
		mqsd.Options |= ibmmq.MQSO_MANAGED
	}

	mqsd.ObjectString = topic

	subObj, err := ci.si.qMgr.Sub(mqsd, pubQObj)
	if err != nil {
		extraInfo := ""
		mqrc := err.(*ibmmq.MQReturn).MQRC
		switch mqrc {
		case ibmmq.MQRC_HANDLE_NOT_AVAILABLE:
			extraInfo = "You may need to increase the MAXHANDS attribute on the queue manager."
		}
		e2 := fmt.Errorf("Error subscribing to topic '%s': %v %s", topic, err, extraInfo)
		traceExitErr("subscribeWithOptions", 1, e2)
		return subObj, e2
	}

	traceExitErr("subscribeWithOptions", 0, err)
	return subObj, err
}

/*
Return the current platform - the MQPL_* definition value. It
can be turned into a string if necessary via ibmmq.MQItoString("PL"...)
*/
func GetPlatform() int32 {
	ci := getConnection(GetConnectionKey())
	return ci.si.platform
}

/*
Return the current command level
*/
func GetCommandLevel() int32 {
	ci := getConnection(GetConnectionKey())
	return ci.si.commandLevel
}
