/*
 * This is an example of a Go program that can run as a simple trigger monitor. While
 * the product-supplied runmqtrm or the amqstrg sample will work for most cases, there may
 * be environments where the "system()" equivalent is insufficient. This sample can
 * be modified to meet any extra requirements if you prefer working in Go.
 *
 * Other things that you might want to think about, and which runmqtrm does:
 * - move malformed trigger messages to a DLQ
 * - better error reporting/debug output
 * - any escape-characters or special formatting needed for parameters eg anything including quote chars
 * - a way to exit the program cleanly
 * - use the EnvData to set the application environment
 *
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
	"os/exec"

	mq "github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

var qMgrObject mq.MQObject
var qObject mq.MQObject

func main() {
	os.Exit(mainWithRc())
}

// The real main function is here to set a return code.
func mainWithRc() int {

	// The default queue manager and queue to be used. These can be overridden on command line.
	qMgrName := "QM1"
	qName := "DEV.INITQ"

	fmt.Println("Sample AMQSTRG.GO start")

	// Get the queue and queue manager names from command line for overriding
	// the defaults. Parameters are not required.
	if len(os.Args) >= 2 {
		qName = os.Args[1]
	}

	if len(os.Args) >= 3 {
		qMgrName = os.Args[2]
	}

	qMgrObject, err := mq.Conn(qMgrName)
	if err != nil {
		fmt.Println(err)
	} else {
		defer disc(qMgrObject)
	}

	// Open of the queue
	if err == nil {
		mqod := mq.NewMQOD()
		openOptions := mq.MQOO_INPUT_AS_Q_DEF
		mqod.ObjectType = mq.MQOT_Q
		mqod.ObjectName = qName

		qObject, err = qMgrObject.Open(mqod, openOptions)
		if err != nil {
			fmt.Println(err)
		} else {
			defer close(qObject)
		}
	}

	msgAvail := true
	for msgAvail == true && err == nil {

		// The GET requires control structures, the Message Descriptor (MQMD)
		// and Get Options (MQGMO). Create those with default values.
		getmqmd := mq.NewMQMD()
		getmqmd.Version = 2
		gmo := mq.NewMQGMO()

		gmo.Options = mq.MQGMO_NO_SYNCPOINT

		// Set options to wait for new messages to arrive
		gmo.Options |= mq.MQGMO_WAIT

		// A monitor should set this to "unlimited" so it runs forever (or until there's an error)
		gmo.WaitInterval = mq.MQWI_UNLIMITED

		// Create a buffer for the Trigger data. Must be at least MQTM_CURRENT_LENGTH.
		buffer := make([]byte, 0, 1024)

		// Now we can try to get the message. This operation returns
		// a buffer that can be used directly.
		buffer, _, err = qObject.GetSlice(getmqmd, gmo, buffer)

		if err != nil {
			// With an unlimited wait, we won't expect to see MQRC_NO_MSG_AVAILABLE
			msgAvail = false
			fmt.Println(err)
		} else if getmqmd.Format == mq.MQFMT_TRIGGER {
			// We've got a message, that should be an MQTM structure.

			// fmt.Printf("TRG Got message of length %d & format %s\n", datalen, getmqmd.Format)

			tm, err := mq.NewMQTM(buffer)
			if err == nil {
				app := tm.ApplId
				tmc := tm.ToTMC2(qMgrName)

				// Current versions of Go will not look in the current directory, regardless of PATH,
				// for application names that do not have any directory qualification. Use "./" in the ApplicId
				// or a full path to make sure the program can be found.
				cmd := exec.Command(app, tmc)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err := cmd.Run() // This waits for the started program to finish
				if err != nil {
					fmt.Printf("Error executing %s: %v\n", app, err)
				}
			} else {
				fmt.Printf("Error creating TM: %v\n", err)
			}

		} else {
			fmt.Printf("Unexpected message format: %s\n", getmqmd.Format)
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
