/*
  Copyright (c) IBM Corporation 2026

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
*/

package ibmmq

/*
Regression tests for bounds checking in ReadPCFParameter.

Malformed PCF elements previously used wire length fields directly in Go
slice expressions (e.g. buf[offset:offset+stringLength]), which panics on
oversized or negative lengths and can crash monitoring processes that parse
untrusted reply/statistics messages.

These tests construct minimal on-wire elements (no full MQCFH) and assert
that ReadPCFParameter returns without panicking.
*/

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// Element type constants from cmqcfc.h (avoid relying on C names in assertions).
const (
	testMQCFT_INTEGER_LIST       = 5
	testMQCFT_STRING             = 4
	testMQCFT_STRING_LIST        = 6
	testMQCFT_BYTE_STRING        = 9
	testMQCFT_STRING_FILTER      = 14
	testMQCFT_BYTE_STRING_FILTER = 15
	testMQCFT_INTEGER64_LIST     = 25
)

func testBuildStringParm(stringLength int32, payload string) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, endian, int32(testMQCFT_STRING))
	body := []byte(payload)
	// StrucLength = fixed (20) + rounded payload; use honest length of what we write
	strucLen := int32(20 + len(body))
	_ = binary.Write(buf, endian, strucLen)
	_ = binary.Write(buf, endian, int32(1)) // Parameter
	_ = binary.Write(buf, endian, int32(0)) // CCSID
	_ = binary.Write(buf, endian, stringLength)
	buf.Write(body)
	return buf.Bytes()
}

func testBuildByteStringParm(stringLength int32, payload string) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, endian, int32(testMQCFT_BYTE_STRING))
	body := []byte(payload)
	strucLen := int32(16 + len(body))
	_ = binary.Write(buf, endian, strucLen)
	_ = binary.Write(buf, endian, int32(1)) // Parameter
	_ = binary.Write(buf, endian, stringLength)
	buf.Write(body)
	return buf.Bytes()
}

func testBuildStringListParm(count, stringLength int32, one string) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, endian, int32(testMQCFT_STRING_LIST))
	// fixed header 24 bytes then count*stringLength of data (we write one honest element)
	body := []byte(one)
	// pad body to stringLength if positive and small
	if stringLength > 0 && int(stringLength) >= len(body) && stringLength < 1024 {
		padded := make([]byte, int(stringLength))
		copy(padded, body)
		body = padded
	}
	strucLen := int32(24 + len(body))
	_ = binary.Write(buf, endian, strucLen)
	_ = binary.Write(buf, endian, int32(1)) // Parameter
	_ = binary.Write(buf, endian, int32(0)) // CCSID
	_ = binary.Write(buf, endian, count)
	_ = binary.Write(buf, endian, stringLength)
	buf.Write(body)
	return buf.Bytes()
}

func testBuildStringFilterParm(stringLength int32, payload string) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, endian, int32(testMQCFT_STRING_FILTER))
	body := []byte(payload)
	strucLen := int32(24 + len(body))
	_ = binary.Write(buf, endian, strucLen)
	_ = binary.Write(buf, endian, int32(1)) // Filter Parameter
	_ = binary.Write(buf, endian, int32(1)) // Operator
	_ = binary.Write(buf, endian, int32(0)) // CCSID
	_ = binary.Write(buf, endian, stringLength)
	buf.Write(body)
	return buf.Bytes()
}

func testBuildByteStringFilterParm(stringLength int32, payload string) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, endian, int32(testMQCFT_BYTE_STRING_FILTER))
	body := []byte(payload)
	strucLen := int32(20 + len(body))
	_ = binary.Write(buf, endian, strucLen)
	_ = binary.Write(buf, endian, int32(1)) // Filter Parameter
	_ = binary.Write(buf, endian, int32(1)) // Operator
	_ = binary.Write(buf, endian, stringLength)
	buf.Write(body)
	return buf.Bytes()
}

func testBuildIntegerListParm(count int32, values []int32) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, endian, int32(testMQCFT_INTEGER_LIST))
	strucLen := int32(16 + 4*len(values)) // fixed header 16 + 4 bytes per value
	_ = binary.Write(buf, endian, strucLen)
	_ = binary.Write(buf, endian, int32(1)) // Parameter
	_ = binary.Write(buf, endian, count)    // possibly-lying count
	for _, v := range values {
		_ = binary.Write(buf, endian, v)
	}
	return buf.Bytes()
}

func testBuildInteger64ListParm(count int32, values []int64) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, endian, int32(testMQCFT_INTEGER64_LIST))
	strucLen := int32(16 + 8*len(values)) // fixed header 16 + 8 bytes per value
	_ = binary.Write(buf, endian, strucLen)
	_ = binary.Write(buf, endian, int32(1)) // Parameter
	_ = binary.Write(buf, endian, count)    // possibly-lying count
	for _, v := range values {
		_ = binary.Write(buf, endian, v)
	}
	return buf.Bytes()
}

