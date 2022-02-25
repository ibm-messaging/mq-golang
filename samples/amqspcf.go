/*
 * This is an example of a Go program to create and parse PCF messages. It
 * issues the equivalent of "DISPLAY Q(x) ALL".
 *
 * The queue manager name and the queue to inquire on can be given as parameters on the
 * command line. Defaults are coded in the program. It also accesses
 * SYSTEM.ADMIN.COMMAND.QUEUE and SYSTEM.DEFAULT.MODEL.QUEUE.
 *
 * The program loops until either the command responses indicate completion, or no more replies are on
 * the queue, waiting for at most 3 seconds for new messages to arrive.
 *
 */
package main

/*
  Copyright (c) IBM Corporation 2022

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
var qCommandObject ibmmq.MQObject
var qReplyObject ibmmq.MQObject

const (
	qCommandName = "SYSTEM.ADMIN.COMMAND.QUEUE"
	qReplyName   = "SYSTEM.DEFAULT.MODEL.QUEUE"
	blank8       = "        "
	blank16      = blank8 + blank8
	blank32      = blank16 + blank16
	blank64      = blank32 + blank32
)

func main() {
	os.Exit(mainWithRc())
}

// The real main function is here to set a return code.
func mainWithRc() int {
	// The default queue manager and queue to be used. These can be overridden on command line.
	qMgrName := "QM1"
	qName := "DEV.QUEUE.1" // This is the queue on which we inquire; it is not actually opened

	fmt.Println("Sample AMQSPCF.GO start")

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

	// Open the command queue to send the PCF message
	if err == nil {
		// Create the Object Descriptor that allows us to give the queue name
		mqod := ibmmq.NewMQOD()

		// We have to say how we are going to use this queue. In this case, to GET
		// messages. That is done in the openOptions parameter.
		openOptions := ibmmq.MQOO_OUTPUT

		// Opening a QUEUE (rather than a Topic or other object type) and give the name
		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = qCommandName

		qCommandObject, err = qMgrObject.Open(mqod, openOptions)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Opened queue", qCommandObject.Name)
			defer close(qCommandObject)
		}
	}

	// Open the reply queue
	if err == nil {
		// Create the Object Descriptor that allows us to give the queue name
		mqod := ibmmq.NewMQOD()

		// We have to say how we are going to use this queue. In this case, to GET
		// messages. That is done in the openOptions parameter.
		openOptions := ibmmq.MQOO_INPUT_EXCLUSIVE

		// Opening a QUEUE (rather than a Topic or other object type) and give the name
		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = qReplyName

		qReplyObject, err = qMgrObject.Open(mqod, openOptions)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Opened queue", qReplyObject.Name)
			defer close(qReplyObject)
		}
	}

	// Now we can issue the PCF command to the queue manager and wait for responses
	if err == nil {
		err = putCommandMessage(qName)
	}
	if err == nil {
		getReplies()
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

func putCommandMessage(qName string) error {
	// Create the MQMD and MQPMO structures that will
	// be needed to put the message
	putmqmd := ibmmq.NewMQMD()
	pmo := ibmmq.NewMQPMO()

	pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT
	pmo.Options |= ibmmq.MQPMO_NEW_MSG_ID
	pmo.Options |= ibmmq.MQPMO_NEW_CORREL_ID
	pmo.Options |= ibmmq.MQPMO_FAIL_IF_QUIESCING

	// This is an ADMIN message, and replies need to be
	// sent back to the designated replyQ
	putmqmd.Format = "MQADMIN"
	putmqmd.ReplyToQ = qReplyObject.Name
	putmqmd.MsgType = ibmmq.MQMT_REQUEST
	putmqmd.Report = ibmmq.MQRO_PASS_DISCARD_AND_EXPIRY

	// Create a buffer where the PCF contents will be put
	buf := make([]byte, 0)

	// A PCF command consists of the CFH structure followed
	// by the actual parameters to the command
	cfh := ibmmq.NewMQCFH()
	cfh.Version = ibmmq.MQCFH_VERSION_3
	cfh.Type = ibmmq.MQCFT_COMMAND_XR
	cfh.Command = ibmmq.MQCMD_INQUIRE_Q

	// Add the parameters one at a time into a buffer.
	// The INQUIRE_Q command is quite simple as it only needs
	// the queue name. But the pattern is the same for each
	// parameter, where this block gets replicated and modified as needed.
	//
	// The values of the parameters are put into an array;
	// if starting again I might choose a different design for the PCFParameter
	// structure but this works.
	// The parameter Type will determine how the value is converted
	// The Bytes() function serialises the parameter based on the Type and adds
	// it into the buf variable.
	pcfparm := new(ibmmq.PCFParameter)
	pcfparm.Type = ibmmq.MQCFT_STRING
	pcfparm.Parameter = ibmmq.MQCA_Q_NAME
	pcfparm.String = []string{qName}
	cfh.ParameterCount++
	buf = append(buf, pcfparm.Bytes()...)

	// More parameters could be added here such as constraining it to
	// only local queues. Read the PCF documntation for the list of mandatory
	// and optional elements that apply to each command.

	// Once we know the total number of parameters, put the
	// CFH header on the front of the buffer.
	buf = append(cfh.Bytes(), buf...)

	// Now put the message
	err := qCommandObject.Put(putmqmd, pmo, buf)
	if err != nil {
		fmt.Printf("PutCommandMessage: error is %+v\n", err)
	} else {
		fmt.Printf("Put message to command queue\n")
	}

	return err
}

// Get the replies to the command, and print them in a readable format
func getReplies() error {
	var err error
	allDone := false

	// Loop through the retrieval until there is an error,
	// or no more messages, or the command response indicates there will be
	// no more replies.
	for allDone == false && err == nil {
		var datalen int

		getmqmd := ibmmq.NewMQMD()
		gmo := ibmmq.NewMQGMO()

		gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT

		// Set options to wait for a maximum of 3 seconds for any new message to arrive
		gmo.Options |= ibmmq.MQGMO_WAIT
		gmo.WaitInterval = 3 * 1000 // The WaitInterval is in milliseconds

		// Create a buffer for the message data. This one is large enough
		// for most PCF responses.
		buffer := make([]byte, 0, 10*1024)

		// Now we can try to get the message. This operation returns
		// a buffer that can be used directly.
		buffer, datalen, err = qReplyObject.GetSlice(getmqmd, gmo, buffer)
		if err != nil {
			allDone = true
			mqret := err.(*ibmmq.MQReturn)
			if mqret.MQRC == ibmmq.MQRC_NO_MSG_AVAILABLE {
				// If there's no message available, then we won't treat that as a real error as
				// it's an expected situation
				err = nil
			} else {
				fmt.Println(err)
			}
		} else {
			fmt.Printf("Got message of format %s and length %d: \n", getmqmd.Format, datalen)

			cfh, offset := ibmmq.ReadPCFHeader(buffer)
			//fmt.Printf("CFH is %+v\n",cfh)

			reason := cfh.Reason
			if cfh.Control == ibmmq.MQCFC_LAST {
				allDone = true
			}

			if reason != ibmmq.MQRC_NONE {
				fmt.Printf("Command returned error %s [%d]\n", ibmmq.MQItoString("MQRC", int(reason)), reason)
			}
			// Can now walk through the returned buffer, extracting one parameter at a time. The
			// bytesRead value returned from each iteration tells us where the next structure starts. Using a slice
			// starting at the new offset is an easy way to step through
			for offset < datalen {
				pcfParm, bytesRead := ibmmq.ReadPCFParameter(buffer[offset:])
				printPcfParm(pcfParm)
				offset += bytesRead
			}
		}
	}

	return err
}

// For some of the returned fields, print the name and value.
// We are not going to print all of the different types that might be returned
// but you can see the pattern. As an additional example, the QueueType field gets
// transformed into the string equivalent. The amqsevta.c sample program in the MQ product
// has much fuller examples of how to recognise and convert the different elements.
func printPcfParm(p *ibmmq.PCFParameter) {
	name := ""
	val := ""
	switch p.Type {
	case ibmmq.MQCFT_INTEGER:
		name = ibmmq.MQItoString("MQIA", int(p.Parameter))
		v := int(p.Int64Value[0])
		if p.Parameter == ibmmq.MQIA_Q_TYPE {
			val = ibmmq.MQItoString("MQQT", v)
		} else {
			val = fmt.Sprintf("%d", v)
		}
	case ibmmq.MQCFT_STRING:
		name = ibmmq.MQItoString("MQCA", int(p.Parameter))
		val = p.String[0]
	default:
		// Do nothing for this example even though other types might be returned
	}
	if name != "" {
		fmt.Printf("Name: %s Value: %v\n", (name + blank64)[0:33], val)
	}
}

// Close the queue if it was opened
func close(object ibmmq.MQObject) error {
	err := object.Close(0)
	if err == nil {
		fmt.Println("Closed queue " + object.Name)
	} else {
		fmt.Println(err)
	}
	return err
}
