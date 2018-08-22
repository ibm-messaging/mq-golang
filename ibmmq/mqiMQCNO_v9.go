// +build !MQv8

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

#include <stdlib.h>
#include <string.h>
#include <cmqc.h>
#include <cmqxc.h>

*/
import "C"
import "unsafe"

func init() {
	copyCCDTUrlToC = func(mqcno *C.MQCNO, gocno *MQCNO) {
		mqcno.CCDTUrlOffset = 0
		if len(gocno.CCDTUrl) != 0 {
			mqcno.CCDTUrlPtr = C.PMQCHAR(unsafe.Pointer(C.CString(gocno.CCDTUrl)))
			mqcno.CCDTUrlLength = C.MQLONG(len(gocno.CCDTUrl))
		} else {
			mqcno.CCDTUrlPtr = nil
			mqcno.CCDTUrlLength = 0
		}
	}

	copyCCDTUrlFromC = func(gocno *MQCNO, mqcno *C.MQCNO) {
		if mqcno.CCDTUrlPtr != nil {
			C.free(unsafe.Pointer(mqcno.CCDTUrlPtr))
		}
	}
}
