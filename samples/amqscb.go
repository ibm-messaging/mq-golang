/*
 * This is an example of a Go program to get messages from an IBM MQ
 * queue. It uses the asynchronous callback operation instead of using
 * a synchronous MQGET.
 *
 * The queue and queue manager name can be given as parameters on the
 * command line. Defaults are coded in the program.
 *
 * Each MQI call prints its success or failure.
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
	"strings"
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

var qMgrObject ibmmq.MQObject
var qObject ibmmq.MQObject
var mh ibmmq.MQMessageHandle

var ok = true

// The main function just expects to be given a return code for Exit()
func main() {
	os.Exit(mainWithRc())
}

// This is the callback function invoked when a message arrives on the queue.
func cb(hConn *ibmmq.MQQueueManager, hObj *ibmmq.MQObject, md *ibmmq.MQMD, gmo *ibmmq.MQGMO, buffer []byte, cbc *ibmmq.MQCBC, err *ibmmq.MQReturn) {
	buflen := len(buffer)
	if err.MQCC != ibmmq.MQCC_OK {
		fmt.Println(err)
		ok = false
	} else {
		// Assume the message is a printable string, which it will be
		// if it's been created by the amqsput program
		fmt.Printf("In callback - Got message of length %d from queue %s: ", buflen, hObj.Name)
		fmt.Println(strings.TrimSpace(string(buffer[:buflen])))
	}
}

// The real main function is here to set a return code.
func mainWithRc() int {

	// The default queue manager and queue to be used. These can be overridden on command line.
	qMgrName := "QM1"
	qName := "DEV.QUEUE.1"

	fmt.Println("Sample AMQSCB.GO start")

	// Get the queue and queue manager names from command line for overriding
	// the defaults. Parameters are not required.
	if len(os.Args) >= 2 {
		qName = os.Args[1]
	}

	if len(os.Args) >= 3 {
		qMgrName = os.Args[2]
	}

	// Connect to the queue manager.
	qMgrObject, err := ibmmq.Conn(qMgrName)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("Connected to queue manager %s\n", qMgrName)
		defer disc(qMgrObject)
	}

	// Open the queue
	if err == nil {
		// Create the Object Descriptor that allows us to give the queue name
		mqod := ibmmq.NewMQOD()

		// We have to say how we are going to use this queue. In this case, to GET
		// messages. That is done in the openOptions parameter.
		openOptions := ibmmq.MQOO_INPUT_EXCLUSIVE

		// Opening a QUEUE (rather than a Topic or other object type) and give the name
		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = qName

		qObject, err = qMgrObject.Open(mqod, openOptions)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Opened queue", qObject.Name)
			defer close(qObject)
		}
	}

	if err == nil {
		cmho := ibmmq.NewMQCMHO()
		mh, err = qMgrObject.CrtMH(cmho)
	}

	if err == nil {
		// The GET/MQCB requires control structures, the Message Descriptor (MQMD)
		// and Get Options (MQGMO). Create those with default values.
		getmqmd := ibmmq.NewMQMD()
		gmo := ibmmq.NewMQGMO()

		// The default options are OK, but it's always
		// a good idea to be explicit about transactional boundaries as
		// not all platforms behave the same way.
		gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT

		// Set options to wait for a maximum of 3 seconds for any new message to arrive
		gmo.Options |= ibmmq.MQGMO_WAIT
		gmo.WaitInterval = 3 * 1000 // The WaitInterval is in milliseconds

		gmo.Options |= ibmmq.MQGMO_PROPERTIES_IN_HANDLE
		gmo.MsgHandle = mh

		// The MQCBD structure is used to specify the function to be invoked
		// when a message arrives on a queue
		cbd := ibmmq.NewMQCBD()
		cbd.CallbackFunction = cb // The function at the top of this file

		// Register the callback function along with any selection criteria from the
		// MQMD and MQGMO parameters
		err = qObject.CB(ibmmq.MQOP_REGISTER, cbd, getmqmd, gmo)
	}

	if err == nil {
		// Then we are ready to enable the callback function. Any messages
		// on the queue will be sent to the callback
		ctlo := ibmmq.NewMQCTLO() // Default parameters are OK
		err = qMgrObject.Ctl(ibmmq.MQOP_START, ctlo)
		if err == nil {
			// Use defer to disable the message consumer when we are ready to exit.
			// Otherwise the shutdown will give MQRC_HCONN_ASYNC_ACTIVE error
			defer qMgrObject.Ctl(ibmmq.MQOP_STOP, ctlo)
		}
	}

	// Keep the program running until the callback has indicated there are no
	// more messages.
	d, _ := time.ParseDuration("5s")
	for ok && err == nil {
		time.Sleep(d)
	}

	// Exit with any return code extracted from the failing MQI call.
	// Deferred disconnect/close will happen after the return
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

// Close the queue if it was opened
func close(object ibmmq.MQObject) error {
	err := object.Close(0)
	if err == nil {
		fmt.Println("Closed queue")
	} else {
		fmt.Println(err)
	}
	return err
}
