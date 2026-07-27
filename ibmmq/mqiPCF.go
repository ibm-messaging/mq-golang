package ibmmq

/*
  Copyright (c) IBM Corporation 2016,2024

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
	"encoding/hex"
	"fmt"
	"runtime/debug"
	"strings"
)

/*
MQCFH is a structure containing the MQ PCF Header fields
*/
type MQCFH struct {
	Type           int32
	StrucLength    int32
	Version        int32
	Command        int32
	MsgSeqNumber   int32
	Control        int32
	CompCode       int32
	Reason         int32
	ParameterCount int32
}

/*
MQEPH is a structure containing the MQ PCF Embedded Header fields
It is always followed by a CFH which we do not include here
*/
type MQEPH struct {
	Version        int32
	StrucLength    int32
	Encoding       int32
	CodedCharSetId int32
	Format         string
	Control        int32
	Flags          int32
	Cfh            MQCFH
}

/*
PCFParameter is a structure containing the data associated with
various types of PCF element. Use the Type field to decide which
of the data fields is relevant.
*/
type PCFFilter struct {
	Type        int32
	Parameter   int32
	Operator    int32
	FilterValue interface{}
}
type PCFParameter struct {
	Type           int32
	Parameter      int32
	Int64Value     []int64 // Always store as 64; cast to 32 when needed
	String         []string
	CodedCharSetId int32
	ParameterCount int32
	GroupList      []*PCFParameter
	Filter         PCFFilter
	strucLength    int32 // Do not need to expose these
	stringLength   int32 // lengths
}

/*
NewMQCFH returns a PCF Command Header structure with correct initialisation
*/
func NewMQCFH() *MQCFH {
	cfh := new(MQCFH)
	cfh.Type = C.MQCFT_COMMAND
	cfh.StrucLength = C.MQCFH_STRUC_LENGTH
	cfh.Version = C.MQCFH_VERSION_1
	cfh.Command = C.MQCMD_NONE
	cfh.MsgSeqNumber = 1
	cfh.Control = C.MQCFC_LAST
	cfh.CompCode = C.MQCC_OK
	cfh.Reason = C.MQRC_NONE
	cfh.ParameterCount = 0

	return cfh
}

/*
NewMQEPH returns a PCF Embedded Header structure with correct initialisation
*/
func NewMQEPH() *MQEPH {
	eph := new(MQEPH)
	eph.StrucLength = C.MQEPH_STRUC_LENGTH_FIXED
	eph.Version = C.MQEPH_VERSION_1
	eph.Encoding = 0
	eph.CodedCharSetId = C.MQCCSI_UNDEFINED
	eph.Format = C.MQFMT_NONE
	eph.Flags = C.MQEPH_NONE

	eph.Cfh.Type = C.MQCFT_NONE
	eph.Cfh.StrucLength = C.MQCFH_STRUC_LENGTH
	eph.Cfh.Version = C.MQCFH_VERSION_3
	eph.Cfh.Command = C.MQCMD_NONE
	eph.Cfh.MsgSeqNumber = 1
	eph.Cfh.Control = C.MQCFC_LAST
	eph.Cfh.CompCode = C.MQCC_OK
	eph.Cfh.Reason = C.MQRC_NONE
	eph.Cfh.ParameterCount = 0

	return eph
}

/*
Bytes serialises an MQCFH structure as if it were the corresponding C structure
*/
func (cfh *MQCFH) Bytes() []byte {

	buf := make([]byte, cfh.StrucLength)
	offset := 0

	endian.PutUint32(buf[offset:], uint32(cfh.Type))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(cfh.StrucLength))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(cfh.Version))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(cfh.Command))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(cfh.MsgSeqNumber))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(cfh.Control))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(cfh.CompCode))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(cfh.Reason))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(cfh.ParameterCount))
	offset += 4

	return buf
}

