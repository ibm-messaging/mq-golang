/*
This is a short sample to show how to connect to a remote
queue manager in a Go program by using a JWT token.

The sample makes an API call to the Token Server to authenticate a user,
and uses the returned token to connect to the queue manager which must have been
configured to recognise tokens.

There is no attempt in this sample to configure advanced security features
such as TLS for the queue manager connection. It does, however, use a minimal
TLS connection to the Token Server.

Defaults are provided for all parameters. Use "-?" to see the options. The
userid/password option is now deprecated; instead use the clientId/clientSecret
mechanism.

If an error occurs, the error is reported.
*/
package main

/*
  Copyright (c) IBM Corporation 2023, 2025

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
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"encoding/json"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

const (
	/* How to connect to a queue manager. Only a few basic parameters are used here - no TLS */
	defaultQMgrName       = "QM1"
	defaultChannel        = "SYSTEM.DEF.SVRCONN"
	defaultConnectionName = "localhost(1414)"

	/* Get these values from the Token issuer. */
	defaultTokenHost         = "localhost"
	defaultTokenPort         = 8443
	defaultTokenUserName     = "jwtuser"
	defaultTokenPassword     = "passw0rd"
	defaultTokenClientSecret = ""

	defaultTokenClientId = "jwtcid"
	defaultTokenRealm    = "mq"
)

type Config struct {
	qMgrName          string
	connectionName    string
	channel           string
	tokenHost         string
	tokenPort         int
	tokenUserName     string
	tokenPassword     string
	tokenClientId     string
	tokenClientSecret string
	tokenRealm        string
}

// We only care about one field in the JSON data returned from
// the call to the JWT server
type JWTResponse struct {
	AccessToken string `json:"access_token"`
}

var cf Config
var jwtResponseStruct JWTResponse

func main() {
	var err error
	var qMgr ibmmq.MQQueueManager
	var rc int
	token := ""

	fmt.Println("Sample AMQSJWT.GO start")

	initParms()
	err = parseParms()
	if err != nil {
		os.Exit(1)
	}

	// Allocate the MQCNO and MQCD structures needed for the CONNX call.
	cno := ibmmq.NewMQCNO()
	cd := ibmmq.NewMQCD()

	// Fill in required fields in the MQCD channel definition structure
	cd.ChannelName = cf.channel
	cd.ConnectionName = cf.connectionName

	// Reference the CD structure from the CNO and indicate that we definitely want to
	// use the client connection method.
	cno.ClientConn = cd
	cno.Options = ibmmq.MQCNO_CLIENT_BINDING

	token, err = obtainToken()
	if err == nil {
		if token != "" {
			csp := ibmmq.NewMQCSP()
			csp.Token = token
			l := 100
			if len(token) < 100 {
				l = len(token)
			}
			fmt.Printf("Token: %s\n", token[0:l]+"...")

			// Make the CNO refer to the CSP structure so it gets used during the connection
			cno.SecurityParms = csp
		} else {
			fmt.Printf("An empty token was returned")
			os.Exit(1)
		}
	} else {
		fmt.Printf("Could not get token: error %v\n", err)
		os.Exit(1)
	}

	// And now we can try to connect. Wait a short time before disconnecting.
	qMgr, err = ibmmq.Connx(cf.qMgrName, cno)
	if err == nil {
		fmt.Printf("MQCONN to QM %s succeeded.\n", cf.qMgrName)
		d, _ := time.ParseDuration("3s")
		time.Sleep(d)
		qMgr.Disc() // Ignore errors from disconnect as we can't do much about it anyway
		rc = 0
	} else {
		fmt.Printf("MQCONN to %s failed.\n", cf.qMgrName)
		fmt.Println(err)
		rc = int(err.(*ibmmq.MQReturn).MQCC)
	}

	fmt.Println("Done.")
	os.Exit(rc)
}

/*
 * Function to query a token from the token endpoint. Build the
 * command that is used to retrieve a JSON response from the token
 * server. Parse the response to find the token to be added into the MQCSP.
 */
