/*
 * This is an example of a Go program to subscribe to publications from an IBM MQ
 * topic.
 *
 * The topic and queue manager name can be given as parameters on the
 * command line. Defaults are coded in the program.
 *
 * The program loops until no more publications arv available, waiting for
 * at most 3 seconds for new messages to arrive.
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

	"github.com/ibm-messaging/mq-golang/ibmmq"
)

var qMgrObject ibmmq.MQObject
var qObject ibmmq.MQObject
var subscriptionObject ibmmq.MQObject

func main() {

	// The default queue manager and topic to be used. These can be overridden on command line.
	qMgrName := "QM1"
	topic := "GO.TEST.TOPIC"

	qMgrConnected := false
	subscriptionMade := false

	fmt.Println("Sample AMQSSUB.GO start")

	// Get the queue and queue manager names from command line for overriding
	// the defaults. Parameters are not required.
	if len(os.Args) >= 2 {
		topic = os.Args[1]
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

	// Subscribe to the topic
	if err == nil {
		// Create the Object Descriptor that allows us to give the queue name
		mqsd := ibmmq.NewMQSD()

		// We have to say how we are going to use this subscription. The most important flags
		// here say that
		// a) the subscription is non-durable (it will be automatically removed at the end of the program)
		// b) the queue manager will automatically manage creation and deletion of the queue
		// where publications are delivered
		mqsd.Options = ibmmq.MQSO_CREATE |
			ibmmq.MQSO_NON_DURABLE |
			ibmmq.MQSO_FAIL_IF_QUIESCING |
			ibmmq.MQSO_MANAGED

		// When opening a Subscription, MQ has a choice of whether to refer to
		// the object through an ObjectName value or the ObjectString value or both.
		// For simplicity, here we work with just the ObjectString
		mqsd.ObjectString = topic

		// The qObject is filled in with a reference to the queue created automatically
		// for publications. It will be used in a moment for the Get operations
		subscriptionObject, err = qMgrObject.Sub(mqsd, &qObject)
		if err != nil {
			fmt.Println(err)
		} else {
			subscriptionMade = true
			fmt.Println("Subscription made to topic ", topic)
		}
	}

	msgAvail := true
	for msgAvail == true && err == nil {
		var datalen int

		// The GET requires control structures, the Message Descriptor (MQMD)
		// and Get Options (MQGMO). Create those with default values.
		getmqmd := ibmmq.NewMQMD()
		gmo := ibmmq.NewMQGMO()

		// The default options are OK, but it's always
		// a good idea to be explicit about transactional boundaries as
		// not all platforms behave the same way. It's also good practice to
		// set the FAIL_IF_QUIESCING flag on all verbs.
		gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT | ibmmq.MQGMO_FAIL_IF_QUIESCING

		// Set options to wait for a maximum of 3 seconds for any new message to arrive
		gmo.Options |= ibmmq.MQGMO_WAIT
		gmo.WaitInterval = 3 * 1000 // The WaitInterval is in milliseconds

		// Create a buffer for the message data. This one is large enough
		// for the messages put by the amqsput sample.
		buffer := make([]byte, 1024)

		// Now we can try to get the message
		datalen, err = qObject.Get(getmqmd, gmo, buffer)

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
			// if it's been created by the amqspub program
			fmt.Printf("Got message of length %d: ", datalen)
			fmt.Println(strings.TrimSpace(string(buffer[:datalen])))
		}
	}

	// The usual tidy up at the end of a program is for queues to be closed,
	// queue manager connections to be disconnected etc.
	// In a larger Go program, we might move this to a defer() section to ensure
	// it gets done regardless of other flows through the program.

	// Close the subscription if it was opened. This will also close the
	// managed publication queue.
	if subscriptionMade {
		err = subscriptionObject.Close(0)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Closed topic")
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
