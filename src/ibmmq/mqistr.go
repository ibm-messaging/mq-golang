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
#include <cmqstrc.h>
*/
import "C"

import (
	"fmt"
)

/*
   Convert MQCC/MQRC values into readable text using
   the functions introduced in cmqstrc.h in MQ V8004
*/
func mqstrerror(verb string, mqcc C.MQLONG, mqrc C.MQLONG) error {
	return fmt.Errorf("%s: MQCC = %s [%d] MQRC = %s [%d]", verb,
		C.GoString(C.MQCC_STR(mqcc)), mqcc,
		C.GoString(C.MQRC_STR(mqrc)), mqrc)
}

/*
   These are wrappers around some of the MQI string mapping functions.
   Seems that we can't refer to them directly in multiple source
   files without getting "duplicate symbol" errors; nor does there seem
   to be a way to refer to them explicitly from other packages. Hence
   the need for these wrappers.
*/
func MQBACF_STR(v int32) string {
	return C.GoString(C.MQBACF_STR(C.MQLONG(v)))
}

func MQCACF_STR(v int32) string {
	return C.GoString(C.MQCACF_STR(C.MQLONG(v)))
}

func MQCACH_STR(v int32) string {
	return C.GoString(C.MQCACH_STR(C.MQLONG(v)))
}

func MQCAMO_STR(v int32) string {
	return C.GoString(C.MQCAMO_STR(C.MQLONG(v)))
}

func MQCA_STR(v int32) string {
	return C.GoString(C.MQCA_STR(C.MQLONG(v)))
}

func MQCC_STR(v int32) string {
	return C.GoString(C.MQCC_STR(C.MQLONG(v)))
}

func MQCMD_STR(v int32) string {
	return C.GoString(C.MQCMD_STR(C.MQLONG(v)))
}

func MQIACF_STR(v int32) string {
	return C.GoString(C.MQIACF_STR(C.MQLONG(v)))
}

func MQIACH_STR(v int32) string {
	return C.GoString(C.MQIACH_STR(C.MQLONG(v)))
}

func MQIAMO64_STR(v int32) string {
	return C.GoString(C.MQIAMO64_STR(C.MQLONG(v)))
}

func MQIAMO_STR(v int32) string {
	return C.GoString(C.MQIAMO_STR(C.MQLONG(v)))
}

func MQIA_STR(v int32) string {
	return C.GoString(C.MQIA_STR(C.MQLONG(v)))
}

func MQRCCF_STR(v int32) string {
	return C.GoString(C.MQRCCF_STR(C.MQLONG(v)))
}

func MQRC_STR(v int32) string {
	return C.GoString(C.MQRC_STR(C.MQLONG(v)))
}
