package ibmmq

/*
  Copyright (c) IBM Corporation 2016,2026

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
#include <stdlib.h>
#include <cmqc.h>
#include <cmqcfc.h>
*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

/*
 * The structures in this module help with the implementation of both trigger
 * monitors and triggered applications written in Go. Though there's no requirement for
 * both elements to be in the same language. We could have a Go trigger monitor starting
 * a Java program.
 *
 * A trigger monitor program reads messages from the initQ using MQGET. These messages
 * are mapped into the MQTM structure, assuming here that the format is MQFMT_TRIGGER:
 *     buffer, datalen, err = qObject.GetSlice(getmqmd, gmo, buffer)
 *     tm, err := ibmmq.NewMQTM(buffer)
 *
 * The monitor program extracts any relevant fields from the structure to work out what to
 * call, and also setting any parameters for the triggered program. The standard mechanism
 * requires a string version of the MQTMC (version2) structure. The qmgr name
 * must be added to the TMC, so that's an extra parameter when flattening the fields.
 *     app := tm.ApplId
 *	   tmcString := tm.ToTMC(qMgrName)
 *	   cmd := exec.Command(app, tmcString) // Or some env-specific way of starting the app
 *     cmd.Run()
 *
 * The triggered program can use the MQTMC structure passed on the command line to determine
 * which qmgr and queue to use:
 *     s := argv[1]
 *     tmc := NewMQTMC(s)
 *     MQCONN(tmc.QMgrName)
 *     ...
 * Trigger monitors and triggers apps may "cooperate" to agree on starting methods and any other
 * special parameters that may be passed, but the mechanism here follows the standard pattern
 */

// This is the format as read from the queue
type MQTM struct {
	version     int32 // no need for this to be public
	QName       string
	ProcessName string
	TriggerData string
	ApplType    int32
	ApplId      string
	EnvData     string
	UserData    string
}

// TMC has the same fields as the TM, but ints become 4-char strings
// And V2 of the structure also adds the qmgrName.
type MQTMC2 struct {
	Version     string
	QName       string
	ProcessName string
	TriggerData string
	ApplType    string
	ApplId      string
	EnvData     string
	UserData    string
	QMgrName    string
}

// Create a new MQTM structure. This will normally be initialised by passing in
// a buffer containing the corresponding bytes that have to be parsed into the
// Go structure fields. All strings are returned as trimmed.
func NewMQTM(buf []byte) (*MQTM, error) {
	var version int32
	var strucid string

	tm := new(MQTM)
	tm.version = MQTM_VERSION_1

	// Make sure this package-wide value is set
	if (C.MQENC_NATIVE % 2) == 0 {
		endian = binary.LittleEndian
	} else {
		endian = binary.BigEndian
	}

	if buf == nil {
		tm.ApplType = 0
		// Other public strings default to empty
	} else {
		// There's only been a single version of this structure in the MQI. Make sure the buffer
		// is long enough to work with.
		if len(buf) != int(MQTM_CURRENT_LENGTH) {
			err := MQReturn{MQCC: MQCC_FAILED, MQRC: MQRC_TM_ERROR}
			return nil, &err
		} else {
			// Note that readStringFromFixedBuffer returns trimmed versions of the strings
			r := bytes.NewBuffer(buf)
			strucid = readStringFromFixedBuffer(r, 4)
			if strucid != "TM" { // The trimmed version
				// Unexpected strucid
				err := MQReturn{MQCC: MQCC_FAILED, MQRC: MQRC_TM_ERROR}
				return nil, &err
			}
			binary.Read(r, endian, &version)
			tm.QName = readStringFromFixedBuffer(r, MQ_Q_NAME_LENGTH)
			tm.ProcessName = readStringFromFixedBuffer(r, MQ_PROCESS_NAME_LENGTH)
			tm.TriggerData = readStringFromFixedBuffer(r, MQ_TRIGGER_DATA_LENGTH)
			binary.Read(r, endian, &tm.ApplType)
			tm.ApplId = readStringFromFixedBuffer(r, MQ_PROCESS_APPL_ID_LENGTH)
			tm.EnvData = readStringFromFixedBuffer(r, MQ_PROCESS_ENV_DATA_LENGTH)
			tm.UserData = readStringFromFixedBuffer(r, MQ_PROCESS_USER_DATA_LENGTH)
		}
	}

	return tm, nil
}

