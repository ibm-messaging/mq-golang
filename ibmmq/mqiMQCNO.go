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
#cgo CFLAGS: -I/opt/mqm/inc -D_REENTRANT

#include <stdlib.h>
#include <string.h>
#include <cmqc.h>

*/
import "C"
import "unsafe"

/*
MQCNO is a structure containing the MQ Connection Options (MQCNO)
Note that only a subset of the real structure is exposed in this
version.
*/
type MQCNO struct {
	Version       int32
	Options       int32
	SecurityParms *MQCSP
	CCDTUrl       string
}

/*
MQCSP is a structure containing the MQ Security Parameters (MQCSP)
*/
type MQCSP struct {
	AuthenticationType int32
	UserId             string
	Password           string
}

/*
NewMQCNO fills in default values for the MQCNO structure
*/
func NewMQCNO() *MQCNO {

	cno := new(MQCNO)
	cno.Version = int32(C.MQCNO_VERSION_1)
	cno.Options = int32(C.MQCNO_NONE)
	cno.SecurityParms = nil

	return cno
}

/*
NewMQCSP fills in default values for the MQCSP structure
*/
func NewMQCSP() *MQCSP {

	csp := new(MQCSP)
	csp.AuthenticationType = int32(C.MQCSP_AUTH_NONE)
	csp.UserId = ""
	csp.Password = ""

	return csp
}

func copyCNOtoC(mqcno *C.MQCNO, gocno *MQCNO) {
	var i int
	var mqcsp C.PMQCSP

	setMQIString((*C.char)(&mqcno.StrucId[0]), "CNO ", 4)
	mqcno.Version = C.MQLONG(gocno.Version)
	mqcno.Options = C.MQLONG(gocno.Options)

	mqcno.ClientConnOffset = 0
	mqcno.ClientConnPtr = nil

	for i = 0; i < C.MQ_CONN_TAG_LENGTH; i++ {
		mqcno.ConnTag[i] = 0
	}
	for i = 0; i < C.MQ_CONNECTION_ID_LENGTH; i++ {
		mqcno.ConnectionId[i] = 0
	}

	mqcno.SSLConfigOffset = 0
	mqcno.SSLConfigPtr = nil

	mqcno.SecurityParmsOffset = 0
	if gocno.SecurityParms != nil {
		gocsp := gocno.SecurityParms

		mqcsp = C.PMQCSP(C.malloc(C.MQCSP_CURRENT_LENGTH))
		setMQIString((*C.char)(&mqcsp.StrucId[0]), "CSP ", 4)
		mqcsp.Version = C.MQCSP_CURRENT_VERSION
		mqcsp.AuthenticationType = C.MQLONG(gocsp.AuthenticationType)
		mqcsp.CSPUserIdOffset = 0
		mqcsp.CSPPasswordOffset = 0

		if gocsp.UserId != "" {
			mqcsp.CSPUserIdPtr = C.MQPTR(unsafe.Pointer(C.CString(gocsp.UserId)))
			mqcsp.CSPUserIdLength = C.MQLONG(len(gocsp.UserId))
		}
		if gocsp.Password != "" {
			mqcsp.CSPPasswordPtr = C.MQPTR(unsafe.Pointer(C.CString(gocsp.Password)))
			mqcsp.CSPPasswordLength = C.MQLONG(len(gocsp.Password))
		}
		mqcno.SecurityParmsPtr = C.PMQCSP(mqcsp)

	} else {
		mqcno.SecurityParmsPtr = nil
	}

	mqcno.CCDTUrlOffset = 0
	if len(gocno.CCDTUrl) != 0 {
		mqcno.CCDTUrlPtr = C.PMQCHAR(unsafe.Pointer(C.CString(gocno.CCDTUrl)))
		mqcno.CCDTUrlLength = C.MQLONG(len(gocno.CCDTUrl))
	} else {
		mqcno.CCDTUrlPtr = nil
		mqcno.CCDTUrlLength = 0
	}
	return
}

func copyCNOfromC(mqcno *C.MQCNO, gocno *MQCNO) {

	if mqcno.SecurityParmsPtr != nil {
		if mqcno.SecurityParmsPtr.CSPUserIdPtr != nil {
			C.free(unsafe.Pointer(mqcno.SecurityParmsPtr.CSPUserIdPtr))
		}
		if mqcno.SecurityParmsPtr.CSPPasswordPtr != nil {
			C.free(unsafe.Pointer(mqcno.SecurityParmsPtr.CSPPasswordPtr))
		}
		C.free(unsafe.Pointer(mqcno.SecurityParmsPtr))
		// TODO - if userid/password set, C.free(them)
	}

	if mqcno.CCDTUrlPtr != nil {
		C.free(unsafe.Pointer(mqcno.CCDTUrlPtr))
	}
	return
}
