package ibmmq

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
#cgo CFLAGS: -I/opt/mqm/inc
#cgo LDFLAGS: -L/opt/mqm/lib64 -lmqm -m64

#include <stdlib.h>
#include <string.h>
#include <cmqc.h>

*/
import "C"

/*
 * This file contains operations on the MQ Put MessageOptions (MQPMO)
 *
 */

type MQPMO struct {
	StrucId           string
	Version           C.MQLONG
	Options           C.MQLONG
	Timeout           C.MQLONG
	Context           C.MQHOBJ
	KnownDestCount    C.MQLONG
	UnknownDestCount  C.MQLONG
	InvalidDestCount  C.MQLONG
	ResolvedQName     string
	ResolvedQMgrName  string
	RecsPresent       C.MQLONG
	PutMsgRecFields   C.MQLONG
	PutMsgRecOffset   C.MQLONG
	ResponseRecOffset C.MQLONG
	PutMsgRecPtr      C.MQPTR
	ResponseRecPtr    C.MQPTR

	OriginalMsgHandle C.MQHMSG
	NewMsgHandle      C.MQHMSG
	Action            C.MQLONG
	PubLevel          C.MQLONG
}

func NewMQPMO() *MQPMO {

	pmo := new(MQPMO)
	pmo.StrucId = "PMO "

	pmo.Version = C.MQPMO_VERSION_1
	pmo.Options = C.MQPMO_NONE
	pmo.Timeout = -1
	pmo.Context = 0
	pmo.KnownDestCount = 0
	pmo.UnknownDestCount = 0
	pmo.InvalidDestCount = 0
	pmo.ResolvedQName = ""
	pmo.ResolvedQMgrName = ""
	pmo.RecsPresent = 0
	pmo.PutMsgRecFields = 0
	pmo.PutMsgRecOffset = 0
	pmo.ResponseRecOffset = 0
	pmo.PutMsgRecPtr = nil
	pmo.ResponseRecPtr = nil

	pmo.OriginalMsgHandle = C.MQHM_NONE
	pmo.NewMsgHandle = C.MQHM_NONE
	pmo.Action = C.MQACTP_NEW
	pmo.PubLevel = 9

	return pmo
}

func copyPMOtoC(mqpmo *C.MQPMO, gopmo *MQPMO) {

	setMQIString((*C.char)(&mqpmo.StrucId[0]), gopmo.StrucId, 4)
	mqpmo.Version = gopmo.Version

	mqpmo.Options = gopmo.Options
	mqpmo.Timeout = gopmo.Timeout
	mqpmo.Context = gopmo.Context
	mqpmo.KnownDestCount = gopmo.KnownDestCount
	mqpmo.UnknownDestCount = gopmo.UnknownDestCount
	mqpmo.InvalidDestCount = gopmo.InvalidDestCount

	setMQIString((*C.char)(&mqpmo.ResolvedQName[0]), gopmo.ResolvedQName, C.MQ_OBJECT_NAME_LENGTH)
	setMQIString((*C.char)(&mqpmo.ResolvedQMgrName[0]), gopmo.ResolvedQMgrName, C.MQ_OBJECT_NAME_LENGTH)

	mqpmo.RecsPresent = gopmo.RecsPresent
	mqpmo.PutMsgRecFields = gopmo.PutMsgRecFields
	mqpmo.PutMsgRecOffset = gopmo.PutMsgRecOffset
	mqpmo.ResponseRecOffset = gopmo.ResponseRecOffset
	mqpmo.PutMsgRecPtr = gopmo.PutMsgRecPtr
	mqpmo.ResponseRecPtr = gopmo.ResponseRecPtr

	mqpmo.OriginalMsgHandle = gopmo.OriginalMsgHandle
	mqpmo.NewMsgHandle = gopmo.NewMsgHandle
	mqpmo.Action = gopmo.Action
	mqpmo.PubLevel = gopmo.PubLevel

	return
}

func copyPMOfromC(mqpmo *C.MQPMO, gopmo *MQPMO) {

	gopmo.StrucId = C.GoStringN((*C.char)(&mqpmo.StrucId[0]), 4)
	mqpmo.Version = gopmo.Version

	gopmo.Options = mqpmo.Options
	gopmo.Timeout = mqpmo.Timeout
	gopmo.Context = mqpmo.Context
	gopmo.KnownDestCount = mqpmo.KnownDestCount
	gopmo.UnknownDestCount = mqpmo.UnknownDestCount
	gopmo.InvalidDestCount = mqpmo.InvalidDestCount

	gopmo.ResolvedQName = C.GoStringN((*C.char)(&mqpmo.ResolvedQName[0]), C.MQ_OBJECT_NAME_LENGTH)
	gopmo.ResolvedQMgrName = C.GoStringN((*C.char)(&mqpmo.ResolvedQMgrName[0]), C.MQ_OBJECT_NAME_LENGTH)

	gopmo.RecsPresent = mqpmo.RecsPresent
	gopmo.PutMsgRecFields = mqpmo.PutMsgRecFields
	gopmo.PutMsgRecOffset = mqpmo.PutMsgRecOffset
	gopmo.ResponseRecOffset = mqpmo.ResponseRecOffset
	gopmo.PutMsgRecPtr = mqpmo.PutMsgRecPtr
	gopmo.ResponseRecPtr = mqpmo.ResponseRecPtr

	gopmo.OriginalMsgHandle = mqpmo.OriginalMsgHandle
	gopmo.NewMsgHandle = mqpmo.NewMsgHandle
	gopmo.Action = mqpmo.Action
	gopmo.PubLevel = mqpmo.PubLevel
	return
}
