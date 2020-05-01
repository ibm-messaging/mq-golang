/*
 * This is an example of a Go program to browse messages from an IBM MQ
 * queue.
 *
 * The queue and queue manager name can be given as parameters on the
 * command line. Defaults are coded in the program.
 *
 * The program loops until no more messages are on the queue, waiting for
 * at most 3 seconds for new messages to arrive.
 *
 * Each MQI call prints its success or failure.
 *
 */
package main

/*
  Copyright (c) IBM Corporation 2018, 2019

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

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

var qMgrObject ibmmq.MQObject
var qObject ibmmq.MQObject

func main() {
	os.Exit(mainWithRc())
}

// The real main function is here to set a return code.
func mainWithRc() int {

	// The default queue manager and queue to be used. These can be overridden on command line.
	qMgrName := "QM1"
	qName := "DEV.QUEUE.1"

	fmt.Println("Sample AMQSGBR.GO start")

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
		fmt.Printf("Connected to queue manager %s\n", qMgrName)
		defer disc(qMgrObject)
	}

	// Open of the queue
	if err == nil {
		// Create the Object Descriptor that allows us to give the queue name
		mqod := ibmmq.NewMQOD()

		// We have to say how we are going to use this queue. In this case, to BROWSE
		// messages. That is done in the openOptions parameter.
		openOptions := ibmmq.MQOO_BROWSE

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

	msgAvail := true
	browseOption := ibmmq.MQGMO_BROWSE_FIRST
	for msgAvail == true && err == nil {
		var datalen int

		// The GET requires control structures, the Message Descriptor (MQMD)
		// and Get Options (MQGMO). Create those with default values.
		getmqmd := ibmmq.NewMQMD()
		gmo := ibmmq.NewMQGMO()

		// The default options are OK, but it's always
		// a good idea to be explicit about transactional boundaries as
		// not all platforms behave the same way.
		gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT | browseOption

		// Set options to wait for a maximum of 3 seconds for any new message to arrive
		gmo.Options |= ibmmq.MQGMO_WAIT
		gmo.WaitInterval = 3 * 1000 // The WaitInterval is in milliseconds

		// Create a buffer for the message data. This one is large enough
		// for the messages put by the amqsput sample. Note that in this case
		// the make() operation is just allocating space - len(buffer)==0 initially.
		buffer := make([]byte, 0, 1024)

		// Now we can try to get the message. This operation returns
		// a buffer that can be used directly.
		buffer, datalen, err = qObject.GetSlice(getmqmd, gmo, buffer)

		if err != nil {
			msgAvail = false
			fmt.Println(err)
			mqret := err.(*ibmmq.MQReturn)
			if mqret.MQRC == ibmmq.MQRC_NO_MSG_AVAILABLE {
				// If there's no message available, then I won't treat that as a real error as
				// it's an expected situation
				err = nil
			}
		} else {
			// Assume the message is a printable string, which it will be
			// if it's been created by the amqsput program
			fmt.Printf("Got message of length %d: ", datalen)
			fmt.Println(strings.TrimSpace(string(buffer)))
		}
		browseOption = ibmmq.MQGMO_BROWSE_NEXT
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