func (eph *MQEPH) Bytes() []byte {

	// There's no constant defining the length of just the "EPH" wrapper.
	// The STRUC_LENGTH_FIXED includes the CFH length. So we have to start
	// by calculating it
	buf := make([]byte, C.MQEPH_STRUC_LENGTH_FIXED-C.MQCFH_STRUC_LENGTH)
	offset := 0

	copy(buf[offset:], "EPH ")
	offset += 4
	endian.PutUint32(buf[offset:], uint32(eph.Version))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(eph.StrucLength))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(eph.Encoding))
	offset += 4
	endian.PutUint32(buf[offset:], uint32(eph.CodedCharSetId))
	offset += 4
	copy(buf[offset:], (eph.Format + space8)[0:8])
	offset += 8
	endian.PutUint32(buf[offset:], uint32(eph.Flags))
	offset += 4

	buf = append(buf, eph.Cfh.Bytes()...)

	return buf
}

/*
Bytes serialises a PCFParameter into the C structure
corresponding to its type.

TODO: Only a subset of the PCF
parameter types are handled here - those needed for
command queries. Other types could be added if
necessary later.
*/
func (p *PCFParameter) Bytes() []byte {
	var buf []byte

	switch p.Type {
	case C.MQCFT_GROUP:
		buf = make([]byte, C.MQCFGR_STRUC_LENGTH)
		offset := 0
		l := len(p.GroupList)

		endian.PutUint32(buf[offset:], uint32(p.Type))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(len(buf)))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(p.Parameter))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(l))
		offset += 4
		for i := 0; i < l; i++ {
			buf = append(buf, p.GroupList[i].Bytes()...)
		}

	case C.MQCFT_INTEGER:
		buf = make([]byte, C.MQCFIN_STRUC_LENGTH)
		offset := 0

		endian.PutUint32(buf[offset:], uint32(p.Type))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(len(buf)))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(p.Parameter))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(p.Int64Value[0]))
		offset += 4

	case C.MQCFT_INTEGER_LIST:
		l := len(p.Int64Value)
		buf = make([]byte, C.MQCFIL_STRUC_LENGTH_FIXED+4*l)
		offset := 0

		endian.PutUint32(buf[offset:], uint32(p.Type))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(len(buf)))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(p.Parameter))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(l))
		offset += 4
		for i := 0; i < l; i++ {
			endian.PutUint32(buf[offset:], uint32(p.Int64Value[i]))
			offset += 4
		}

	case C.MQCFT_INTEGER_FILTER:
		buf = make([]byte, C.MQCFIF_STRUC_LENGTH)
		offset := 0

		endian.PutUint32(buf[offset:], uint32(p.Type))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(len(buf)))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(p.Filter.Parameter))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(p.Filter.Operator))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(p.Filter.FilterValue.(int32)))
		offset += 4

	case C.MQCFT_STRING:
		buf = make([]byte, C.MQCFST_STRUC_LENGTH_FIXED+roundTo4(int32(len(p.String[0]))))
		offset := 0
		endian.PutUint32(buf[offset:], uint32(p.Type))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(len(buf)))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(p.Parameter))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(C.MQCCSI_DEFAULT))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(len(p.String[0])))
		offset += 4
		copy(buf[offset:], []byte(p.String[0]))

	case C.MQCFT_STRING_FILTER:
		// Use "\000" as the string if you need an empty/null parameter
		fv := p.Filter.FilterValue.(string)

		buf = make([]byte, C.MQCFSF_STRUC_LENGTH_FIXED+roundTo4(int32(len(fv))))
		offset := 0
		endian.PutUint32(buf[offset:], uint32(p.Type))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(len(buf)))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(p.Filter.Parameter))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(p.Filter.Operator))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(C.MQCCSI_DEFAULT))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(len(fv)))
		offset += 4
		copy(buf[offset:], []byte(fv))

	// Expect to be given a string that can be decoded into bytes. For example, "aabb12".
	// Return nil if the bytestring cannot be converted from the hex
	case C.MQCFT_BYTE_STRING:
		bs, err := hex.DecodeString(p.String[0])
		if err != nil {
			// Can't easily change to return or report the error without breaking existing apps.
			logError("Trying to serialise PCF ByteString parameter \"%s\" : %v\n", p.String[0], err)
			return nil
		}
		buf = make([]byte, C.MQCFST_STRUC_LENGTH_FIXED+roundTo4(int32(len(bs))))
		offset := 0
		endian.PutUint32(buf[offset:], uint32(p.Type))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(len(buf)))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(p.Parameter))
		offset += 4
		endian.PutUint32(buf[offset:], uint32(len(bs)))
		offset += 4
		copy(buf[offset:], bs)

	default:
		logError("mqiPCF.go: Trying to serialise PCF parameter. Unknown PCF type %d\n", p.Type)
	}
	return buf
}

