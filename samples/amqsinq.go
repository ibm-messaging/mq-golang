/*
 * This is an example of a Go program to inquire on some attributes of some IBM MQ
 * objects. The program looks at a nominated queue, and also at a NAMELIST
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

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

var qMgrObject ibmmq.MQObject
var object ibmmq.MQObject

/*
 * This is an example of how to call MQINQ with a "map" format for
 * responses
 */
func inquire(obj ibmmq.MQObject, selectors []int32) {
	// This is the function to do the actual inquiry
	values, err := obj.Inq(selectors)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("----------- %s ----------\n", obj.Name)
		for s, v := range values {
			// Having got the values, print the selector number and name, along with the value
			ss := ibmmq.MQItoString("CA", int(s))
			if ss == "" {
				ss = ibmmq.MQItoString("IA", int(s))
			}
			fmt.Printf("%-4d %-32.32s\t'%v'\n", s, ss, v)
		}
	}
}

// Main function that simply calls a subfunction to ensure defer routines are called before os.Exit happens
func main() {
	os.Exit(mainWithRc())
}

// The real main function is here to set a return code.
func mainWithRc() int {
	var selectors []int32
	// The default queue manager, a queue and a namelist to be used. These can be overridden on command line.
	qMgrName := "QM1"
	qName := "DEV.QUEUE.1"
	nlName := "SYSTEM.DEFAULT.NAMELIST"

	fmt.Println("Sample AMQSINQ.GO start")

	// Get the object names from command line for overriding
	// the defaults. Parameters are not required.
	// Order is important here. Should be
	//    amqsinq qName qmgrName namelistName
	if len(os.Args) >= 2 {
		qName = os.Args[1]
	}

	if len(os.Args) >= 3 {
		qMgrName = os.Args[2]
	}

	if len(os.Args) >= 4 {
		nlName = os.Args[3]
	}

	qMgrObject, err := ibmmq.Conn(qMgrName)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("Connected to queue manager %s\n", qMgrName)
		defer disc(qMgrObject)
	}

	// Open an object
	if err == nil {
		// Create the Object Descriptor that allows us to give the object name
		mqod := ibmmq.NewMQOD()

		// We have to say how we are going to use this object.
		openOptions := ibmmq.MQOO_INQUIRE
		mqod.ObjectType = ibmmq.MQOT_Q_MGR
		// Do not need the qmgr name when opening it

		object, err = qMgrObject.Open(mqod, openOptions)
		if err != nil {
			fmt.Println(err)
		} else {
			selectors = []int32{
				ibmmq.MQCA_Q_MGR_NAME,
				ibmmq.MQCA_DEAD_LETTER_Q_NAME,
				ibmmq.MQIA_COMMAND_LEVEL}
			inquire(object, selectors)
			object.Close(0)
		}
	}

	// Open of another object - a queue
	if err == nil {
		// Create the Object Descriptor that allows us to give the object name
		mqod := ibmmq.NewMQOD()

		// We have to say how we are going to use this object.
		openOptions := ibmmq.MQOO_INQUIRE
		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = qName

		object, err = qMgrObject.Open(mqod, openOptions)
		if err != nil {
			fmt.Println(err)
		} else {
			selectors = []int32{
				ibmmq.MQIA_DEF_PERSISTENCE,
				ibmmq.MQCA_Q_NAME,
				ibmmq.MQCA_ALTERATION_DATE,
				ibmmq.MQIA_MAX_Q_DEPTH}
			inquire(object, selectors)
			object.Close(0)
		}
	}

	// Open of another object - a namelist. This has the ability to extract
	// the list of queues it refers to.
	if err == nil {
		// Create the Object Descriptor that allows us to give the object name
		mqod := ibmmq.NewMQOD()

		// We have to say how we are going to use this object.
		openOptions := ibmmq.MQOO_INQUIRE
		mqod.ObjectType = ibmmq.MQOT_NAMELIST
		mqod.ObjectName = nlName

		object, err = qMgrObject.Open(mqod, openOptions)
		if err != nil {
			fmt.Println(err)
		} else {
			selectors = []int32{ibmmq.MQCA_NAMELIST_NAME,
				ibmmq.MQIA_NAME_COUNT,
				ibmmq.MQCA_NAMES}

			inquire(object, selectors)
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
