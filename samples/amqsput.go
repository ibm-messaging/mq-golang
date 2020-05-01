/*
 * This is an example of a Go program to put messages to an IBM MQ
 * queue.
 *
 * The queue and queue manager name can be given as parameters on the
 * command line. Defaults are coded in the program.
 *
 * A single message is put, containing a "hello" and timestamp.
 * Each MQI call prints its success or failure. The MsgId of the
 * put message is also printed so it can be used as an optional
 * selection criterion on the amqsget sample program.
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
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

var qMgrObject ibmmq.MQObject
var qObject ibmmq.MQObject

// Main function that simply calls a subfunction to ensure defer routines are called before os.Exit happens
func main() {
	os.Exit(mainWithRc())
}

// The real main function is here to set a return code.
func mainWithRc() int {

	// The default queue manager and queue to be used. These can be overridden on command line.
	qMgrName := "QM1"
	qName := "DEV.QUEUE.1"

	fmt.Println("Sample AMQSPUT.GO start")

	// Get the queue and queue manager names from command line for overriding
	// the defaults. Parameters are not required.
	if len(os.Args) >= 2 {
		qName = os.Args[1]
	}

	if len(os.Args) >= 3 {
		qMgrName = os.Args[2]
	}

	// This is where we connect to the queue manager. It is assumed
	// that the queue manager is either local, or you have set the
	// client connection information externally eg via a CCDT or the
	// MQSERVER environment variable
	qMgrObject, err := ibmmq.Conn(qMgrName)
	if err != nil {
		fmt.Println(err)
	} else {
		// Make sure we disconnect from the queue manager later
		fmt.Printf("Connected to queue manager %s\n", qMgrName)
		defer disc(qMgrObject)
	}

	// Open the queue
	if err == nil {
		// Create the Object Descriptor that allows us to give the queue name
		mqod := ibmmq.NewMQOD()

		// We have to say how we are going to use this queue. In this case, to PUT
		// messages. That is done in the openOptions parameter.
		openOptions := ibmmq.MQOO_OUTPUT

		// Opening a QUEUE (rather than a Topic or other object type) and give the name
		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = qName

		qObject, err = qMgrObject.Open(mqod, openOptions)
		if err != nil {
			fmt.Println(err)
		} else {
			// Make sure we close the queue once we're done with it
			fmt.Println("Opened queue", qObject.Name)
			defer close(qObject)
		}
	}

	// PUT a message to the queue
	if err == nil {
		// The PUT requires control structures, the Message Descriptor (MQMD)
		// and Put Options (MQPMO). Create those with default values.
		putmqmd := ibmmq.NewMQMD()
		pmo := ibmmq.NewMQPMO()

		// The default options are OK, but it's always
		// a good idea to be explicit about transactional boundaries as
		// not all platforms behave the same way.
		pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT

		// Tell MQ what the message body format is. In this case, a text string
		putmqmd.Format = ibmmq.MQFMT_STRING

		// And create the contents to include a timestamp just to prove when it was created
		msgData := "Hello from Go at " + time.Now().Format(time.RFC3339)

		// The message is always sent as bytes, so has to be converted before the PUT.
		buffer := []byte(msgData)

		// Now put the message to the queue
		err = qObject.Put(putmqmd, pmo, buffer)

		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Put message to", strings.TrimSpace(qObject.Name))
			// Print the MsgId so it can be used as a parameter to amqsget
			fmt.Println("MsgId:" + hex.EncodeToString(putmqmd.MsgId))
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
