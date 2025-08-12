/*
Package ibmmqotel provides an interface between an OpenTelemetry-instrumented
Go program and the IBM MQ client. It is a separate package to avoid dragging
OTel dependencies into applications that are not using OTel.

It is essentially a propagator, transforming Span/Trace information
in and out of the MQ Message Properties that are carried between
MQ nodes.

The main ibmmq package has the hook points in place to allow
this package to register itself and be called at the appropriate points

An instrumented application needs to

  - Import this package

  - call ibmmqotel.Setup()

  - set Ctx field in MQPMO/MQGMO structures passed to the Put/Get/CB verbs in the ibmmq package,
    passing the relevant context associated with the current Span

If using the MQCB/CallbackFunc for asynchronous consumption, the Context is available in the
MQCBD structure as the Ctx field
*/
package ibmmqotel

/*
Copyright (c) IBM Corporation 2024

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
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	mq "github.com/ibm-messaging/mq-golang/v5/ibmmq"

	oteltrace "go.opentelemetry.io/otel/trace"
)

// Stash information about the in-use queue
type propOptions struct {
	propCtl int32 // The PROPCTL attribute on the queue, or -1 if unknown
	gmo     int32 // Currently-active GMO Options value so we can reset
}

var (
	otelInit    = false // Has the module been initialised
	otelEnabled = false // Are we going to actually do any OTel work

	objectMapHandle  = make(map[string]*mq.MQMessageHandle)
	objectMapOptions = make(map[string]*propOptions)

	omh sync.Mutex
	omo sync.Mutex
)

const (
	// The names of the properties to set in the <usr> folder of the RFH2 for propagation
	traceparent = "traceparent"
	tracestate  = "tracestate"

	// Use this as a bitmap filter to pull out relevant value from GMO.
	// The AS_Q_DEF value is 0 so would not contribute.
	getPropsOptions = mq.MQGMO_PROPERTIES_FORCE_MQRFH2 |
		mq.MQGMO_PROPERTIES_IN_HANDLE |
		mq.MQGMO_NO_PROPERTIES |
		mq.MQGMO_PROPERTIES_COMPATIBILITY

	// Options in an MQOPEN that mean we might do MQGET
	// Do not include BROWSE variants
	openGetOptions = mq.MQOO_INPUT_AS_Q_DEF |
		mq.MQOO_INPUT_SHARED |
		mq.MQOO_INPUT_EXCLUSIVE
)

// Executed at package startup
func init() {
	// Set this so any underlying equivalent code in the C library will not try to do its own thing
	os.Setenv("AMQ_OTEL_INSTRUMENTED", "true")
}

// Go's maps are not concurrently accessible. So we need to add locks. This might not be required for
// all operations, but better to be safe. These are trivial functions, but we might want to add tracing/debug
// occasionally. So it's better to wrap the real calls.
func lockMapOptions() {
	omo.Lock()
}
func unlockMapOptions() {
	omo.Unlock()
}

func lockMapHandle() {
	omh.Lock()
}
func unlockMapHandle() {
	omh.Unlock()
}

// This is the function that applications have to call, to ensure the OTel
// interface is available.
func Setup() {
	if otelInit {
		return
	}

	otelInit = true

	if os.Getenv("MQIGOOTEL_TRACE") != "" {
		SetTrace(true)
	}

	traceEntry("Setup")

	// Unless explicitly disabled, setup the function callbacks from
	// the main ibmmq package
	if os.Getenv("MQIGO_NOOTEL") == "" {
		otelEnabled = true
		f := mq.MQOtelFuncs{
			Disc:           otelDisc,
			Open:           otelOpen,
			Close:          otelClose,
			PutTraceBefore: otelPutTraceBefore,
			PutTraceAfter:  otelPutTraceAfter,
			GetTraceBefore: otelGetTraceBefore,
			GetTraceAfter:  otelGetTraceAfter,
		}
		mq.SetOtelFuncs(f)
	}

	logTrace("OTelEnabled: %v", otelEnabled)
	traceExit("Setup")
}

// This function can be useful to create a unique key related to the hConn and hObj values
// where the object name is not sufficiently unique
func objectKey(hc *mq.MQQueueManager, ho *mq.MQObject) string {
	suffix := "*"
	if ho != nil {
		suffix = strconv.Itoa(int(ho.GetValue()))
	}
	return strconv.Itoa(int(hc.GetValue())) + "/" + suffix
}

// Do we have a MsgHandle for this hConn? If not, create a new one
func getMsgHandle(hConn *mq.MQQueueManager, hObj *mq.MQObject) *mq.MQMessageHandle {
	key := objectKey(hConn, hObj)
	lockMapHandle()
	if _, ok := objectMapHandle[key]; !ok {

		cmho := mq.NewMQCMHO()
		mh, err := hConn.CrtMH(cmho)
		if err == nil {
			objectMapHandle[key] = &mh
		} else {
			fmt.Printf(err.Error())
		}

	}

	o := objectMapHandle[key]
	unlockMapHandle()

	return o
}

// Is the GMO/PMO MsgHandle one that we allocated?
func compareMsgHandle(hConn *mq.MQQueueManager, hObj *mq.MQObject, mh *mq.MQMessageHandle) bool {
	rc := false
	key := objectKey(hConn, hObj)
	lockMapHandle()
	if oh, ok := objectMapHandle[key]; ok {
		mhLocal := oh
		if mhLocal.GetValue() == mh.GetValue() {
			rc = true
		}
	}
	unlockMapHandle()
	return rc
}

// Is there a property of the given name?
func propsContain(mh mq.MQMessageHandle, prop string) bool {
	rc := false

	pd := mq.NewMQPD()
	impo := mq.NewMQIMPO()
	impo.Options = mq.MQIMPO_CONVERT_VALUE | mq.MQIMPO_INQ_FIRST

	// Don't care about the actual value of the property, just that
	// it exists.
	_, _, err := mh.InqMP(impo, pd, prop)
	if err != nil {
		rc = true
	}

	return rc
}

// Extract a substring from the RFH2 properties
func extractRFH2PropVal(props []string, prop string) string {
	propXml := "<" + prop + ">"
	val := ""

	for i := 0; i < len(props); i++ {
		propEntry := props[i]
		idx := strings.Index(propEntry, propXml)
		if idx != -1 {
			start := propEntry[idx+len(propXml):]
			// Where does the next tag begin
			end := strings.Index(start, "<")
			if end != -1 {
				val = start[0:end]
				break
			}
		}
	}
	logTrace("Searched for %s in RFH2 msg. Found: \"%s\"", prop, val)
	return val
}

// Get rid of entries in the hconn/hobj maps when the application
// calls MQDISC
func otelDisc(qMgr *mq.MQQueueManager) {
	traceEntry("disc")

	// Both the maps are keyed by a string which begins with
	// the hConn value. As this is MQDISC, we don't care about
	// any specific hObj
	prefix := fmt.Sprintf("%d/", qMgr.GetValue())
	lockMapHandle()
	for k, mh := range objectMapHandle {
		if strings.HasPrefix(k, prefix) {
			dmho := mq.NewMQDMHO()
			mh.DltMH(dmho)
			delete(objectMapHandle, k)
		}
	}
	unlockMapHandle()

	// And delete information about any OPENed object too
	lockMapOptions()
	for k, _ := range objectMapOptions {
		if strings.HasPrefix(k, prefix) {
			delete(objectMapOptions, k)
		}
	}
	unlockMapOptions()

	traceExit("disc")
	return
}

// When a queue is opened for INPUT, then it will help to
// know the PROPCTL setting so we know if we can add a MsgHandle or to expect
// an RFH2 response. If the MQINQ fails, that's OK - we'll just ignore the error
// but might not be able to get any property/RFH from an inbound message
//
// Note that we can't (and don't need to) do the same for an MQPUT1 because the
// information we are trying to discover is only useful on MQGET/CallBack.
func otelOpen(hObj *mq.MQObject, od *mq.MQOD, openOptions int32) {
	var propCtl int32

	traceEntry("open")

	if !otelEnabled {
		traceExit("open")
		return
	}

	// Do the MQINQ and stash the information
	// Only care if there's an INPUT option. We do the MQINQ on every relevant MQOPEN
	// because it might change between an MQCLOSE and a subsequent MQOPEN. The MQCLOSE
	// will, in any case, have discarded the entry from this map.
	// If the user opened the queue with MQOO_INQUIRE, then we can reuse the object handle.
	// Otherwise we have to do our own open/inq/close.
	if (od.ObjectType == mq.MQOT_Q) && (openOptions&openGetOptions) != 0 {
		hConn := hObj.GetHConn()
		key := objectKey(hConn, hObj)

		propCtl = 0
		selectors := []int32{mq.MQIA_PROPERTY_CONTROL}
		if openOptions&mq.MQOO_INQUIRE != 0 {
			logTrace("open: Reusing existing hObj")
			values, err := hObj.Inq(selectors)
			if err == nil {
				logTrace("Inq Responses: %+v", values)
				propCtl = values[selectors[0]].(int32)
			} else {
				logTrace("open: Inq err %s", err.Error())
				propCtl = -1
			}
		} else {

			inqOd := mq.NewMQOD()
			inqOd.ObjectName = od.ObjectName
			inqOd.ObjectQMgrName = od.ObjectQMgrName
			inqOd.ObjectType = mq.MQOT_Q
			inqOpenOptions := mq.MQOO_INQUIRE

			logTrace("open: pre-Reopen")
			// This gets a little recursive as this Open will end up calling back into this function. But
			// as it's only doing MQOO_INQUIRE, then we don't nest any further
			inqHObj, err := hConn.Open(inqOd, inqOpenOptions)

			if err != nil {
				logTrace("open: Reopen err %s", err.Error())
				propCtl = -1
			} else {
				values, err := inqHObj.Inq(selectors)
				if err == nil {
					logTrace("Inq Responses: %+v", values)
					propCtl = values[selectors[0]].(int32)
				} else {
					logTrace("open: Inq err %s", err.Error())
					propCtl = -1
				}

				inqHObj.Close(0) // Ignore any error
			}

		}
		// Create an object to hold the discovered value
		options := propOptions{propCtl: propCtl}
		// replace any existing value for this object handle
		lockMapOptions()
		objectMapOptions[key] = &options
		unlockMapOptions()

	} else {
		logTrace("open: not doing Inquire")
	}

	traceExit("open")
	return
}

// Called during the MQCLOSE
func otelClose(hObj *mq.MQObject) {
	traceEntry("close")

	key := objectKey(hObj.GetHConn(), hObj)
	lockMapOptions()
	delete(objectMapOptions, key)
	unlockMapOptions()

	traceExit("close")
	return
}

// Discover any active Span and convert its context into message properties
// This is called during MQPUT and MQPUT1
func otelPutTraceBefore(otelOpts mq.OtelOpts, x *mq.MQQueueManager, md *mq.MQMD,
	pmo *mq.MQPMO, buffer []byte) {

	var mh mq.MQMessageHandle
	var mho mq.MQMessageHandle

	traceEntry("putTraceBefore")

	if !otelEnabled {
		traceExit("putTraceBefore")
		return
	}

	ctx := otelOpts.Context

	skipParent := false
	skipState := false

	// Is the app already using a MsgHandle for its PUT? If so, we
	// can piggy-back on that. If not, then we need to use our
	// own handle. That handle can be reused for all PUTs/GETs on this
	// hConn. This works, even when the app is primarily using an RFH2 for
	// its own properties - the RFH2 and the Handle contents are merged.
	//
	// If there was an app-provided handle, then have they set
	// either of the key properties? If so, then we will
	// leave them alone as we are not trying to create a new span in this
	// layer.
	if mq.IsUsableHandle(pmo.NewMsgHandle) {
		mh = pmo.NewMsgHandle
		if propsContain(mh, traceparent) {
			skipParent = true
		}
		if propsContain(mh, tracestate) {
			skipState = true
		}
	} else if mq.IsUsableHandle(pmo.OriginalMsgHandle) {
		mho = pmo.OriginalMsgHandle
		if propsContain(mho, traceparent) {
			skipParent = true
		}
		if propsContain(mho, tracestate) {
			skipState = true
		}
	} else {
		mh = *getMsgHandle(x, nil)
		pmo.OriginalMsgHandle = mh
	}

	// Make sure we've got one of the handles set
	if mq.IsUsableHandle(mho) && !mq.IsUsableHandle(mh) {
		mh = mho
	}

	// The message MIGHT have been constructed with an explicit RFH2
	// header. Unlikely, but possible as we tend to prefer properties. If so, then we extract the properties
	// from that header (assuming there's only a single structure, and it's not
	// chained). Then very simply look for the property names in there as strings. These tests would
	// incorrectly succeed if someone had put "traceparent" into a non-"usr" folder but that would be
	// very unexpected.
	if md.Format == mq.MQFMT_RF_HEADER_2 {
		hdr, _, _ := mq.GetHeader(md, buffer)
		rfh2, ok := hdr.(*mq.MQRFH2)
		if ok {
			props := rfh2.Get(buffer)

			for i := 0; i < len(props); i++ {
				if strings.Contains(props[i], "<"+traceparent+">") {
					skipParent = true
				}
				if strings.Contains(props[i], "<"+tracestate+">") {
					skipState = true
				}
			}
		}
	}

	// We're now ready to extract the context information and set the MQ message property
	// We are not going to try to propagate baggage via another property
	span := oteltrace.SpanFromContext(ctx)
	logTrace("Span/Context: %+v\n", span)

	if span.SpanContext().IsValid() {
		smpo := mq.NewMQSMPO()
		pd := mq.NewMQPD()

		logTrace("About to extract context from an active span")
		if !skipParent {
			traceId := span.SpanContext().TraceID().String()
			spanId := span.SpanContext().SpanID().String()
			traceFlags := span.SpanContext().TraceFlags()
			traceFlagsString := "01"
			if traceFlags != 1 {
				traceFlagsString = "00"
			}

			// This is the W3C-defined format for the trace property
			value := fmt.Sprintf("%s-%s-%s-%s", "00", traceId, spanId, traceFlagsString)
			logTrace("Setting %s to %s\n", traceparent, value)

			err := mh.SetMP(smpo, traceparent, pd, value)
			if err != nil {
				// Should we fail silently?
				//logError(err.Error())
			}
		}

		if !skipState {
			// Need to convert any TraceState map to a single serialised string
			ts := span.SpanContext().TraceState()
			value := ts.String()
			if value != "" {
				logTrace("Setting %s to %s", tracestate, value)
				err := mh.SetMP(smpo, tracestate, pd, value)
				if err != nil {
					// Should we fail silently?
					//logError(err.Error())
				}
			}
		}
	}

	traceExit("putTraceBefore")

}

// If we added our own MsgHandle to the PMO, then remove it
// before returning to the application. We don't need to delete
// the handle as it can be reused for subsequent PUTs on this hConn
func otelPutTraceAfter(otelOpts mq.OtelOpts, hConn *mq.MQQueueManager, gopmo *mq.MQPMO) {
	traceEntry("putTraceAfter\n")

	if !otelEnabled {
		traceExit("putTraceAfter")
		return
	}

	// ctx := otelOpts.Context

	mh := gopmo.OriginalMsgHandle
	if compareMsgHandle(hConn, nil, &mh) {
		gopmo.OriginalMsgHandle = mq.MQMessageHandle{}
	}

	traceExit("putTraceAfter")

	return
}

func otelGetTraceBefore(otelOpts mq.OtelOpts, hConn *mq.MQQueueManager, hObj *mq.MQObject, gogmo *mq.MQGMO, async bool) {
	var propCtl int32

	traceEntry("getTraceBefore")

	if !otelEnabled {
		return
	}

	// Option combinations:
	// MQGMO_NO_PROPERTIES: Always add our own handle
	// MQGMO_PROPERTIES_IN_HANDLE: Use it
	// MQGMO_PROPERTIES_COMPAT/FORCE_RFH2: Any returned properties will be in RFH2
	// MQGMO_PROPERTIES_AS_Q_DEF:
	//      PROPCTL: NONE: same as GMO_NO_PROPERTIES
	//               ALL/COMPATV6COMPAT: Any returned properties will be either in RFH2 or Handle if supplied
	//               FORCE: Any returned properties will be in RFH2
	propGetOptions := gogmo.Options & getPropsOptions
	logTrace("propGetOptions: %d", propGetOptions)

	if mq.IsUsableHandle(gogmo.MsgHandle) {
		logTrace("Using app-supplied msg handle")
	} else {
		key := objectKey(hConn, hObj)
		propCtl = -1
		lockMapOptions()
		if opts, ok := objectMapOptions[key]; ok {
			propCtl = opts.propCtl
			// Stash the GMO options so they can be restored afterwards
			opts.gmo = gogmo.Options
			objectMapOptions[key] = opts
		}
		unlockMapOptions()

		// If we know that the app or queue is configured for not returning any properties, then we will override that into our handle
		if (propGetOptions == mq.MQGMO_NO_PROPERTIES) || (propGetOptions == mq.MQGMO_PROPERTIES_AS_Q_DEF && propCtl == mq.MQPROP_NONE) {
			gogmo.Options &= ^mq.MQGMO_NO_PROPERTIES
			gogmo.Options |= mq.MQGMO_PROPERTIES_IN_HANDLE
			ho := hObj
			if !async {
				ho = nil
			}
			gogmo.MsgHandle = *getMsgHandle(hConn, ho)
			logTrace("Using mqiotel msg handle. getPropsOptions=%d propCtl=%d\n", propGetOptions, propCtl)
		} else {
			// Hopefully they will have set something suitable on the PROPCTL attribute
			// or are asking specifically for an RFH2-style response
			logTrace("Not setting a message handle. propGetOptions=%d\n", propGetOptions)
		}
	}

	traceExit("getTraceBefore")

	return
}

// Extract the properties from the message, either with the properties API
// or from the RFH2. Construct an object with the span information.
// We do not try to extract/propagate any baggage-related fields.
func otelGetTraceAfter(otelOpts mq.OtelOpts, hObj *mq.MQObject, gogmo *mq.MQGMO, gomd *mq.MQMD, buffer []byte, async bool) int {

	traceEntry("getTraceAfter")

	traceparentVal := ""
	tracestateVal := ""

	if !otelEnabled {
		traceExit("getTraceAfter")
		return 0
	}

	ctx := otelOpts.Context
	removed := 0
	mh := gogmo.MsgHandle
	if mq.IsUsableHandle(mh) {

		pd := mq.NewMQPD()
		impo := mq.NewMQIMPO()
		impo.Options = mq.MQIMPO_CONVERT_VALUE | mq.MQIMPO_INQ_FIRST

		_, val, err := mh.InqMP(impo, pd, traceparent)
		if err == nil {
			logTrace("Found traceparent property: %s", val.(string))
			traceparentVal = val.(string)
		} else {
			mqret := err.(*mq.MQReturn)
			if mqret.MQRC != mq.MQRC_PROPERTY_NOT_AVAILABLE {
				// Should not happen
				logError(err.Error())
			}
		}

		_, val, err = mh.InqMP(impo, pd, tracestate)
		if err == nil {
			logTrace("Found tracestate property: %s", val.(string))
			tracestateVal = val.(string)
		} else {
			mqret := err.(*mq.MQReturn)
			if mqret.MQRC != mq.MQRC_PROPERTY_NOT_AVAILABLE {
				// Should not happen
				logError(err.Error())
			}
		}

		// If we added our own handle in the GMO, then reset
		// but don't do it for async callbacks.
		ho := hObj
		hc := hObj.GetHConn()
		if !async {
			ho = nil
		}

		if !async && compareMsgHandle(hc, ho, &mh) {
			gogmo.MsgHandle = mq.MQMessageHandle{}
			key := objectKey(hc, ho)
			lockMapOptions()
			if opts, ok := objectMapOptions[key]; ok {
				gogmo.Options = opts.gmo
			} else {
				gogmo.Options &= ^mq.MQGMO_PROPERTIES_IN_HANDLE
			}
			unlockMapOptions()
			logTrace("Removing our handle: hObj %v", hObj)
		}

		// Should we also remove the properties?
		// Probably not worth it, as any app dealing with
		// properties ought to be able to handle unexpected props.

	} else if gomd.Format == mq.MQFMT_RF_HEADER_2 {
		hdr, len, err := mq.GetHeader(gomd, buffer)
		if err == nil {
			rfh2, ok := hdr.(*mq.MQRFH2)
			if ok {
				props := rfh2.Get(buffer)

				traceparentVal = extractRFH2PropVal(props, traceparent)
				tracestateVal = extractRFH2PropVal(props, tracestate)
			}

			if otelOpts.RemoveRFH2 {
				// If the only properties in the RFH2 are the OTEL ones, then perhaps
				// the application cannot process the message. But we don't know for sure,
				// and maybe the properties are useful for higher-level span generation.
				// So we have an option to forcibly remove the RFH2.
				gomd.Format = rfh2.Format
				gomd.CodedCharSetId = rfh2.CodedCharSetId
				gomd.Encoding = rfh2.Encoding
				removed = len
			}
		}
	}

	// We now should have the relevant message properties to pass upwards
	currentSpan := oteltrace.SpanFromContext(ctx)
	logTrace("Span/Context: %+v\n", currentSpan)

	if currentSpan.SpanContext().IsValid() {

		haveNewContext := false

		msgContextConfig := oteltrace.SpanContextConfig{}

		if traceparentVal != "" {
			// Split the inbound traceparent value into its components to allow
			// construction of a new context
			elem := strings.Split(traceparentVal, "-")
			if len(elem) == 4 {
				// elem[0] = 0 (version indicator. Always 0 for now)

				traceID, err := oteltrace.TraceIDFromHex(elem[1])
				if err == nil {
					msgContextConfig.TraceID = traceID
				}

				spanID, err := oteltrace.SpanIDFromHex(elem[2])
				if err == nil {
					msgContextConfig.SpanID = spanID
				}

				// Final element can only be 00 or 01 (for now)
				if elem[3] == "00" {
					msgContextConfig.TraceFlags = oteltrace.TraceFlags(0)
				} else {
					msgContextConfig.TraceFlags = oteltrace.FlagsSampled
				}
				haveNewContext = true
			}
		}

		if tracestateVal != "" {
			// Build a TraceState structure by parsing the string
			ts, err := oteltrace.ParseTraceState(tracestateVal)
			if err == nil {
				msgContextConfig.TraceState = ts
				haveNewContext = true
			}
		}

		// If there is a current span, and we have at least one of the
		// parent/state properties, then create a link referencing these values
		if haveNewContext {
			msgContext := oteltrace.NewSpanContext(msgContextConfig)
			link := oteltrace.Link{SpanContext: msgContext}

			//logTrace("Created new context: %v", msgContext)
			currentSpan.AddLink(link)
			logTrace("Added link to current span")
			//logTrace("Updated span: %+v", currentSpan)

		} else {
			logTrace("No context properties found")
		}
	} else {
		// If there is no current active span, then we are not going to
		// try to create a new one, as we would have no way of knowing when it
		// ends. The properties are (probably) still available to the application if
		// it wants to work with them itself.
		logTrace("No current span to update")
	}

	logTrace("getTraceAfter: removed:%d", removed)
	traceExit("getTraceAfter")
	return removed
}
