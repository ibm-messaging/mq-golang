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
MQMD is a structure containing the MQ Message Descriptor (MQMD)
*/
type MQMD struct {
	StrucId          string
	Version          int
	Report           int
	MsgType          int
	Expiry           int
	Feedback         int
	Encoding         int
	CodedCharSetId   int
	Format           string
	Priority         int
	Persistence      int
	MsgId            []byte
	CorrelId         []byte
	BackoutCount     int
	ReplyToQ         string
	ReplyToQMgr      string
	UserIdentifier   string
	AccountingToken  []byte
	ApplIdentityData string
	PutApplType      int
	PutApplName      string
	PutDate          string
	PutTime          string
	ApplOriginData   string
	GroupId          []byte
	MsgSeqNumber     int
	Offset           int
	MsgFlags         int
	OriginalLength   int
}

/*
NewMQMD fills in default values for the MQMD structure
*/
func NewMQMD() *MQMD {
	md := new(MQMD)
	md.StrucId = "MD  "
	md.Version = int(C.MQMD_VERSION_1)
	md.Report = int(C.MQRO_NONE)
	md.MsgType = int(C.MQMT_DATAGRAM)
	md.Expiry = int(C.MQEI_UNLIMITED)
	md.Feedback = int(C.MQFB_NONE)
	md.Encoding = int(C.MQENC_NATIVE)
	md.CodedCharSetId = int(C.MQCCSI_Q_MGR)
	md.Format = "        "
	md.Priority = int(C.MQPRI_PRIORITY_AS_Q_DEF)
	md.Persistence = int(C.MQPER_PERSISTENCE_AS_Q_DEF)
	md.MsgId = bytes.Repeat([]byte{0}, C.MQ_MSG_ID_LENGTH)
	md.CorrelId = bytes.Repeat([]byte{0}, C.MQ_CORREL_ID_LENGTH)
	md.BackoutCount = 0
	md.ReplyToQ = ""
	md.ReplyToQMgr = ""
	md.UserIdentifier = ""
	md.AccountingToken = bytes.Repeat([]byte{0}, C.MQ_ACCOUNTING_TOKEN_LENGTH)
	md.ApplIdentityData = ""
	md.PutApplType = int(C.MQAT_NO_CONTEXT)
	md.PutApplName = ""
	md.PutDate = ""
	md.PutTime = ""
	md.ApplOriginData = ""
	md.GroupId = bytes.Repeat([]byte{0}, C.MQ_GROUP_ID_LENGTH)
	md.MsgSeqNumber = 1
	md.Offset = 0
	md.MsgFlags = int(C.MQMF_NONE)
	md.OriginalLength = int(C.MQOL_UNDEFINED)

	return md
}

func copyMDtoC(mqmd *C.MQMD, gomd *MQMD) {
	var i int
	setMQIString((*C.char)(&mqmd.StrucId[0]), gomd.StrucId, 4)
	mqmd.Version = C.MQLONG(gomd.Version)
	mqmd.Report = C.MQLONG(gomd.Report)
	mqmd.MsgType = C.MQLONG(gomd.MsgType)
	mqmd.Expiry = C.MQLONG(gomd.Expiry)
	mqmd.Feedback = C.MQLONG(gomd.Feedback)
	mqmd.Encoding = C.MQLONG(gomd.Encoding)
	mqmd.CodedCharSetId = C.MQLONG(gomd.CodedCharSetId)
	setMQIString((*C.char)(&mqmd.Format[0]), gomd.Format, C.MQ_FORMAT_LENGTH)
	mqmd.Priority = C.MQLONG(gomd.Priority)
	mqmd.Persistence = C.MQLONG(gomd.Persistence)

	for i = 0; i < C.MQ_MSG_ID_LENGTH; i++ {
		mqmd.MsgId[i] = C.MQBYTE(gomd.MsgId[i])
	}
	for i = 0; i < C.MQ_CORREL_ID_LENGTH; i++ {
		mqmd.CorrelId[i] = C.MQBYTE(gomd.CorrelId[i])
	}
	mqmd.BackoutCount = C.MQLONG(gomd.BackoutCount)

	setMQIString((*C.char)(&mqmd.ReplyToQ[0]), gomd.ReplyToQ, C.MQ_OBJECT_NAME_LENGTH)
	setMQIString((*C.char)(&mqmd.ReplyToQMgr[0]), gomd.ReplyToQMgr, C.MQ_OBJECT_NAME_LENGTH)

	setMQIString((*C.char)(&mqmd.UserIdentifier[0]), gomd.UserIdentifier, C.MQ_USER_ID_LENGTH)
	for i = 0; i < C.MQ_ACCOUNTING_TOKEN_LENGTH; i++ {
		mqmd.AccountingToken[i] = C.MQBYTE(gomd.AccountingToken[i])
	}
	setMQIString((*C.char)(&mqmd.ApplIdentityData[0]), gomd.ApplIdentityData, C.MQ_APPL_IDENTITY_DATA_LENGTH)
	mqmd.PutApplType = C.MQLONG(gomd.PutApplType)
	setMQIString((*C.char)(&mqmd.PutApplName[0]), gomd.PutApplName, C.MQ_PUT_APPL_NAME_LENGTH)
	setMQIString((*C.char)(&mqmd.PutDate[0]), gomd.PutDate, C.MQ_PUT_DATE_LENGTH)
	setMQIString((*C.char)(&mqmd.PutTime[0]), gomd.PutTime, C.MQ_PUT_TIME_LENGTH)
	setMQIString((*C.char)(&mqmd.ApplOriginData[0]), gomd.ApplOriginData, C.MQ_APPL_ORIGIN_DATA_LENGTH)

	for i = 0; i < C.MQ_GROUP_ID_LENGTH; i++ {
		mqmd.GroupId[i] = C.MQBYTE(gomd.GroupId[i])
	}
	mqmd.MsgSeqNumber = C.MQLONG(gomd.MsgSeqNumber)
	mqmd.Offset = C.MQLONG(gomd.Offset)
	mqmd.MsgFlags = C.MQLONG(gomd.MsgFlags)
	mqmd.OriginalLength = C.MQLONG(gomd.OriginalLength)

	return
}

