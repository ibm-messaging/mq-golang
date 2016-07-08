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
import "bytes"

/*
 * This file contains operations on the MQ Get Message Options (MQGMO)
 *
 */

type MQGMO struct {
	StrucId        string
	Version        C.MQLONG
	Options        C.MQLONG
	WaitInterval   C.MQLONG
	Signal1        C.MQLONG
	Signal2        C.MQLONG
	ResolvedQName  string
	MatchOptions   C.MQLONG
	GroupStatus    C.MQCHAR
	SegmentStatus  C.MQCHAR
	Segmentation   C.MQCHAR
	Reserved1      C.MQCHAR
	MsgToken       []byte
	ReturnedLength C.MQLONG
	Reserved2      C.MQLONG
	MsgHandle      C.MQHMSG
}

func NewMQGMO() *MQGMO {

	gmo := new(MQGMO)
	gmo.StrucId = "GMO "
	gmo.Version = C.MQGMO_VERSION_1
	gmo.Options = C.MQGMO_NO_WAIT + C.MQGMO_PROPERTIES_AS_Q_DEF
	gmo.WaitInterval = C.MQWI_UNLIMITED
	gmo.Signal1 = 0
	gmo.Signal2 = 0
	gmo.ResolvedQName = ""
	gmo.MatchOptions = C.MQMO_MATCH_MSG_ID + C.MQMO_MATCH_CORREL_ID
	gmo.GroupStatus = C.MQGS_NOT_IN_GROUP
	gmo.SegmentStatus = C.MQSS_NOT_A_SEGMENT
	gmo.Segmentation = C.MQSEG_INHIBITED
	gmo.Reserved1 = ' '
	gmo.MsgToken = bytes.Repeat([]byte{0}, C.MQ_MSG_TOKEN_LENGTH)
	gmo.ReturnedLength = C.MQRL_UNDEFINED
	gmo.Reserved2 = 0
	gmo.MsgHandle = C.MQHM_NONE

	return gmo
}

func copyGMOtoC(mqgmo *C.MQGMO, gogmo *MQGMO) {
	var i int

	setMQIString((*C.char)(&mqgmo.StrucId[0]), gogmo.StrucId, 4)
	mqgmo.Version = gogmo.Version
	mqgmo.Options = gogmo.Options
	mqgmo.WaitInterval = gogmo.WaitInterval
	mqgmo.Signal1 = gogmo.Signal1
	mqgmo.Signal2 = gogmo.Signal2
	setMQIString((*C.char)(&mqgmo.ResolvedQName[0]), gogmo.ResolvedQName, C.MQ_OBJECT_NAME_LENGTH)
	mqgmo.MatchOptions = gogmo.MatchOptions
	mqgmo.GroupStatus = gogmo.GroupStatus
	mqgmo.SegmentStatus = gogmo.SegmentStatus
	mqgmo.Segmentation = gogmo.Segmentation
	mqgmo.Reserved1 = gogmo.Reserved1
	for i = 0; i < C.MQ_MSG_TOKEN_LENGTH; i++ {
		mqgmo.MsgToken[i] = C.MQBYTE(gogmo.MsgToken[i])
	}
	mqgmo.ReturnedLength = gogmo.ReturnedLength
	mqgmo.Reserved2 = gogmo.Reserved2
	mqgmo.MsgHandle = gogmo.MsgHandle
	return
}

func copyGMOfromC(mqgmo *C.MQGMO, gogmo *MQGMO) {
	var i int

	gogmo.StrucId = C.GoStringN((*C.char)(&mqgmo.StrucId[0]), 4)
	gogmo.Version = mqgmo.Version
	gogmo.Options = mqgmo.Options
	gogmo.WaitInterval = mqgmo.WaitInterval
	gogmo.Signal1 = mqgmo.Signal1
	gogmo.Signal2 = mqgmo.Signal2
	gogmo.ResolvedQName = C.GoStringN((*C.char)(&mqgmo.ResolvedQName[0]), C.MQ_OBJECT_NAME_LENGTH)
	gogmo.MatchOptions = mqgmo.MatchOptions
	gogmo.GroupStatus = mqgmo.GroupStatus
	gogmo.SegmentStatus = mqgmo.SegmentStatus
	gogmo.Segmentation = mqgmo.Segmentation
	gogmo.Reserved1 = mqgmo.Reserved1
	for i = 0; i < C.MQ_MSG_TOKEN_LENGTH; i++ {
		gogmo.MsgToken[i] = (byte)(mqgmo.MsgToken[i])
	}
	gogmo.ReturnedLength = mqgmo.ReturnedLength
	gogmo.Reserved2 = mqgmo.Reserved2
	gogmo.MsgHandle = mqgmo.MsgHandle
	return
}
