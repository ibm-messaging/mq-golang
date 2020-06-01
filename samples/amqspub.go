/*
 * This is an example of a Go program to publish messages to an IBM MQ
 * topic.
 *
 * The topic and queue manager name can be given as parameters on the
 * command line. Defaults are coded in the program.
 *
 * A single message is published, containing a "hello" and timestamp.
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
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

var qMgrObject ibmmq.MQObject
var topicObject ibmmq.MQObject

func main() {
	os.Exit(mainWithRc())
}

// The real main function is here to set a return code.
func mainWithRc() int {

	// The default queue manager and topic to be used. These can be overridden on command line.
	qMgrName := "QM1"
	topicString := "dev/GoTopic"

	fmt.Println("Sample AMQSPUB.GO start")

	// Get the topic and queue manager names from command line for overriding
	// the defaults. Parameters are not required.
	if len(os.Args) >= 2 {
		topicString = os.Args[1]
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

	// Open of the topic object
	if err == nil {
		// Create the Object Descriptor that allows us to give the topic name
		mqod := ibmmq.NewMQOD()

		// We have to say how we are going to use this object. In this case, to PUBLISH
		// messages. That is done in the openOptions parameter.
		openOptions := ibmmq.MQOO_OUTPUT

		// When opening a Topic, MQ has a choice of whether to refer to
		// the object through an ObjectName value or the ObjectString value or both.
		// For simplicity, here we work with just the ObjectString
		mqod.ObjectType = ibmmq.MQOT_TOPIC
		mqod.ObjectString = topicString

		topicObject, err = qMgrObject.Open(mqod, openOptions)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Opened topic ", topicString)
			defer close(topicObject)
		}
	}

	// PUBLISH a message to the queue
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
		err = topicObject.Put(putmqmd, pmo, buffer)

		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Published message to", topicString)
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

// Close the topic if it was opened
func close(object ibmmq.MQObject) error {
	err := object.Close(0)
	if err == nil {
		fmt.Println("Closed topic")
	} else {
		fmt.Println(err)
	}
	return err
}
