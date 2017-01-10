/*
This is a short sample to show how to connect to a remote
queue manager in a Go program without requiring external
client configuration such as a CCDT. Only the basic
parameters are needed here - channel name and connection information -
along with the queue manager name.

For example, run as
   clienttest QMGR1 "SYSTEM.DEF.SVRCONN" "myhost.example.com(1414)"

There is no attempt in this sample to configure security features
such as userid/password or TLS.

If an error occurs, the error is reported.
*/
package main

/*
  Copyright (c) IBM Corporation 2017

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific

   Contributors:
     Mark Taylor - Initial Contribution
*/

import (
	"fmt"
	"ibmmq"
	"os"
	"time"
)

func main() {
	var qMgrName string
	var err error
	var qMgr ibmmq.MQQueueManager

	if len(os.Args) != 4 {
		fmt.Println("clienttest <qmgrname> <channelname> <conname>")
		fmt.Println("  All  parms required")
		os.Exit(1)
	}

	// Which queue manager do we want to connect to
	qMgrName = os.Args[1]

	// Allocate the MQCNO and MQCD structures needed for the
	// MQCONNX call.
	cno := ibmmq.NewMQCNO()
	cd := ibmmq.NewMQCD()

	// Fill in the required fields in the
	// MQCD channel definition structure
	cd.ChannelName = os.Args[2]
	cd.ConnectionName = os.Args[3]

	// Reference the CD structure from the CNO
	// and indicate that we want to use the client
	// connection method.
	cno.ClientConn = cd
	cno.Options = ibmmq.MQCNO_CLIENT_BINDING

	// And connect. Wait a short time before
	// disconnecting.
	qMgr, mqreturn, err := ibmmq.Connx(qMgrName, cno)
	if err == nil {
		fmt.Printf("Connection to %s succeeded.\n", qMgrName)
		d, _ := time.ParseDuration("10s")
		time.Sleep(d)
		qMgr.Disc()
	} else {
		fmt.Printf("Connection to %s failed.\n", qMgrName)
		fmt.Println(err)
	}

	fmt.Println("Done.")
	os.Exit((int)(mqreturn.MQCC))

}
