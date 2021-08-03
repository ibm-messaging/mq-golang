/*
 * This is an example of a Go program that deals with unprocessable or "poison" messages on an IBM MQ
 * queue by moving them to a backout queue. As the message is moved to the backout queue
 * a dead-letter header is attached both to indicate the reason for the move, and to
 * permit a DLQ handler program to further process the message.
 *
 * The queue and queue manager name can be given as parameters on the
 * command line. Defaults are coded in the program.
 *
 * The input queue should be defined with both BOTHRESH and BOQNAME values and
 * the backout queue must exist
 *   DEF QL(DEV.QUEUE.1) BOTHRESH(3) BOQNAME(DEV.QUEUE.BACKOUT) REPLACE
 *   DEF QL(DEV.QUEUE.BACKOUT)
 *
 * Each MQI call prints its success or failure.
 *
 */
package main

/*
  Copyright (c) IBM Corporation 2018, 2021

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

func main() {
	os.Exit(mainWithRc())
}

// This function puts the message to the nominated backout queue. Adding a Dead Letter Header to the message is
// usually done, although some application designs choose not to do it.
func moveMsg(qMgrObject ibmmq.MQQueueManager, qName string, boQName string, md *ibmmq.MQMD, buffer []byte, reason int32) error {

	// Construct the DLH based on the original message descriptor. This also modifies
	// the message descriptor.
	dlh := ibmmq.NewMQDLH(md)

	// Fill in the reason this message needs to be put to a DLQ along with
	// any other relevant information.
	dlh.Reason = reason
	dlh.DestQName = qName
	dlh.DestQMgrName = qMgrObject.Name

	mqod := ibmmq.NewMQOD()
	mqod.ObjectType = ibmmq.MQOT_Q
	mqod.ObjectName = boQName

	pmo := ibmmq.NewMQPMO()
	pmo.Options = ibmmq.MQGMO_SYNCPOINT

	// The message is put directly to the backout queue. Since we don't
	// expect to use this queue frequently, using Put1 is a better idea
	// than separately opening the queue and using Put.
	fmt.Printf("About to move poison message to %s queue\n", boQName)
	return qMgrObject.Put1(mqod, md, pmo, append(dlh.Bytes(), buffer...))
}

// The real main function is here to set a return code.
func mainWithRc() int {
	var values map[int32]interface{}

	// These are the attributes that control how a poison message is handled
	boQName := ""
	boThreshold := int32(0)

	// The default queue manager and queue to be used. These can be overridden on command line.
	qMgrName := "QM1"
	qName := "DEV.QUEUE.1"

	fmt.Println("Sample AMQSBO.GO start")

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

	// Open of the application queue
	if err == nil {
		// Create the Object Descriptor that allows us to give the queue name
		mqod := ibmmq.NewMQOD()

		// We have to say how we are going to use this queue. In this case, to GET
		// messages and to look for the backout configuration options associated
		// with the queue. That is done in the openOptions parameter.
		openOptions := ibmmq.MQOO_INPUT_EXCLUSIVE | ibmmq.MQOO_INQUIRE

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

		// The backout configuration options for the queue are discoverable by the Inq() verb
		selectors := []int32{
			ibmmq.MQCA_BACKOUT_REQ_Q_NAME,
			ibmmq.MQIA_BACKOUT_THRESHOLD,
		}

		values, err = qObject.Inq(selectors)
		if err != nil {
			fmt.Println(err)
		} else {
			// The returned values are extracted and converted to usable
			// datatypes. See amqsinq.go for more information on this verb
			boQName = (values[ibmmq.MQCA_BACKOUT_REQ_Q_NAME]).(string)
			boThreshold = (values[ibmmq.MQIA_BACKOUT_THRESHOLD]).(int32)
			fmt.Printf("Backout QName=%s Threshold=%d\n", boQName, boThreshold)

			// If the queue doesn't have suitable configuration, then we can't continue
			if boQName == "" || boThreshold == 0 {
				err = fmt.Errorf("Backout parameters not correctly set")
				fmt.Println(err)
			}
		}
	}

	msgAvail := true
	for msgAvail == true && err == nil {
		var datalen int

		// The GET requires control structures, the Message Descriptor (MQMD)
		// and Get Options (MQGMO). Create those with default values.
		mqmd := ibmmq.NewMQMD()
		gmo := ibmmq.NewMQGMO()

		// Use syncpoint control so that commit/backout works as expected
		gmo.Options = ibmmq.MQGMO_SYNCPOINT
		gmo.Options |= ibmmq.MQGMO_NO_WAIT

		// Create a buffer for the message data. This one is large enough
		// for the messages put by the amqsput sample. Note that in this case
		// the make() operation is just allocating space - len(buffer)==0 initially.
		buffer := make([]byte, 0, 1024)

		// Now we can try to get the message. This operation returns
		// a buffer that can be used directly.
		buffer, datalen, err = qObject.GetSlice(mqmd, gmo, buffer)

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
			fmt.Printf("Got message of length %d. Backout Count=%d \t%s\n", datalen, mqmd.BackoutCount, strings.TrimSpace(string(buffer)))

			// If we have reached the backout threshold then move the message to the backout queue
			if mqmd.BackoutCount >= boThreshold {
				// Pick an reason for the failure - this can be a user-chosen number
				reason := ibmmq.MQRC_UNEXPECTED_ERROR
				err = moveMsg(qMgrObject, qName, boQName, mqmd, buffer, reason)
				if err != nil {
					fmt.Println(err)
				}
				// For this program, we'll commit even if there is an error putting to the backout queue so we don't
				// get into an infinite loop. But there may be more advanced strategies depending on the error code. For
				// example, you might count the number of failures here and delay the retries before really giving up.
				qMgrObject.Cmit()
			} else {
				// In real life, there would be some processing of the message here before deciding to backout or
				// commit the transaction. But here we will always do the backout.
				qMgrObject.Back()

				// Adding an increasing delay in here may help with some error conditions so you don't just spin quickly.
				// For example if the reason for the failure in processing the message is due to a temporary unavailability
				// of another component such as a database.
				time.Sleep(1 * time.Second)
			}
		}
	}

	// Exit with any return code extracted from the failing MQI call.
	// Deferred disconnect will happen after the return
	if err != nil {
		mqret, ok := err.(*ibmmq.MQReturn)
		if ok {
			return int(mqret.MQCC)
		}
	}
	return 0
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
