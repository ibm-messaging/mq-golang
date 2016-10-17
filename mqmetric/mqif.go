package mqmetric

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
This file holds most of the calls to the MQI, so we
don't need to repeat common setups eg of MQMD or MQSD structures.
*/

import (
	log "github.com/Sirupsen/logrus"
	"ibmmq"
)

var (
	qMgr      ibmmq.MQQueueManager
	cmdQObj   ibmmq.MQObject
	replyQObj ibmmq.MQObject
	getBuffer = make([]byte, 32768)

	qmgrConnected = false
	queuesOpened  = false
	subsOpened    = false
)

type ClientConfig struct {
	ClientMode bool
}

/*
InitConnection connects to the queuemanager, and then
opens both the command queue and a dynamic reply queue
to be used for all responses including the publications
*/
func InitConnection(qMgrName string, replyQ string, cc *ClientConfig) error {
	var err error
	gocno := ibmmq.NewMQCNO()

	if cc.ClientMode {
		gocno.Options = ibmmq.MQCNO_CLIENT_BINDING
	} else {
		gocno.Options = ibmmq.MQCNO_LOCAL_BINDING
	}
	gocno.Options |= ibmmq.MQCNO_HANDLE_SHARE_BLOCK

	qMgr, _, err = ibmmq.Connx(qMgrName, gocno)
	if err == nil {
		qmgrConnected = true
	}

	// MQOPEN of the COMMAND QUEUE
	if err == nil {
		mqod := ibmmq.NewMQOD()

		openOptions := ibmmq.MQOO_OUTPUT | ibmmq.MQOO_FAIL_IF_QUIESCING

		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = "SYSTEM.ADMIN.COMMAND.QUEUE"

		cmdQObj, _, err = qMgr.Open(mqod, openOptions)
		if err == nil {
			log.Infoln("Command queue open ok")
		}

	}

	// MQOPEN of a reply queue
	if err == nil {
		mqod := ibmmq.NewMQOD()
		openOptions := ibmmq.MQOO_INPUT_AS_Q_DEF | ibmmq.MQOO_FAIL_IF_QUIESCING
		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = replyQ
		replyQObj, _, err = qMgr.Open(mqod, openOptions)
		if err == nil {
			queuesOpened = true
			log.Infoln("Reply queue open ok")
		}
	}

	if err != nil {
		log.Errorf("Cannot access qmgr. Error %s", err)
	}

	return err

}

/*
EndConnection tidies up by closing the queues and disconnecting.
*/
func EndConnection() {

	// MQCLOSE all subscriptions
	if subsOpened {
		for _, cl := range Metrics.Classes {
			for _, ty := range cl.Types {
				for _, hObj := range ty.subHobj {
					hObj.Close(0)
				}
			}
		}
	}

	// MQCLOSE the queues
	if queuesOpened {
		cmdQObj.Close(0)
		replyQObj.Close(0)
	}

	// MQDISC regardless of other errors
	if qmgrConnected {
		qMgr.Disc()
	}

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
	var err error
	var datalen int
	var mqreturn ibmmq.MQReturn

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

	datalen, mqreturn, err = replyQObj.Get(getmqmd, gmo, getBuffer)

	if mqreturn.MQRC == ibmmq.MQRC_Q_MGR_NOT_AVAILABLE ||
		mqreturn.MQRC == ibmmq.MQRC_Q_MGR_NAME_ERROR ||
		mqreturn.MQRC == ibmmq.MQRC_Q_MGR_QUIESCING {
		log.Fatal("Queue Manager error: ", err)
	}
	if mqreturn.MQCC == ibmmq.MQCC_FAILED && mqreturn.MQRC != ibmmq.MQRC_NO_MSG_AVAILABLE {
		log.Error("Get message: ", err)
	}

	return getBuffer[0:datalen], err
}

/*
subscribe to the nominated topic. The previously-opened
replyQ is used for publications; we do not use a managed queue here,
so that everything can be read from one queue. The object handle for the
subscription is returned so we can close it when it's no longer needed.
*/
func subscribe(topic string) (ibmmq.MQObject, error) {
	var err error

	mqsd := ibmmq.NewMQSD()
	mqsd.Options = ibmmq.MQSO_CREATE
	mqsd.Options |= ibmmq.MQSO_NON_DURABLE
	mqsd.Options |= ibmmq.MQSO_FAIL_IF_QUIESCING

	mqsd.ObjectString = topic

	log.Infof("Subscribing to topic '%s'", topic)
	subObj, _, err := qMgr.Sub(mqsd, &replyQObj)
	if err != nil {
		log.Errorf("Error subscribing to topic '%s': %v", topic, err)
	}
	return subObj, err

}