// Either create a completely new MQTMC2 (give "" as the input string), or extract the fields
// from the string given to the triggered app on the command line. All the elements
// have a fixed length in the flattened string; they have spaces trimmed for the Go TMC2 structure.
func NewMQTMC2(s string) (*MQTMC2, error) {
	tmc := new(MQTMC2)
	if s == "" {
		tmc.Version = fmt.Sprintf("%4d", 2) // MQTMC_CURRENT_VERSION is a string, not int
		// All other public strings default to empty
	} else {

		// fmt.Printf("Extracting fields from string len:%d val:\"%s\"\n", len(s), s)
		if len(s) < int(MQTMC2_LENGTH_1) {
			err := MQReturn{MQCC: MQCC_FAILED, MQRC: MQRC_TMC_ERROR}
			return nil, &err
		}

		o := int32(0)
		strucid := s[o : o+4]
		o += 4
		if strucid != "TMC " {
			err := MQReturn{MQCC: MQCC_FAILED, MQRC: MQRC_TMC_ERROR}
			return nil, &err
		}

		tmc.Version = strings.TrimSpace(s[o : o+4])
		o += 4
		tmc.QName = strings.TrimSpace(s[o : o+MQ_Q_NAME_LENGTH])
		o += MQ_Q_NAME_LENGTH
		tmc.ProcessName = strings.TrimSpace(s[o : o+MQ_PROCESS_NAME_LENGTH])
		o += MQ_PROCESS_NAME_LENGTH
		tmc.TriggerData = strings.TrimSpace(s[o : o+MQ_TRIGGER_DATA_LENGTH])
		o += MQ_TRIGGER_DATA_LENGTH
		tmc.ApplType = strings.TrimSpace(s[o : o+4])
		o += 4
		tmc.ApplId = strings.TrimSpace(s[o : o+MQ_PROCESS_APPL_ID_LENGTH])
		o += MQ_PROCESS_APPL_ID_LENGTH
		tmc.EnvData = strings.TrimSpace(s[o : o+MQ_PROCESS_ENV_DATA_LENGTH])
		o += MQ_PROCESS_ENV_DATA_LENGTH
		tmc.UserData = strings.TrimSpace(s[o : o+MQ_PROCESS_USER_DATA_LENGTH])
		o += MQ_PROCESS_USER_DATA_LENGTH
		if len(s) >= int(MQTMC2_LENGTH_2) {
			tmc.QMgrName = strings.TrimSpace(s[o : o+MQ_Q_MGR_NAME_LENGTH])
			o += MQ_Q_MGR_NAME_LENGTH
		}
	}

	return tmc, nil
}

// Convert an MQTM (read by a trigger monitor) to the char-based version that can be passed to an app on the command line
// as a string. The elements are right-padded with spaces to get fixed lengths.
func (tm *MQTM) ToTMC2(qMgrName string) string {
	s := "TMC "                // StrucId
	s += fmt.Sprintf("%4d", 2) // MQTMC_CURRENT_VERSION is a string
	s += fmt.Sprintf("%-*s", MQ_Q_NAME_LENGTH, tm.QName)
	s += fmt.Sprintf("%-*s", MQ_PROCESS_NAME_LENGTH, tm.ProcessName)
	s += fmt.Sprintf("%-*s", MQ_TRIGGER_DATA_LENGTH, tm.TriggerData)
	s += fmt.Sprintf("%4d", tm.ApplType)
	s += fmt.Sprintf("%-*s", MQ_PROCESS_APPL_ID_LENGTH, tm.ApplId)
	s += fmt.Sprintf("%-*s", MQ_PROCESS_ENV_DATA_LENGTH, tm.EnvData)
	s += fmt.Sprintf("%-*s", MQ_PROCESS_USER_DATA_LENGTH, tm.UserData)

	s += fmt.Sprintf("%-*s", MQ_Q_MGR_NAME_LENGTH, qMgrName)

	// fmt.Printf("Created TMC2 of length %d: \"%s\"\n", len(s), s)
	return s
}
