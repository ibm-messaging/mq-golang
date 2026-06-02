/*
 * This is an example of a Go program to get messages from an IBM MQ
 * queue in response to being triggered. The qmgr and queue names
 * are given via the MQTMC2 string/structure from the command line. There are
 * no default values.
 *
 * The program loops until no more messages are on the queue, waiting for
 * at most 1 second for new messages to arrive. Triggered apps should always
 * loop and wait a little while, to minimise the number of trigger messages
 * created and processed.
 *
 * Sample MQRC to configure triggering:
 *   DEF QL(DEV.ECHO) INITQ(DEV.INITQ) PROCESS(DEV.PROC) trigger trigtype(first) trigdata('sometriggerdata') replace
 *   DEF QL(DEV.INITQ) replace
 *   DEF PROCESS(DEV.PROC) applicid('$curdir/amqsech') userdata('someuserdata') envrdata('someenvdata') replace
 *
 * Then run "amqstrg DEV.INITQ QM1" and put messages to the DEV.ECHO queue
 * This program should be triggered and print the message contents.
 */

package main

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
  limitations under the license.

   Contributors:
     Mark Taylor - Initial Contribution
*/

import (
	"fmt"
	"os"
	"strings"

	mq "github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

var qMgrObject mq.MQObject
var qObject mq.MQObject

func main() {
	os.Exit(mainWithRc())
}

// The real main function is here to set a return code.
func mainWithRc() int {

	qMgrName := ""
	qName := ""

	fmt.Println("Sample AMQSECH.GO start")

	// Get the queue and queue manager names from the MQTMC2 structure's contents.
	if len(os.Args) >= 2 {
		s := os.Args[1]

		// fmt.Printf("Got parameter provided of len %d: %s\n", len(s), s)
		tmc, err := mq.NewMQTMC2(s)
		if err != nil {
			fmt.Printf("Bad TMC2 parameter provided: %s\n", s)
			return 1
		}

		// Print the MQTMC structure so we can see what's been passed
		// fmt.Printf("TMC2: %+v\n", tmc)

		qMgrName = tmc.QMgrName
		qName = tmc.QName
	} else {
		fmt.Printf("No TMC2 parameter provided\n")
		return 1
	}

	qMgrObject, err := mq.Conn(qMgrName)
	if err != nil {
		fmt.Println(err)
		return 1
	} else {
		defer disc(qMgrObject)
	}

	// Open of the queue
	if err == nil {
		mqod := mq.NewMQOD()
		openOptions := mq.MQOO_INPUT_EXCLUSIVE

		mqod.ObjectType = mq.MQOT_Q
		mqod.ObjectName = qName

		qObject, err = qMgrObject.Open(mqod, openOptions)
		if err != nil {
			fmt.Println(err)
			return 1
		} else {
			defer close(qObject)
		}
	}

	msgAvail := true
	for msgAvail == true && err == nil {
		var datalen int

		getmqmd := mq.NewMQMD()
		getmqmd.Version = 2
		gmo := mq.NewMQGMO()

		gmo.Options = mq.MQGMO_NO_SYNCPOINT

		// Set options to wait for a maximum of 1 second for any new message to arrive
		gmo.Options |= mq.MQGMO_WAIT
		gmo.Options |= mq.MQGMO_ACCEPT_TRUNCATED_MSG
		gmo.WaitInterval = 1 * 1000 // The WaitInterval is in milliseconds

		// Create a buffer for the message data. This one is large enough
		// for the messages put by the amqsput sample.
		buffer := make([]byte, 0, 1024)

		// Now we can try to get the message. This operation returnsibmmq.
		// a buffer that can be used directly.
		buffer, datalen, err = qObject.GetSlice(getmqmd, gmo, buffer)

		if err != nil {
			msgAvail = false
			mqret := err.(*mq.MQReturn)
			if mqret.MQRC == mq.MQRC_NO_MSG_AVAILABLE {
				// If there's no message available, then I won't treat that as a real error as
				// it's an expected situation
				err = nil
			} else if mqret.MQRC == mq.MQRC_TRUNCATED_MSG_ACCEPTED {
				fmt.Printf("Got message of length %d: \n", datalen)
				fmt.Printf("Bufflen(Slice)        %d: \n", len(buffer))
				fmt.Println(strings.TrimSpace(string(buffer)))
				msgAvail = true
				err = nil
			} else {
				fmt.Println(err)
			}
		} else {
			// Assume the message is a printable string, which it will be
			// if it's been created by the amqsput program
			fmt.Printf("ECH: Got message of length %d: ", datalen)
			fmt.Println(strings.TrimSpace(string(buffer)))
		}
	}

	// Exit with any return code extracted from the failing MQI call.
	// Deferred disconnect will happen after the return
	mqret := 0
	if err != nil {
		mqret = int((err.(*mq.MQReturn)).MQCC)
	}
	return mqret
}

// Disconnect from the queue manager
func disc(qMgrObject mq.MQQueueManager) error {
	err := qMgrObject.Disc()
	if err != nil {
		fmt.Println(err)
	}
	return err
}

// Close the queue if it was opened
func close(object mq.MQObject) error {
	err := object.Close(0)
	if err != nil {
		fmt.Println(err)
	}
	return err
}
