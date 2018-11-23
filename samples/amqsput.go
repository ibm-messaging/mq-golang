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

	"github.com/ibm-messaging/mq-golang/ibmmq"
)

var qMgrObject ibmmq.MQObject
var qObject ibmmq.MQObject

func main() {

	// The default queue manager and queue to be used. These can be overridden on command line.
	qMgrName := "QM1"
	qName := "DEV.QUEUE.1"

	qMgrConnected := false
	qOpened := false

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
		qMgrConnected = true
		fmt.Printf("Connected to queue manager %s\n", qMgrName)
	}

	// Open of the queue
	if err == nil {
		// Create the Object Descriptor that allows us to give the queue name
		mqod := ibmmq.NewMQOD()

		// We have to say how we are going to use this queue. In this case, to PUT
		// messages. That is done in the openOptions parameter.
		openOptions := ibmmq.MQOO_OUTPUT + ibmmq.MQOO_FAIL_IF_QUIESCING

		// Opening a QUEUE (rather than a Topic or other object type) and give the name
		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = qName

		qObject, err = qMgrObject.Open(mqod, openOptions)
		if err != nil {
			fmt.Println(err)
		} else {
			qOpened = true
			fmt.Println("Opened queue", qObject.Name)
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
		// not all platforms behave the same way. It's also good practice to
		// set the FAIL_IF_QUIESCING flag on all verbs, even for short-running
		// operations like this PUT.
		pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT | ibmmq.MQPMO_FAIL_IF_QUIESCING

		// Tell MQ what the message body format is. In this case, a text string
		putmqmd.Format = "MQSTR"

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

	// The usual tidy up at the end of a program is for queues to be closed,
	// queue manager connections to be disconnected etc.
	// In a larger Go program, we might move this to a defer() section to ensure
	// it gets done regardless of other flows through the program.

	// Close the queue if it was opened
	if qOpened {
		err = qObject.Close(0)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Closed queue")
		}
	}

	// Disconnect from the queue manager
	if qMgrConnected {
		err = qMgrObject.Disc()
		fmt.Printf("Disconnected from queue manager %s\n", qMgrName)
	}

	// Exit with any return code extracted from the failing MQI call.
	if err == nil {
		os.Exit(0)
	} else {
		mqret := err.(*ibmmq.MQReturn)
		os.Exit((int)(mqret.MQCC))
	}
}
