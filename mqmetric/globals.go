package mqmetric

/*
  Copyright (c) IBM Corporation 2016, 2021

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
	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

type sessionInfo struct {
	qMgr            ibmmq.MQQueueManager
	cmdQObj         ibmmq.MQObject
	replyQObj       ibmmq.MQObject
	qMgrObject      ibmmq.MQObject
	replyQBaseName  string
	statusReplyQObj ibmmq.MQObject

	platform         int32
	commandLevel     int32
	maxHandles       int32
	resolvedQMgrName string

	qmgrConnected bool
	queuesOpened  bool
	subsOpened    bool
}

type connectionInfo struct {
	si sessionInfo

	tzOffsetSecs         float64
	usePublications      bool
	useStatus            bool
	useResetQStats       bool
	showInactiveChannels bool

	// Only issue the warning about a '/' in an object name once.
	globalSlashWarning bool
	localSlashWarning  bool

	discoveryDone    bool
	publicationCount int

	objectStatus [GOOT_LAST_USED + 1]objectStatus
}

type objectStatus struct {
	init       bool
	objectSeen map[string]bool
}

const (
	GOOT_Q             = 1
	GOOT_NAMELIST      = 2
	GOOT_PROCESS       = 3
	GOOT_STORAGE_CLASS = 4
	GOOT_Q_MGR         = 5
	GOOT_CHANNEL       = 6
	GOOT_AUTH_INFO     = 7
	GOOT_TOPIC         = 8
	GOOT_COMM_INFO     = 9
	GOOT_CF_STRUC      = 10
	GOOT_LISTENER      = 11
	GOOT_SERVICE       = 12
	GOOT_APP           = 13
	GOOT_PUB           = 14
	GOOT_SUB           = 15
	GOOT_NHA           = 16
	GOOT_BP            = 17
	GOOT_PS            = 18
	GOOT_LAST_USED     = GOOT_PS
)

var ci *connectionInfo

// This are used externally so we need to maintain them as public exports until
// there's a major version change. At which point we will move them to fields of
// the objectStatus structure, retrievable by a getXXX() call instead of as public
// variables. The mq-metric-samples exporters will then need to change to match.
var (
	Metrics            AllMetrics
	QueueManagerStatus StatusSet
	ChannelStatus      StatusSet
	QueueStatus        StatusSet
	TopicStatus        StatusSet
	SubStatus          StatusSet
	UsagePsStatus      StatusSet
	UsageBpStatus      StatusSet
)

func newConnectionInfo() *connectionInfo {

	traceEntry("newConnectionInfo")

	ci := new(connectionInfo)
	ci.si.qmgrConnected = false
	ci.si.queuesOpened = false
	ci.si.subsOpened = false

	ci.usePublications = true
	ci.useStatus = false
	ci.useResetQStats = false
	ci.showInactiveChannels = false

	ci.globalSlashWarning = false
	ci.localSlashWarning = false
	ci.discoveryDone = false
	ci.publicationCount = 0

	for i := 1; i <= GOOT_LAST_USED; i++ {
		ci.objectStatus[i].init = false
	}

	traceExit("newConnectionInfo", 0)

	return ci
}

// Initialise this package with a default connection object for compatibility
func initConnection() {
	traceEntry("initConnection")

	ci = newConnectionInfo()

	traceExit("initConnection", 0)

}