func obtainToken() (string, error) {
	var resp *http.Response

	/*
	   These curl commands are the base of the call to get a token. It uses form data to
	   set the various parameters. The 2nd format is now preferred.

	   curl -k -X POST "https://$host:$port/realms/$realm/protocol/openid-connect/token" \
	        -H "Content-Type: application/x-www-form-urlencoded" \
	        -d "username=$user" -d "password=$password" \
	        -d "grant_type=password" -d "client_id=$cid" \
	        -o $output -Ss


	   curl -k -X POST "https://$host:$port/realms/$realm/protocol/openid-connect/token" \
	        -H "Content-Type: application/x-www-form-urlencoded" \
	        -d "client_secret=$secret" \
	        -d "grant_type=client_credentials" -d "client_id=$cid" \
	        -o $output -Ss
	*/

	/*
	   NOTE 1: The SkipVerify is is not a good idea for production, but it means we don't need to
	   set up a truststore for the token server's certificate. We will simply trust it - useful if it's a
	   development-level server with a self-signed cert.

	   NOTE 2: If you do choose to set up a truststore/keystore for the connection to the token server,
	   then they must be in a suitable format for OpenSSL (such as pem, p12), not the kdb format usually
	   used for an MQ connection.
	*/
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}

	// Build the URL. We will assume HTTPS. The path may need to change for different token servers.
	endpoint := fmt.Sprintf("https://%s:%d/realms/%s/protocol/openid-connect/token", cf.tokenHost, cf.tokenPort, cf.tokenRealm)

	// Fill in the pieces of data that the server expects
	formData := url.Values{
		"client_secret": {cf.tokenClientSecret},
		"client_id":     {cf.tokenClientId},
		"grant_type":    {"client_credentials"},
	}

	if cf.tokenClientSecret == "" {
		fmt.Printf("Username/Password authentication is deprecated. Use clientSecret instead.\n")
		formData = url.Values{
			"username":   {cf.tokenUserName},
			"password":   {cf.tokenPassword},
			"client_id":  {cf.tokenClientId},
			"grant_type": {"password"},
		}
	}

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(formData.Encode()))

	// And make the call to the token server
	if err == nil {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		resp, err = client.Do(req)
	}

	if err != nil {
		// we will get an error at this stage if the request fails, such as if the
		// requested URL is not found, or if the server is not reachable.
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
		return "", err
	}

	// If it all worked, we can parse the response. We don't need all of the returned
	// fields, only the token.
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	} else {
		// fmt.Printf("Got back a response: %s\n", data)
		err = json.Unmarshal(data, &jwtResponseStruct)
	}

	return jwtResponseStruct.AccessToken, err
}

// Command line parameters - set flags and defaults
func initParms() {
	flag.StringVar(&cf.qMgrName, "m", defaultQMgrName, "Queue Manager")
	flag.StringVar(&cf.connectionName, "connection", defaultConnectionName, "MQ Connection Name")
	flag.StringVar(&cf.channel, "channel", defaultChannel, "MQ Channel Name")
	flag.StringVar(&cf.tokenHost, "host", defaultTokenHost, "Hostname for the token server")
	flag.IntVar(&cf.tokenPort, "port", defaultTokenPort, "Portnumber for the token server")

	flag.StringVar(&cf.tokenUserName, "user", defaultTokenUserName, "UserName (deprecated)")
	flag.StringVar(&cf.tokenPassword, "password", defaultTokenPassword, "Password (deprecated)")
	flag.StringVar(&cf.tokenClientId, "clientId", defaultTokenClientId, "ClientId")
	flag.StringVar(&cf.tokenClientSecret, "clientSecret", defaultTokenClientSecret, "ClientSecret")

	flag.StringVar(&cf.tokenRealm, "realm", defaultTokenRealm, "Realm")
}

// Parse the command line. There should be nothing left after all the known parameters are used
func parseParms() error {
	var err error

	flag.Parse()

	if len(flag.Args()) > 0 {
		err = fmt.Errorf("Unexpected additional command line parameters given.")
		fmt.Fprintf(flag.CommandLine.Output(), "Error: %v\n\n", err)
		flag.Usage()
	}
	return err
}
