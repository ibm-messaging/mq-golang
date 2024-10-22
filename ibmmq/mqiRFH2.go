package ibmmq

/*
  Copyright (c) IBM Corporation 2024

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
)

type MQRFH2 struct {
	strucLength    int32
	Encoding       int32
	CodedCharSetId int32
	Format         string
	Flags          int32
	NameValueCCSID int32
}

func NewMQRFH2(md *MQMD) *MQRFH2 {

	rfh2 := new(MQRFH2)

	rfh2.CodedCharSetId = MQCCSI_INHERIT

	rfh2.Format = ""
	rfh2.Flags = MQRFH_NONE

	rfh2.strucLength = int32(MQRFH_STRUC_LENGTH_FIXED_2)

	if md != nil {

		rfh2.Encoding = md.Encoding
		if md.CodedCharSetId == MQCCSI_DEFAULT {
			rfh2.CodedCharSetId = MQCCSI_INHERIT
		} else {
			rfh2.CodedCharSetId = md.CodedCharSetId
		}
		rfh2.Format = md.Format

		md.Format = MQFMT_RF_HEADER_2
		md.CodedCharSetId = MQCCSI_Q_MGR
	}

	if (C.MQENC_NATIVE % 2) == 0 {
		endian = binary.LittleEndian
	} else {
		endian = binary.BigEndian
	}

	return rfh2
}

func (rfh2 *MQRFH2) Bytes() []byte {
	buf := make([]byte, rfh2.strucLength)
	offset := 0

	copy(buf[offset:], "RHF ")
	offset += 4
	endian.PutUint32(buf[offset:], uint32(MQRFH_VERSION_2))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(rfh2.strucLength))
	offset += 4

	endian.PutUint32(buf[offset:], uint32(rfh2.Encoding))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(rfh2.CodedCharSetId))
	offset += 4

	// Make sure the format is space padded to the correct length
	copy(buf[offset:], (rfh2.Format + space8)[0:8])
	offset += int(MQ_FORMAT_LENGTH)

	endian.PutUint32(buf[offset:], uint32(rfh2.Flags))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(rfh2.NameValueCCSID))
	offset += 4

	return buf
}

/*
We have a byte array for the message contents. The start of that buffer
is the MQRFH2 structure. We read the bytes from that fixed header to match
the C structure definition for each field. We will assume use of RFH v2
*/
func getHeaderRFH2(md *MQMD, buf []byte) (*MQRFH2, int, error) {

	var version int32

	rfh2 := NewMQRFH2(nil)

	r := bytes.NewBuffer(buf)
	_ = readStringFromFixedBuffer(r, 4) // StrucId
	binary.Read(r, endian, &version)
	binary.Read(r, endian, &rfh2.strucLength)

	binary.Read(r, endian, &rfh2.Encoding)
	binary.Read(r, endian, &rfh2.CodedCharSetId)

	rfh2.Format = readStringFromFixedBuffer(r, MQ_FORMAT_LENGTH)
	binary.Read(r, endian, &rfh2.Flags)
	binary.Read(r, endian, &rfh2.NameValueCCSID)

	return rfh2, int(rfh2.strucLength), nil
}

// Split the namevalue pairs in the RFH2 into a string array
// Each consists of a length/string duple, so we read the length
// and then the string. And repeat until the buffer is exhausted.
func GetRFH2Properties(hdr *MQRFH2, buf []byte) []string {
	var l int32
	props := make([]string, 0)
	r := bytes.NewBuffer(buf[MQRFH_STRUC_LENGTH_FIXED_2:])

	for offset := 0; offset < r.Len(); {
		binary.Read(r, endian, &l)
		offset += 4
		s := readStringFromFixedBuffer(r, l)
		props = append(props, s)
		offset += int(l)
	}
	return props
}