/*
ReadPCFHeader extracts the MQCFH from an MQ message
*/
func ReadPCFHeader(buf []byte) (*MQCFH, int) {

	fullLen := len(buf)

	if fullLen < C.MQCFH_STRUC_LENGTH {
		return nil, 0
	}

	cfh := new(MQCFH)
	p := bytes.NewBuffer(buf)

	binary.Read(p, endian, &cfh.Type)
	binary.Read(p, endian, &cfh.StrucLength)
	binary.Read(p, endian, &cfh.Version)
	binary.Read(p, endian, &cfh.Command)
	binary.Read(p, endian, &cfh.MsgSeqNumber)
	binary.Read(p, endian, &cfh.Control)
	binary.Read(p, endian, &cfh.CompCode)
	binary.Read(p, endian, &cfh.Reason)
	binary.Read(p, endian, &cfh.ParameterCount)

	bytesRead := fullLen - p.Len()
	return cfh, bytesRead
}

/*
ReadPCFEmbeddedHeader extracts the MQCFH from an MQ message
*/
func ReadPCFEmbeddedHeader(buf []byte) (*MQEPH, int) {
	var dummy int32

	fullLen := len(buf)
	if fullLen < C.MQEPH_STRUC_LENGTH_FIXED {
		return nil, 0
	}

	eph := new(MQEPH)
	p := bytes.NewBuffer(buf)

	offset := 0
	binary.Read(p, endian, &dummy)
	offset += 4
	binary.Read(p, endian, &eph.Version)
	offset += 4

	binary.Read(p, endian, &eph.StrucLength)
	offset += 4

	binary.Read(p, endian, &eph.Encoding)
	offset += 4

	binary.Read(p, endian, &eph.CodedCharSetId)
	offset += 4

	// We don't actually use this next field
	// s := string(buf[offset : offset+8])
	// s = trimToNull(s)
	offset += 8
	p.Next(8)

	binary.Read(p, endian, &eph.Flags)
	bytesRead := fullLen - p.Len()

	return eph, bytesRead
}

