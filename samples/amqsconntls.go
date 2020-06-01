/*
This is a short sample to show how to connect to a remote
queue manager in a Go program with a TLS connection without requiring external
client configuration such as a CCDT. The basic
parameters are still needed here - channel name and connection information -
along with the queue manager name.

For example, run as
   amqsconn QMGR1 "SYSTEM.SSL.SVRCONN" "myhost.example.com(1414)"

If the MQSAMP_USER_ID environment variable is set, then a userid/password
flow is also made to authenticate to the queue manager.

If an error occurs, the error is reported.
*/
package main

/*
  Copyright (c) IBM Corporation 2017, 2019

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

   Contributors:
     Mark Taylor - Initial Contribution
*/

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

func main() {
	var qMgrName string
	var err error
	var qMgr ibmmq.MQQueueManager
	var rc int

	if len(os.Args) != 4 {
		fmt.Println("amqsconntls <qmgrname> <channelname> <conname>")
		fmt.Println("")
		fmt.Println("For example")
		fmt.Println("  amqsconntls QMGR1 \"SYSTEM.TLS.SVRCONN\" \"myhost.example.com(1414)\"")
		fmt.Println("All parameters are required.")
		os.Exit(1)
	}

	// Which queue manager do we want to connect to
	qMgrName = os.Args[1]

	// Allocate the MQCNO and MQCD structures needed for the CONNX call.
	cno := ibmmq.NewMQCNO()
	sco := ibmmq.NewMQSCO()
	cd := ibmmq.NewMQCD()

	// Fill in required fields in the MQCD channel definition structure
	cd.ChannelName = os.Args[2]
	cd.ConnectionName = os.Args[3]

	// The CipherSpec must match what is configured on the corresponding SVRCONN
	cd.SSLCipherSpec = "TLS_RSA_WITH_AES_128_CBC_SHA256"

	// The ClientAuth field says whether or not the client needs to present its own certificate
	// This too must match the SVRCONN definition.
	cd.SSLClientAuth = ibmmq.MQSCA_OPTIONAL

	// The keystore contains at least the certificate to verify the qmgr's cert (usually from
	// a Certificate Authority) and optionally the client's own certificate.
	// We could also optionally specify which certificate represents the client, based on its label
	// but don't need to do this when using the MQSCA_OPTIONAL flag.
	sco.KeyRepository = "./mykey"

	// Reference the CD structure from the CNO and indicate that we definitely want to
	// use the client connection method.
	cno.ClientConn = cd
	cno.Options = ibmmq.MQCNO_CLIENT_BINDING
	cno.SSLConfig = sco

	// Also fill in the userid and password if the MQSAMP_USER_ID
	// environment variable is set. This is the same variable used by the C
	// sample programs such as amqsput shipped with the MQ product.
	userId := os.Getenv("MQSAMP_USER_ID")
	if userId != "" {
		scanner := bufio.NewScanner(os.Stdin)
		csp := ibmmq.NewMQCSP()
		csp.AuthenticationType = ibmmq.MQCSP_AUTH_USER_ID_AND_PWD
		csp.UserId = userId

		fmt.Printf("Enter password for qmgr %s: \n", qMgrName)
		// For simplicity (it doesn't help with understanding the MQ parts of this program)
		// don't try to do anything special like turning off console echo for the password input
		scanner.Scan()
		csp.Password = scanner.Text()

		// Make the CNO refer to the CSP structure so it gets used during the connection
		cno.SecurityParms = csp
	}

	// And now we can try to connect. Wait a short time before disconnecting.
	qMgr, err = ibmmq.Connx(qMgrName, cno)
	if err == nil {
		fmt.Printf("Connection to %s succeeded.\n", qMgrName)
		d, _ := time.ParseDuration("3s")
		time.Sleep(d)
		qMgr.Disc() // Ignore errors from disconnect as we can't do much about it anyway
		rc = 0
	} else {
		fmt.Printf("Connection to %s failed.\n", qMgrName)
		fmt.Println(err)
		rc = int(err.(*ibmmq.MQReturn).MQCC)
	}

	fmt.Println("Done.")
	os.Exit(rc)

}
