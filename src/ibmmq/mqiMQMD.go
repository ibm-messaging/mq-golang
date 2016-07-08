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

import (
	"bytes"
)

/*
 * This file contains operations on the MQ Message Descriptor (MQMD)
 *
 */

type MQMD struct {
	StrucId          string
	Version          C.MQLONG
	Report           C.MQLONG
	MsgType          C.MQLONG
	Expiry           C.MQLONG
	Feedback         C.MQLONG
	Encoding         C.MQLONG
	CodedCharSetId   C.MQLONG
	Format           string
	Priority         C.MQLONG
	Persistence      C.MQLONG
	MsgId            []byte
	CorrelId         []byte
	BackoutCount     C.MQLONG
	ReplyToQ         string
	ReplyToQMgr      string
	UserIdentifier   string
	AccountingToken  []byte
	ApplIdentityData string
	PutApplType      C.MQLONG
	PutApplName      string
	PutDate          string
	PutTime          string
	ApplOriginData   string
	GroupId          []byte
	MsgSeqNumber     C.MQLONG
	Offset           C.MQLONG
	MsgFlags         C.MQLONG
	OriginalLength   C.MQLONG
}

func NewMQMD() *MQMD {
	md := new(MQMD)
	md.StrucId = "MD  "
	md.Version = C.MQMD_VERSION_1
	md.Report = C.MQRO_NONE
	md.MsgType = C.MQMT_DATAGRAM
	md.Expiry = C.MQEI_UNLIMITED
	md.Feedback = C.MQFB_NONE
	md.Encoding = C.MQENC_NATIVE
	md.CodedCharSetId = C.MQCCSI_Q_MGR
	md.Format = "        "
	md.Priority = C.MQPRI_PRIORITY_AS_Q_DEF
	md.Persistence = C.MQPER_PERSISTENCE_AS_Q_DEF
	md.MsgId = bytes.Repeat([]byte{0}, C.MQ_MSG_ID_LENGTH)
	md.CorrelId = bytes.Repeat([]byte{0}, C.MQ_CORREL_ID_LENGTH)
	md.BackoutCount = 0
	md.ReplyToQ = ""
	md.ReplyToQMgr = ""
	md.UserIdentifier = ""
	md.AccountingToken = bytes.Repeat([]byte{0}, C.MQ_ACCOUNTING_TOKEN_LENGTH)
	md.ApplIdentityData = ""
	md.PutApplType = C.MQAT_NO_CONTEXT
	md.PutApplName = ""
	md.PutDate = ""
	md.PutTime = ""
	md.ApplOriginData = ""
	md.GroupId = bytes.Repeat([]byte{0}, C.MQ_GROUP_ID_LENGTH)
	md.MsgSeqNumber = 1
	md.Offset = 0
	md.MsgFlags = C.MQMF_NONE
	md.OriginalLength = C.MQOL_UNDEFINED

	return md
}

func copyMDtoC(mqmd *C.MQMD, gomd *MQMD) {
	var i int
	setMQIString((*C.char)(&mqmd.StrucId[0]), gomd.StrucId, 4)
	mqmd.Version = gomd.Version
	mqmd.Report = gomd.Report
	mqmd.MsgType = gomd.MsgType
	mqmd.Expiry = gomd.Expiry
	mqmd.Feedback = gomd.Feedback
	mqmd.Encoding = gomd.Encoding
	mqmd.CodedCharSetId = gomd.CodedCharSetId
	setMQIString((*C.char)(&mqmd.Format[0]), gomd.Format, C.MQ_FORMAT_LENGTH)
	mqmd.Priority = gomd.Priority
	mqmd.Persistence = gomd.Persistence

	for i = 0; i < C.MQ_MSG_ID_LENGTH; i++ {
		mqmd.MsgId[i] = C.MQBYTE(gomd.MsgId[i])
	}
	for i = 0; i < C.MQ_CORREL_ID_LENGTH; i++ {
		mqmd.CorrelId[i] = C.MQBYTE(gomd.CorrelId[i])
	}
	mqmd.BackoutCount = gomd.BackoutCount

	setMQIString((*C.char)(&mqmd.ReplyToQ[0]), gomd.ReplyToQ, C.MQ_OBJECT_NAME_LENGTH)
	setMQIString((*C.char)(&mqmd.ReplyToQMgr[0]), gomd.ReplyToQMgr, C.MQ_OBJECT_NAME_LENGTH)

	setMQIString((*C.char)(&mqmd.UserIdentifier[0]), gomd.UserIdentifier, C.MQ_USER_ID_LENGTH)
	for i = 0; i < C.MQ_ACCOUNTING_TOKEN_LENGTH; i++ {
		mqmd.AccountingToken[i] = C.MQBYTE(gomd.AccountingToken[i])
	}
	setMQIString((*C.char)(&mqmd.ApplIdentityData[0]), gomd.ApplIdentityData, C.MQ_APPL_IDENTITY_DATA_LENGTH)
	mqmd.PutApplType = gomd.PutApplType
	setMQIString((*C.char)(&mqmd.PutApplName[0]), gomd.PutApplName, C.MQ_PUT_APPL_NAME_LENGTH)
	setMQIString((*C.char)(&mqmd.PutDate[0]), gomd.PutDate, C.MQ_PUT_DATE_LENGTH)
	setMQIString((*C.char)(&mqmd.PutTime[0]), gomd.PutTime, C.MQ_PUT_TIME_LENGTH)
	setMQIString((*C.char)(&mqmd.ApplOriginData[0]), gomd.ApplOriginData, C.MQ_APPL_ORIGIN_DATA_LENGTH)

	for i = 0; i < C.MQ_GROUP_ID_LENGTH; i++ {
		mqmd.GroupId[i] = C.MQBYTE(gomd.GroupId[i])
	}
	mqmd.MsgSeqNumber = gomd.MsgSeqNumber
	mqmd.Offset = gomd.Offset
	mqmd.MsgFlags = gomd.MsgFlags
	mqmd.OriginalLength = gomd.OriginalLength

	return
}

