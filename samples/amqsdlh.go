/*
 * This is an example of a Go program to put and get messages to an IBM MQ
 * queue while manipulating a Dead Letter Header
 *
 * The queue and queue manager name can be given as parameters on the
 * command line. Defaults are coded in the program.
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

func main() {
	os.Exit(mainWithRc())
}

func addDLH(md *ibmmq.MQMD, buf []byte) []byte {
	// Create a new Dead Letter Header. This function modifies
	// the original message descriptor to indicate there is a DLH
	dlh := ibmmq.NewMQDLH(md)

	// Fill in the reason this message needs to be put to a DLQ along with
	// any other relevant information.
	dlh.Reason = ibmmq.MQRC_NOT_AUTHORIZED
	dlh.DestQName = "DEST.QUEUE"
	dlh.DestQMgrName = "DEST.QMGR"
	// Set the current date/time in the header. The way Go does date formatting
	// is very odd.Force the hundredths as there doesn't seem to be a simple way
	// to extract it without a '.' in the format.
	dlh.PutTime = time.Now().Format("030405")
	dlh.PutDate = time.Now().Format("20060102")

	// Then return a modified buffer with the original message data
	// following the DLH
	return append(dlh.Bytes(), buf...)
}

// Extract the DLH from the body of the message, print it and then
// print the remaining body.
func printDLH(md *ibmmq.MQMD, buf []byte) {
	bodyStart := 0
	buflen := len(buf)

	// Look to see if there is indeed a DLH
	fmt.Printf("Format = '%s'\n", md.Format)
	if md.Format == ibmmq.MQFMT_DEAD_LETTER_HEADER {
		header, headerLen, err := ibmmq.GetHeader(md, buf)
		if err == nil {
			dlh, ok := header.(*ibmmq.MQDLH)
			if ok {
				bodyStart += headerLen
				fmt.Printf("DLH Structure = %v\n", dlh)
				fmt.Printf("Format of next element = '%s'\n", dlh.Format)
			}
		}
	}

	// The original message data starts further on in the slice
	fmt.Printf("Got message of total length %d: ", buflen)
	fmt.Println(strings.TrimSpace(string(buf[bodyStart:buflen])))
}

// The real main function is here to set a return code.
func mainWithRc() int {
	var putmqmd *ibmmq.MQMD

	// The default queue manager and queue to be used. These can be overridden on command line.
	qMgrName := "QM1"
	qName := "DEV.QUEUE.1"

	fmt.Println("Sample AMQSDLH.GO start")

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

		// We have to say how we are going to use this queue. In this case, to PUT and GET
		// messages. That is done in the openOptions parameter.
		openOptions := ibmmq.MQOO_OUTPUT | ibmmq.MQOO_INPUT_AS_Q_DEF

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

	// PUT the message to the queue
	if err == nil {
		putmqmd = ibmmq.NewMQMD()
		pmo := ibmmq.NewMQPMO()

		pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT

		// Create the contents to include a timestamp just to prove when it was created
		msgData := "Hello from Go at " + time.Now().Format(time.RFC3339)
		buffer := []byte(msgData)
		putmqmd.Format = ibmmq.MQFMT_STRING

		// Add a Dead Letter Header to the message.
		newBuffer := addDLH(putmqmd, buffer)

		// Put the message to the queue)
		err = qObject.Put(putmqmd, pmo, newBuffer)
		if err != nil {
			fmt.Println(err)
		}
	}

	// And now try to GET the message we just put
	if err == nil {
		getmqmd := ibmmq.NewMQMD()
		gmo := ibmmq.NewMQGMO()

		gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT

		// Set options to not wait - we know the message is there since we just put it
		gmo.Options |= ibmmq.MQGMO_NO_WAIT

		// Use the MsgId to retrieve the same message
		gmo.MatchOptions = ibmmq.MQMO_MATCH_MSG_ID
		getmqmd.MsgId = putmqmd.MsgId

		// Create a buffer for the message data. This one is large enough
		// for the messages put by the amqsput sample.
		buffer := make([]byte, 1024)

		// Now we can try to get the message.
		datalen := 0
		datalen, err = qObject.Get(getmqmd, gmo, buffer)

		if err != nil {
			fmt.Println(err)
		} else {
			// A message has been retrieved. Print the contents, and the DLH
			// if one exists
			printDLH(getmqmd, buffer[0:datalen])
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