/*
ReadPCFParameter extracts the next PCF parameter element from an
MQ message.
*/
func ReadPCFParameter(buf []byte) (*PCFParameter, int) {
	var i32 int32
	var i64 int64
	var mqlong int32
	var count int32

	pcfParm := new(PCFParameter)
	fullLen := len(buf)
	p := bytes.NewBuffer(buf)

	binary.Read(p, endian, &pcfParm.Type)
	binary.Read(p, endian, &pcfParm.strucLength)

	switch pcfParm.Type {
	// There are more PCF element types but the monitoring packages only
	// needed a subset. We can add the others later if necessary.
	case C.MQCFT_INTEGER:
		binary.Read(p, endian, &pcfParm.Parameter)
		binary.Read(p, endian, &i32)
		pcfParm.Int64Value = append(pcfParm.Int64Value, int64(i32))

	case C.MQCFT_INTEGER_LIST:
		binary.Read(p, endian, &pcfParm.Parameter)
		binary.Read(p, endian, &count)
		// Bound the loop by the data actually present. A malformed count would
		// otherwise iterate billions of times and grow the slice without limit;
		// binary.Read fails once the buffer is exhausted, so stop rather than
		// trusting count.
		for i := 0; i < int(count); i++ {
			if err := binary.Read(p, endian, &i32); err != nil {
				break
			}
			pcfParm.Int64Value = append(pcfParm.Int64Value, int64(i32))
		}

	case C.MQCFT_INTEGER64:
		binary.Read(p, endian, &pcfParm.Parameter)
		binary.Read(p, endian, &mqlong) // Used for alignment
		binary.Read(p, endian, &i64)
		pcfParm.Int64Value = append(pcfParm.Int64Value, i64)

	case C.MQCFT_INTEGER64_LIST:
		binary.Read(p, endian, &pcfParm.Parameter)
		binary.Read(p, endian, &count)
		// Same count guard as MQCFT_INTEGER_LIST: stop when the buffer runs out
		// instead of trusting a possibly-malformed count.
		for i := 0; i < int(count); i++ {
			if err := binary.Read(p, endian, &i64); err != nil {
				break
			}
			pcfParm.Int64Value = append(pcfParm.Int64Value, i64)
		}

	case C.MQCFT_STRING:
		offset := int(C.MQCFST_STRUC_LENGTH_FIXED)
		binary.Read(p, endian, &pcfParm.Parameter)
		binary.Read(p, endian, &pcfParm.CodedCharSetId)
		binary.Read(p, endian, &pcfParm.stringLength)
		if raw, ok := pcfSlice(buf, offset, pcfParm.stringLength); ok {
			s := trimToNull(string(raw))
			pcfParm.String = append(pcfParm.String, s)
		}
		pcfSkip(p, int(pcfParm.strucLength)-offset)

	case C.MQCFT_STRING_LIST:
		binary.Read(p, endian, &pcfParm.Parameter)
		binary.Read(p, endian, &pcfParm.CodedCharSetId)
		binary.Read(p, endian, &count)
		binary.Read(p, endian, &pcfParm.stringLength)
		// Guard count as well as each element's length: a huge count would
		// otherwise loop for a long time or overflow the offset arithmetic.
		// stringLength must be strictly positive - a zero length never advances
		// the offset, so a huge count would still spin appending empty strings.
		if count > 0 && pcfParm.stringLength > 0 {
			offset := int(C.MQCFSL_STRUC_LENGTH_FIXED)
			for i := 0; i < int(count); i++ {
				off64 := int64(offset) + int64(i)*int64(pcfParm.stringLength)
				if off64 < 0 || off64 > int64(len(buf)) {
					break
				}
				raw, ok := pcfSlice(buf, int(off64), pcfParm.stringLength)
				if !ok {
					break
				}
				s := trimToNull(string(raw))
				pcfParm.String = append(pcfParm.String, s)
			}
		}
		pcfSkip(p, int(pcfParm.strucLength)-int(C.MQCFSL_STRUC_LENGTH_FIXED))

	case C.MQCFT_GROUP:
		// This reads the entire group, including the group elements.
		// Which might in turn be nested groups
		binary.Read(p, endian, &pcfParm.Parameter)
		binary.Read(p, endian, &pcfParm.ParameterCount)
		offset := 16 // Include the Type/StrucLength words in the already-read count
		bytesRead := 0
		// Reject absurd/negative ParameterCount so we do not allocate huge slices
		// or loop forever on malformed input.
		if pcfParm.ParameterCount < 0 || int(pcfParm.ParameterCount) > fullLen {
			return pcfParm, offset
		}
		pcfParm.GroupList = make([]*PCFParameter, pcfParm.ParameterCount)
		for i := 0; i < int(pcfParm.ParameterCount); i++ {
			if offset >= fullLen {
				break
			}
			pcfParm.GroupList[i], bytesRead = ReadPCFParameter(buf[offset:])
			if bytesRead <= 0 {
				break
			}
			offset += bytesRead
		}
		return pcfParm, offset

	case C.MQCFT_BYTE_STRING:
		// The byte string is converted to a hex string as that's how
		// we expect to use it in reporting
		offset := int(C.MQCFBS_STRUC_LENGTH_FIXED)
		binary.Read(p, endian, &pcfParm.Parameter)
		binary.Read(p, endian, &pcfParm.stringLength)
		if raw, ok := pcfSlice(buf, offset, pcfParm.stringLength); ok {
			s := hex.EncodeToString(raw)
			pcfParm.String = append(pcfParm.String, s)
		}
		pcfSkip(p, int(pcfParm.strucLength)-offset)

	case C.MQCFT_INTEGER_FILTER:
		binary.Read(p, endian, &pcfParm.Filter.Parameter)
		binary.Read(p, endian, &pcfParm.Filter.Operator)
		binary.Read(p, endian, &i32)
		pcfParm.Filter.FilterValue = int64(i32)

	case C.MQCFT_STRING_FILTER:
		offset := int(C.MQCFSF_STRUC_LENGTH_FIXED)

		binary.Read(p, endian, &pcfParm.Filter.Parameter)
		binary.Read(p, endian, &pcfParm.Filter.Operator)
		binary.Read(p, endian, &pcfParm.CodedCharSetId)
		binary.Read(p, endian, &pcfParm.stringLength)
		if raw, ok := pcfSlice(buf, offset, pcfParm.stringLength); ok {
			s := trimToNull(string(raw))
			pcfParm.Filter.FilterValue = s
		}
		pcfSkip(p, int(pcfParm.strucLength)-offset)

	case C.MQCFT_BYTE_STRING_FILTER:
		offset := int(C.MQCFBF_STRUC_LENGTH_FIXED)
		binary.Read(p, endian, &pcfParm.Filter.Parameter)
		binary.Read(p, endian, &pcfParm.Filter.Operator)
		binary.Read(p, endian, &pcfParm.stringLength)
		if raw, ok := pcfSlice(buf, offset, pcfParm.stringLength); ok {
			s := hex.EncodeToString(raw)
			pcfParm.Filter.FilterValue = s
		}
		pcfSkip(p, int(pcfParm.strucLength)-offset)

	default:
		// This should not happen, but if it does then dump various pieces of
		// debug information that might help solve the problem.
		// TODO: Put this in something like an environment variable control option
		localerr := fmt.Errorf("mqiPCF.go: Unknown PCF type %d", pcfParm.Type)
		fmt.Println("Error: ", localerr)
		fmt.Println("Buffer Len: ", len(buf))
		fmt.Println("Buffer: ", buf)

		debug.PrintStack()
		// After dumping the stack, we will try to carry on regardless.
		// Skip the remains of this structure, assuming it really is
		// PCF and we just don't know how to process the element type
		pcfSkip(p, int(pcfParm.strucLength)-8)
	}

	bytesRead := fullLen - p.Len()
	return pcfParm, bytesRead
}

// pcfSlice returns buf[offset:offset+length] when the range lies entirely
// inside buf. length comes from the wire as int32 and must be non-negative.
// Returns ok=false for negative length, overflow, or a range past the buffer
// end — callers must not panic on malformed PCF.
func pcfSlice(buf []byte, offset int, length int32) ([]byte, bool) {
	if length < 0 || offset < 0 {
		return nil, false
	}
	end := int64(offset) + int64(length)
	if end < int64(offset) || end > int64(len(buf)) {
		return nil, false
	}
	return buf[offset:int(end)], true
}

// pcfSkip advances p by n bytes, clamping to the remaining buffer and
// ignoring negative skips (malformed StrucLength).
func pcfSkip(p *bytes.Buffer, n int) {
	if n <= 0 {
		return
	}
	if n > p.Len() {
		n = p.Len()
	}
	p.Next(n)
}

func roundTo4(u int32) int32 {
	return ((u) + ((4 - ((u) % 4)) % 4))
}

func trimToNull(s string) string {
	var rc string
	i := strings.IndexByte(s, 0)
	if i == -1 {
		rc = s
	} else {
		rc = s[0:i]
	}
	return strings.TrimSpace(rc)
}