func copyMDfromC(mqmd *C.MQMD, gomd *MQMD) {
	var i int
	gomd.StrucId = C.GoStringN((*C.char)(&mqmd.StrucId[0]), 4)
	gomd.Version = mqmd.Version
	gomd.Report = mqmd.Report
	gomd.MsgType = mqmd.MsgType
	gomd.Expiry = mqmd.Expiry
	gomd.Feedback = mqmd.Feedback
	gomd.Encoding = mqmd.Encoding
	gomd.CodedCharSetId = mqmd.CodedCharSetId
	gomd.Format = C.GoStringN((*C.char)(&mqmd.Format[0]), C.MQ_FORMAT_LENGTH)
	gomd.Priority = mqmd.Priority
	gomd.Persistence = mqmd.Persistence

	for i = 0; i < C.MQ_MSG_ID_LENGTH; i++ {
		gomd.MsgId[i] = (byte)(mqmd.MsgId[i])
	}
	for i = 0; i < C.MQ_CORREL_ID_LENGTH; i++ {
		gomd.CorrelId[i] = (byte)(mqmd.CorrelId[i])
	}
	gomd.BackoutCount = mqmd.BackoutCount

	gomd.ReplyToQ = C.GoStringN((*C.char)(&mqmd.ReplyToQ[0]), C.MQ_OBJECT_NAME_LENGTH)
	gomd.ReplyToQMgr = C.GoStringN((*C.char)(&mqmd.ReplyToQMgr[0]), C.MQ_OBJECT_NAME_LENGTH)

	gomd.UserIdentifier = C.GoStringN((*C.char)(&mqmd.UserIdentifier[0]), C.MQ_USER_ID_LENGTH)
	for i = 0; i < C.MQ_ACCOUNTING_TOKEN_LENGTH; i++ {
		gomd.AccountingToken[i] = (byte)(mqmd.AccountingToken[i])
	}
	gomd.ApplIdentityData = C.GoStringN((*C.char)(&mqmd.ApplIdentityData[0]), C.MQ_APPL_IDENTITY_DATA_LENGTH)
	gomd.PutApplType = mqmd.PutApplType
	gomd.PutApplName = C.GoStringN((*C.char)(&mqmd.PutApplName[0]), C.MQ_PUT_APPL_NAME_LENGTH)
	gomd.PutDate = C.GoStringN((*C.char)(&mqmd.PutDate[0]), C.MQ_PUT_DATE_LENGTH)
	gomd.PutTime = C.GoStringN((*C.char)(&mqmd.PutTime[0]), C.MQ_PUT_TIME_LENGTH)
	gomd.ApplOriginData = C.GoStringN((*C.char)(&mqmd.ApplOriginData[0]), C.MQ_APPL_ORIGIN_DATA_LENGTH)

	for i = 0; i < C.MQ_GROUP_ID_LENGTH; i++ {
		gomd.GroupId[i] = (byte)(mqmd.GroupId[i])
	}
	gomd.MsgSeqNumber = mqmd.MsgSeqNumber
	gomd.Offset = mqmd.Offset
	gomd.MsgFlags = mqmd.MsgFlags
	gomd.OriginalLength = mqmd.OriginalLength

	return
}
