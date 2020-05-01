/*
 * This is an example of a Go program to set some attributes of an IBM MQ
 * queue through the MQSET verb. The attributes that can be set are limited;
 * those limitations are described in the MQSET documentation.
 *
 * The queue and queue manager name can be given as parameters on the
 * command line. Defaults are coded in the program.
 *
 *
 */
package main

/*
  Copyright (c) IBM Corporation 2018

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
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

var qMgrObject ibmmq.MQObject
var object ibmmq.MQObject

/*
 * This is an example of how to call MQSET
 */
func setAttributes(obj ibmmq.MQObject) {
	// Create a map containing the selectors and their values. The values must be
	// a string (for MQCA attributes) or an int/int32 for the MQIA values.

	// The value being set in the TRIGDATA attribute has a timestamp so you can
	// see if it is successfully changed
	selectors := map[int32]interface{}{
		ibmmq.MQIA_INHIBIT_PUT:     ibmmq.MQQA_PUT_INHIBITED,
		ibmmq.MQIA_TRIGGER_CONTROL: ibmmq.MQTC_ON,
		ibmmq.MQCA_TRIGGER_DATA:    "Data set at " + time.Now().Format(time.RFC3339)}

	// And call the MQI
	err := obj.Set(selectors)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Object attributes successfully changed")
	}
}

// Main function that simply calls a subfunction to ensure defer routines are called before os.Exit happens
func main() {
	os.Exit(mainWithRc())
}

// The real main function is here to set a return code.
func mainWithRc() int {
	// The default queue manager and a queue to be used. These can be overridden on command line.
	qMgrName := "QM1"
	qName := "DEV.QUEUE.1"

	fmt.Println("Sample AMQSSET.GO start")

	// Get the object names from command line for overriding
	// the defaults. Parameters are not required.
	if len(os.Args) >= 2 {
		qName = os.Args[1]
	}

	if len(os.Args) >= 3 {
		qMgrName = os.Args[2]
	}

	qMgrObject, err := ibmmq.Conn(qMgrName)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("Connected to queue manager %s\n", qMgrName)
		defer disc(qMgrObject)
	}

	// Open a queue with the option to say it will be modified
	if err == nil {
		// Create the Object Descriptor that allows us to give the object name
		mqod := ibmmq.NewMQOD()

		// We have to say how we are going to use this object. The MQOO_SET flag
		// says that it will be used for an MQSET operation
		openOptions := ibmmq.MQOO_SET
		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = qName

		object, err = qMgrObject.Open(mqod, openOptions)
		if err != nil {
			fmt.Println(err)
		} else {
			setAttributes(object)
			object.Close(0)
		}
	}

	// Exit with any return code extracted from the failing MQI call.
	// Deferred disconnect will happen after the return
	mqret := 0
	if err != nil {
		mqret = int((err.(*ibmmq.MQReturn)).MQCC)
	}
	return mqret
}

// Disconnect from the queue manager
func disc(qMgrObject ibmmq.MQQueueManager) error {
	err := qMgrObject.Disc()
	if err == nil {
		fmt.Printf("Disconnected from queue manager %s\n", qMgrObject.Name)
	} else {
		fmt.Println(err)
	}
	return err
}