func copyMDfromC(mqmd *C.MQMD, gomd *MQMD) {
	var i int
	gomd.StrucId = C.GoStringN((*C.char)(&mqmd.StrucId[0]), 4)
	gomd.Version = int(mqmd.Version)
	gomd.Report = int(mqmd.Report)
	gomd.MsgType = int(mqmd.MsgType)
	gomd.Expiry = int(mqmd.Expiry)
	gomd.Feedback = int(mqmd.Feedback)
	gomd.Encoding = int(mqmd.Encoding)
	gomd.CodedCharSetId = int(mqmd.CodedCharSetId)
	gomd.Format = C.GoStringN((*C.char)(&mqmd.Format[0]), C.MQ_FORMAT_LENGTH)
	gomd.Priority = int(mqmd.Priority)
	gomd.Persistence = int(mqmd.Persistence)

	for i = 0; i < C.MQ_MSG_ID_LENGTH; i++ {
		gomd.MsgId[i] = (byte)(mqmd.MsgId[i])
	}
	for i = 0; i < C.MQ_CORREL_ID_LENGTH; i++ {
		gomd.CorrelId[i] = (byte)(mqmd.CorrelId[i])
	}
	gomd.BackoutCount = int(mqmd.BackoutCount)

	gomd.ReplyToQ = C.GoStringN((*C.char)(&mqmd.ReplyToQ[0]), C.MQ_OBJECT_NAME_LENGTH)
	gomd.ReplyToQMgr = C.GoStringN((*C.char)(&mqmd.ReplyToQMgr[0]), C.MQ_OBJECT_NAME_LENGTH)

	gomd.UserIdentifier = C.GoStringN((*C.char)(&mqmd.UserIdentifier[0]), C.MQ_USER_ID_LENGTH)
	for i = 0; i < C.MQ_ACCOUNTING_TOKEN_LENGTH; i++ {
		gomd.AccountingToken[i] = (byte)(mqmd.AccountingToken[i])
	}
	gomd.ApplIdentityData = C.GoStringN((*C.char)(&mqmd.ApplIdentityData[0]), C.MQ_APPL_IDENTITY_DATA_LENGTH)
	gomd.PutApplType = int(mqmd.PutApplType)
	gomd.PutApplName = C.GoStringN((*C.char)(&mqmd.PutApplName[0]), C.MQ_PUT_APPL_NAME_LENGTH)
	gomd.PutDate = C.GoStringN((*C.char)(&mqmd.PutDate[0]), C.MQ_PUT_DATE_LENGTH)
	gomd.PutTime = C.GoStringN((*C.char)(&mqmd.PutTime[0]), C.MQ_PUT_TIME_LENGTH)
	gomd.ApplOriginData = C.GoStringN((*C.char)(&mqmd.ApplOriginData[0]), C.MQ_APPL_ORIGIN_DATA_LENGTH)

	for i = 0; i < C.MQ_GROUP_ID_LENGTH; i++ {
		gomd.GroupId[i] = (byte)(mqmd.GroupId[i])
	}
	gomd.MsgSeqNumber = int(mqmd.MsgSeqNumber)
	gomd.Offset = int(mqmd.Offset)
	gomd.MsgFlags = int(mqmd.MsgFlags)
	gomd.OriginalLength = int(mqmd.OriginalLength)

	return
}
