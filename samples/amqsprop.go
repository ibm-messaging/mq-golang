/*
 * This is an example of a Go program to put and get messages to an IBM MQ
 * queue while manipulating the message properties
 *
 * While the main body of this sample is the same as the amqsput and amqsget
 * samples, the important functions to understand here are setProperties and
 * printProperties.
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
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

var qMgrObject ibmmq.MQObject
var qObject ibmmq.MQObject
var putMsgHandle ibmmq.MQMessageHandle
var getMsgHandle ibmmq.MQMessageHandle

/* Set various properties of different types on the message
 */
func setProperties(putMsgHandle ibmmq.MQMessageHandle) error {
	var err error

	// Create the descriptor structures needed to set a property. In most cases,
	// the default values for these descriptors are sufficient.
	smpo := ibmmq.NewMQSMPO()
	pd := ibmmq.NewMQPD()

	// And now set several properties of different types

	// Note how the "value" of each property can change datatype
	// without needing to be explicitly stated.
	name := "PROP1STRING"
	v1 := "helloStringProperty"
	err = putMsgHandle.SetMP(smpo, name, pd, v1)
	if err != nil {
		fmt.Printf("PROP1: %v\n", err)
	}

	name = "PROP2INT"
	v2 := 42
	err = putMsgHandle.SetMP(smpo, name, pd, int(v2))
	if err != nil {
		fmt.Printf("PROP2: %v\n", err)
	}

	name = "PROP2AINT32"
	v2a := 4242
	err = putMsgHandle.SetMP(smpo, name, pd, int32(v2a))
	if err != nil {
		fmt.Printf("PROP2: %v\n", err)
	}

	name = "PROP2BINT64"
	v2b := 424242
	err = putMsgHandle.SetMP(smpo, name, pd, int64(v2b))
	if err != nil {
		fmt.Printf("PROP2: %v\n", err)
	}

	name = "PROP2CINT16"
	v2c := 4242
	err = putMsgHandle.SetMP(smpo, name, pd, int16(v2c))
	if err != nil {
		fmt.Printf("PROP2: %v\n", err)
	}

	name = "PROP3BOOL"
	v3 := true
	err = putMsgHandle.SetMP(smpo, name, pd, v3)
	if err != nil {
		fmt.Println("PROP3: %v\n", err)
	}

	name = "PROP4BYTEARRAY"
	v4 := make([]byte, 6)
	for i := 0; i < 6; i++ {
		v4[i] = byte(0x64 + i)
	}
	err = putMsgHandle.SetMP(smpo, name, pd, v4)
	if err != nil {
		fmt.Println("PROP4: %v\n", err)
	}

	name = "PROP5NULL"
	err = putMsgHandle.SetMP(smpo, name, pd, nil)
	if err != nil {
		fmt.Println("PROP5: %v\n", err)
	}

	name = "PROP6DELETED"
	v6 := 10101
	err = putMsgHandle.SetMP(smpo, name, pd, v6)
	if err != nil {
		fmt.Println("PROP6: %v\n", err)
	}

	// Use the DltMP function to remove a property from the set. So we should
	// end up with 1 fewer properties on the message
	dmpo := ibmmq.NewMQDMPO()
	err = putMsgHandle.DltMP(dmpo, name)
	if err != nil {
		fmt.Println(err)
	}

	name = "PROP7BYTE"
	v7 := (byte)(36)
	err = putMsgHandle.SetMP(smpo, name, pd, v7)
	if err != nil {
		fmt.Printf("PROP7: %v\n", err)
	}

	name = "PROP8FLOAT32"
	v8 := (float32)(3.14159)
	err = putMsgHandle.SetMP(smpo, name, pd, v8)
	if err != nil {
		fmt.Printf("PROP8: %v\n", err)
	}

	name = "PROP9FLOAT64"
	v9 := (float64)(3.14159)
	err = putMsgHandle.SetMP(smpo, name, pd, v9)
	if err != nil {
		fmt.Printf("PROP9: %v\n", err)
	}

	return err
}

/*
Display the properties from the retrieved message. They should match those
that were applied in the setProperties function
*/
func printProperties(getMsgHandle ibmmq.MQMessageHandle) {
	impo := ibmmq.NewMQIMPO()
	pd := ibmmq.NewMQPD()

	impo.Options = ibmmq.MQIMPO_CONVERT_VALUE | ibmmq.MQIMPO_INQ_FIRST
	for propsToRead := true; propsToRead; {
		name, value, err := getMsgHandle.InqMP(impo, pd, "%")
		impo.Options = ibmmq.MQIMPO_CONVERT_VALUE | ibmmq.MQIMPO_INQ_NEXT
		if err != nil {
			mqret := err.(*ibmmq.MQReturn)
			if mqret.MQRC != ibmmq.MQRC_PROPERTY_NOT_AVAILABLE {
				fmt.Println(err)
			} else {
				fmt.Println("All properties read")
			}

			propsToRead = false
		} else {
			fmt.Printf("Name: '%s' Value '%v' \n", name, value)
		}
	}
}

func main() {
	os.Exit(mainWithRc())
}

// The real main function is here to set a return code.
func mainWithRc() int {
	var putmqmd *ibmmq.MQMD

	// The default queue manager and queue to be used. These can be overridden on command line.
	qMgrName := "QM1"
	qName := "DEV.QUEUE.1"

	fmt.Println("Sample AMQSPROP.GO start")

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

	// Create a message handle that will be used to set message properties
	if err == nil {
		cmho := ibmmq.NewMQCMHO()
		putMsgHandle, err = qMgrObject.CrtMH(cmho)
		if err != nil {
			fmt.Println(err)
		}
	}

	// Use a separate message handle to inquire on the message properties
	if err == nil {
		cmho := ibmmq.NewMQCMHO()
		getMsgHandle, err = qMgrObject.CrtMH(cmho)
		if err != nil {
			fmt.Println(err)
		}
	}

	if err == nil {
		// And call a function to set the various properties
		err = setProperties(putMsgHandle)
	}

	// PUT the message to the queue
	if err == nil {
		putmqmd = ibmmq.NewMQMD()
		pmo := ibmmq.NewMQPMO()

		pmo.Options = ibmmq.MQPMO_NO_SYNCPOINT

		// Set the handle that holds the properties
		pmo.OriginalMsgHandle = putMsgHandle

		// Create the contents to include a timestamp just to prove when it was created
		msgData := "Hello from Go at " + time.Now().Format(time.RFC3339)
		buffer := []byte(msgData)
		putmqmd.Format = ibmmq.MQFMT_STRING

		// Now put the message to the queue
		err = qObject.Put(putmqmd, pmo, buffer)
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

		// Set the message handle so that properties can be retrieved and
		// force the properties to be returned in the handle regardless of
		// the queue attributes
		gmo.MsgHandle = getMsgHandle
		gmo.Options |= ibmmq.MQGMO_PROPERTIES_IN_HANDLE

		// Create a buffer for the message data. This one is large enough
		// for the messages put by the amqsput sample.
		buffer := make([]byte, 1024)

		// Now we can try to get the message. Don't care about the actual message
		// data in this sample, just the returned properties
		_, err = qObject.Get(getmqmd, gmo, buffer)

		if err != nil {
			fmt.Println(err)
		} else {
			// A message has been retrieved. Now display its properties
			printProperties(getMsgHandle)
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