// mustNotPanic fails the test if ReadPCFParameter panics.
func mustNotPanic(t *testing.T, name string, buf []byte) (parm *PCFParameter, n int) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s: ReadPCFParameter panicked: %v", name, r)
		}
	}()
	return ReadPCFParameter(buf)
}

func TestReadPCFParameterValidString(t *testing.T) {
	buf := testBuildStringParm(2, "AB")
	parm, n := mustNotPanic(t, "valid-string", buf)
	if n <= 0 {
		t.Fatalf("expected positive bytesRead, got %d", n)
	}
	if len(parm.String) != 1 || parm.String[0] != "AB" {
		t.Fatalf("unexpected string value: %+v", parm.String)
	}
}

func TestReadPCFParameterValidByteString(t *testing.T) {
	buf := testBuildByteStringParm(2, "AB")
	parm, _ := mustNotPanic(t, "valid-bytestring", buf)
	// hex of "AB"
	if len(parm.String) != 1 || parm.String[0] != "4142" {
		t.Fatalf("unexpected byte string hex: %+v", parm.String)
	}
}

func TestReadPCFParameterMalformedLengthsNoPanic(t *testing.T) {
	cases := []struct {
		name string
		buf  []byte
	}{
		{"string/oversized", testBuildStringParm(1<<20, "AB")},
		{"string/maxint", testBuildStringParm(0x7FFFFFFF, "AB")},
		{"string/negative", testBuildStringParm(-1, "AB")},
		{"bytestring/oversized", testBuildByteStringParm(1<<20, "AB")},
		{"bytestring/maxint", testBuildByteStringParm(0x7FFFFFFF, "AB")},
		{"bytestring/negative", testBuildByteStringParm(-1, "AB")},
		{"stringlist/oversized", testBuildStringListParm(1, 1<<20, "AB")},
		{"stringlist/maxint", testBuildStringListParm(1, 0x7FFFFFFF, "AB")},
		{"stringlist/negative-len", testBuildStringListParm(1, -1, "AB")},
		{"stringfilter/maxint", testBuildStringFilterParm(0x7FFFFFFF, "AB")},
		{"stringfilter/negative", testBuildStringFilterParm(-1, "AB")},
		{"bytestringfilter/maxint", testBuildByteStringFilterParm(0x7FFFFFFF, "AB")},
		{"bytestringfilter/negative", testBuildByteStringFilterParm(-1, "AB")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _ = mustNotPanic(t, tc.name, tc.buf)
		})
	}
}

// A malformed count must not drive the list loops past the data actually
// present: the number of decoded values is bounded by the buffer, not by the
// wire count. Previously these loops trusted count and would spin billions of
// times, growing the slice until the process ran out of memory.
func TestReadPCFParameterIntegerListHugeCountBounded(t *testing.T) {
	// count claims 0x7FFFFFFF but only two int32 values follow.
	buf := testBuildIntegerListParm(0x7FFFFFFF, []int32{10, 20})
	parm, _ := mustNotPanic(t, "intlist/hugecount", buf)
	if len(parm.Int64Value) != 2 {
		t.Fatalf("expected loop bounded to 2 present values, got %d", len(parm.Int64Value))
	}
}

func TestReadPCFParameterInteger64ListHugeCountBounded(t *testing.T) {
	buf := testBuildInteger64ListParm(0x7FFFFFFF, []int64{10, 20})
	parm, _ := mustNotPanic(t, "int64list/hugecount", buf)
	if len(parm.Int64Value) != 2 {
		t.Fatalf("expected loop bounded to 2 present values, got %d", len(parm.Int64Value))
	}
}

// stringLength == 0 combined with a huge count previously spun forever
// appending empty strings (the offset never advances). The guard now requires
// a strictly positive length, so no strings are produced and the call returns.
func TestReadPCFParameterStringListZeroLenHugeCount(t *testing.T) {
	buf := testBuildStringListParm(0x7FFFFFFF, 0, "")
	parm, _ := mustNotPanic(t, "stringlist/zerolen-hugecount", buf)
	if len(parm.String) != 0 {
		t.Fatalf("expected no strings for zero-length elements, got %d", len(parm.String))
	}
}

func TestPCFSliceHelper(t *testing.T) {
	buf := []byte{1, 2, 3, 4, 5}
	if _, ok := pcfSlice(buf, 0, 5); !ok {
		t.Fatal("expected full slice ok")
	}
	if _, ok := pcfSlice(buf, 1, 5); ok {
		t.Fatal("expected overflow rejected")
	}
	if _, ok := pcfSlice(buf, 0, -1); ok {
		t.Fatal("expected negative rejected")
	}
	if _, ok := pcfSlice(buf, 0, 0x7FFFFFFF); ok {
		t.Fatal("expected huge length rejected")
	}
}
