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
MQItoString returns a string representation of the MQI #define. Only a few of the
sets of constants are decoded here; see cmqstrc.h for a full set.
*/
func MQItoString(class string, value int32) string {
	s := ""
	v := C.MQLONG(value)
	switch class {
	case "BACF":
		s = C.GoString(C.MQBACF_STR(v))
	case "CACF":
		s = C.GoString(C.MQCACF_STR(v))
	case "CACH":
		s = C.GoString(C.MQCACH_STR(v))
	case "CAMO":
		s = C.GoString(C.MQCAMO_STR(v))
	case "CA":
		s = C.GoString(C.MQCA_STR(v))
	case "CC":
		s = C.GoString(C.MQCC_STR(v))
	case "CMD":
		s = C.GoString(C.MQCMD_STR(v))
	case "IACF":
		s = C.GoString(C.MQIACF_STR(v))
	case "IACH":
		s = C.GoString(C.MQIACH_STR(v))
	case "IAMO":
		s = C.GoString(C.MQIAMO_STR(v))
	case "IAMO64":
		s = C.GoString(C.MQIAMO64_STR(v))
	case "IA":
		s = C.GoString(C.MQIA_STR(v))
	case "RCCF":
		s = C.GoString(C.MQRCCF_STR(v))
	case "RC":
		s = C.GoString(C.MQRC_STR(v))
	}
	return s
}
