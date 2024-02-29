/*
Package mqmetric contains a set of routines common to several
commands used to export MQ metrics to different backend
storage mechanisms including Prometheus and InfluxDB.
*/
package mqmetric

/*
  Copyright (c) IBM Corporation 2016, 2024

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

/*
Need to turn the "friendly" name of each element into something
that is suitable for metric names.

Should also have consistency of units (always use seconds,
bytes etc), and organisation of the elements of the name (units last)

While we can't change the MQ-generated descriptions for its statistics,
we can reformat most of them heuristically here.
*/
import (
	"os"
	"regexp"
	"strings"

	"github.com/ibm-messaging/mq-golang/v5/ibmmq"
)

var (
	mHeur   = make(map[string]string)
	mManual = make(map[string]string)

	mapsFilled = false

	UseManualMetricMaps = false // May move to a config option at some point

)

// Convert the description of the resource publication element into a metric name
// This has always been done using a heuristic algorithm, but we may want to change some of them
// to a format preferred by the backend (eg OpenTelemetry).
func formatDescription(elem *MonElement) string {
	s := ""

	if !mapsFilled {
		// Have to opt into using the non-heuristic maps for now
		if os.Getenv("IBMMQ_MANUAL_METRIC_MAPS") != "" {
			UseManualMetricMaps = true
		} else {
			UseManualMetricMaps = false
		}
		fillMapManual()
		fillMapHeur()

		mapsFilled = true
	}

	if UseManualMetricMaps {
		s = formatDescriptionManual(elem.Description)
	}
	if s == "" {
		s = FormatDescriptionHeuristic(elem, true)
	}
	return s
}

func formatDescriptionManual(s string) string {
	desc := strings.ReplaceAll(s, " ", "_")
	desc = strings.ToLower(desc)

	// Is there an overriding metric name
	if s, ok := mManual[desc]; ok {
		return s
	} else {
		return ""
	}
}

// This function is exported so it can be called during the build/test process for automatic generation
// of some of the mHeur map elements.
func FormatDescriptionHeuristic(elem *MonElement, useMap bool) string {

	// The map has been generated once by hand and we will try to use it.
	desc := strings.ReplaceAll(elem.Description, " ", "_")
	desc = strings.ToLower(desc)

	if useMap {
		if s, ok := mHeur[desc]; ok {
			return s
		} else {
			logWarn("Element %s does not have a defined metric name", elem.Description)
		}
	}

	// If that fails, we go through generating the metric name using this set of
	// rules.
	s := elem.Description
	s = strings.Replace(s, " ", "_", -1)
	s = strings.Replace(s, "/", "_", -1)
	s = strings.Replace(s, "-", "_", -1)

	/* Make sure we don't have multiple underscores */
	multiunder := regexp.MustCompile("__*")
	s = multiunder.ReplaceAllLiteralString(s, "_")

	/* make it all lowercase. Not essential, but looks better */
	s = strings.ToLower(s)

	/* Remove all cases of bytes, seconds, count or percentage (we add them back in later) */
	s = strings.Replace(s, "_count", "", -1)
	s = strings.Replace(s, "_bytes", "", -1)
	s = strings.Replace(s, "_byte", "", -1)
	s = strings.Replace(s, "_seconds", "", -1)
	s = strings.Replace(s, "_second", "", -1)
	s = strings.Replace(s, "_percentage", "", -1)

	// Switch round a couple of specific names
	s = strings.Replace(s, "messages_expired", "expired_messages", -1)

	// Add the unit at end
	switch elem.Datatype {
	case ibmmq.MQIAMO_MONITOR_PERCENT, ibmmq.MQIAMO_MONITOR_HUNDREDTHS:
		s = s + "_percentage"
	case ibmmq.MQIAMO_MONITOR_MB, ibmmq.MQIAMO_MONITOR_GB:
		s = s + "_bytes"
	case ibmmq.MQIAMO_MONITOR_MICROSEC:
		s = s + "_seconds"
	default:
		if strings.Contains(s, "_total") {
			/* If we specify it is a total in description put that at the end */
			s = strings.Replace(s, "_total", "", -1)
			s = s + "_total"
		} else if strings.Contains(s, "log_") {
			/* Weird case where the log datatype is not MB or GB but should be bytes */
			s = s + "_bytes"
		}

		// There are some metrics that have both "count" and "byte count" in
		// the descriptions. They were getting mapped to the same string, so
		// we have to ensure uniqueness.
		if strings.Contains(elem.Description, "byte count") {
			s = s + "_bytes"
		} else if strings.HasSuffix(elem.Description, " count") && !strings.Contains(s, "_count") {
			s = s + "_count"
		}
	}

	logTrace("  [%s] in:%s out:%s", "formatDescription", elem.Description, s)

	return s
}

// These are the original heuristically-derived metric names. This was built from running
// the code once and capturing info from traces. Any new metrics should show up from tools run
// during the release process.
func fillMapHeur() {
	// Class: CPU
	mHeur["user_cpu_time_percentage"] = "user_cpu_time_percentage"
	mHeur["system_cpu_time_percentage"] = "system_cpu_time_percentage"
	mHeur["cpu_load_-_one_minute_average"] = "cpu_load_one_minute_average_percentage"
	mHeur["cpu_load_-_five_minute_average"] = "cpu_load_five_minute_average_percentage"
	mHeur["cpu_load_-_fifteen_minute_average"] = "cpu_load_fifteen_minute_average_percentage"
	mHeur["ram_free_percentage"] = "ram_free_percentage"
	mHeur["ram_total_bytes"] = "ram_total_bytes"
	mHeur["user_cpu_time_-_percentage_estimate_for_queue_manager"] = "user_cpu_time_estimate_for_queue_manager_percentage"
	mHeur["system_cpu_time_-_percentage_estimate_for_queue_manager"] = "system_cpu_time_estimate_for_queue_manager_percentage"
	mHeur["ram_total_bytes_-_estimate_for_queue_manager"] = "ram_total_estimate_for_queue_manager_bytes"

	// Class: Disk
	mHeur["mq_trace_file_system_-_bytes_in_use"] = "mq_trace_file_system_in_use_bytes"
	mHeur["mq_trace_file_system_-_free_space"] = "mq_trace_file_system_free_space_percentage"
	mHeur["mq_errors_file_system_-_bytes_in_use"] = "mq_errors_file_system_in_use_bytes"
	mHeur["mq_errors_file_system_-_free_space"] = "mq_errors_file_system_free_space_percentage"
	mHeur["mq_fdc_file_count"] = "mq_fdc_file_count"
	mHeur["queue_manager_file_system_-_bytes_in_use"] = "queue_manager_file_system_in_use_bytes"
	mHeur["queue_manager_file_system_-_free_space"] = "queue_manager_file_system_free_space_percentage"
	mHeur["log_-_bytes_in_use"] = "log_in_use_bytes"
	mHeur["log_-_bytes_max"] = "log_max_bytes"
	mHeur["log_file_system_-_bytes_in_use"] = "log_file_system_in_use_bytes"
	mHeur["log_file_system_-_bytes_max"] = "log_file_system_max_bytes"
	mHeur["log_-_physical_bytes_written"] = "log_physical_written_bytes"
	mHeur["log_-_logical_bytes_written"] = "log_logical_written_bytes"
	mHeur["log_-_write_latency"] = "log_write_latency_seconds"
	mHeur["log_-_current_primary_space_in_use"] = "log_current_primary_space_in_use_percentage"
	mHeur["log_-_workload_primary_space_utilization"] = "log_workload_primary_space_utilization_percentage"
	mHeur["log_-_bytes_required_for_media_recovery"] = "log_required_for_media_recovery_bytes"
	mHeur["log_-_bytes_occupied_by_reusable_extents"] = "log_occupied_by_reusable_extents_bytes"
	mHeur["log_-_bytes_occupied_by_extents_waiting_to_be_archived"] = "log_occupied_by_extents_waiting_to_be_archived_bytes"
	mHeur["log_-_write_size"] = "log_write_size_bytes"

	mHeur["appliance_data_-_bytes_in_use"] = "appliance_data_in_use_bytes"
	mHeur["appliance_data_-_free_space"] = "appliance_data_free_space_percentage"
	mHeur["system_volume_-_bytes_in_use"] = "system_volume_in_use_bytes"
	mHeur["system_volume_-_free_space"] = "system_volume_free_space_percentage"

	// Class: STATQ and STATMQI
	mHeur["mqinq_count"] = "mqinq_count"
	mHeur["failed_mqinq_count"] = "failed_mqinq_count"
	mHeur["mqset_count"] = "mqset_count"
	mHeur["failed_mqset_count"] = "failed_mqset_count"
	mHeur["interval_total_mqput/mqput1_count"] = "interval_mqput_mqput1_total_count"
	mHeur["interval_total_mqput/mqput1_byte_count"] = "interval_mqput_mqput1_total_bytes"
	mHeur["non-persistent_message_mqput_count"] = "non_persistent_message_mqput_count"
	mHeur["persistent_message_mqput_count"] = "persistent_message_mqput_count"
	mHeur["failed_mqput_count"] = "failed_mqput_count"
	mHeur["non-persistent_message_mqput1_count"] = "non_persistent_message_mqput1_count"
	mHeur["persistent_message_mqput1_count"] = "persistent_message_mqput1_count"
	mHeur["failed_mqput1_count"] = "failed_mqput1_count"
	mHeur["put_non-persistent_messages_-_byte_count"] = "put_non_persistent_messages_bytes"
	mHeur["put_persistent_messages_-_byte_count"] = "put_persistent_messages_bytes"
	mHeur["mqstat_count"] = "mqstat_count"
	mHeur["interval_total_destructive_get-_count"] = "interval_destructive_get_total_count"
	mHeur["interval_total_destructive_get_-_byte_count"] = "interval_destructive_get_total_bytes"
	mHeur["non-persistent_message_destructive_get_-_count"] = "non_persistent_message_destructive_get_count"
	mHeur["persistent_message_destructive_get_-_count"] = "persistent_message_destructive_get_count"
	mHeur["failed_mqget_-_count"] = "failed_mqget_count"
	mHeur["got_non-persistent_messages_-_byte_count"] = "got_non_persistent_messages_bytes"
	mHeur["got_persistent_messages_-_byte_count"] = "got_persistent_messages_bytes"
	mHeur["non-persistent_message_browse_-_count"] = "non_persistent_message_browse_count"
	mHeur["persistent_message_browse_-_count"] = "persistent_message_browse_count"
	mHeur["failed_browse_count"] = "failed_browse_count"
	mHeur["non-persistent_message_browse_-_byte_count"] = "non_persistent_message_browse_bytes"
	mHeur["persistent_message_browse_-_byte_count"] = "persistent_message_browse_bytes"
	mHeur["expired_message_count"] = "expired_message_count"
	mHeur["purged_queue_count"] = "purged_queue_count"
	mHeur["mqcb_count"] = "mqcb_count"
	mHeur["failed_mqcb_count"] = "failed_mqcb_count"
	mHeur["mqctl_count"] = "mqctl_count"
	mHeur["commit_count"] = "commit_count"
	mHeur["rollback_count"] = "rollback_count"
	mHeur["create_durable_subscription_count"] = "create_durable_subscription_count"
	mHeur["alter_durable_subscription_count"] = "alter_durable_subscription_count"
	mHeur["resume_durable_subscription_count"] = "resume_durable_subscription_count"
	mHeur["create_non-durable_subscription_count"] = "create_non_durable_subscription_count"
	mHeur["failed_create/alter/resume_subscription_count"] = "failed_create_alter_resume_subscription_count"
	mHeur["delete_durable_subscription_count"] = "delete_durable_subscription_count"
	mHeur["delete_non-durable_subscription_count"] = "delete_non_durable_subscription_count"
	mHeur["subscription_delete_failure_count"] = "subscription_delete_failure_count"
	mHeur["mqsubrq_count"] = "mqsubrq_count"
	mHeur["failed_mqsubrq_count"] = "failed_mqsubrq_count"
	mHeur["durable_subscriber_-_high_water_mark"] = "durable_subscriber_high_water_mark"
	mHeur["durable_subscriber_-_low_water_mark"] = "durable_subscriber_low_water_mark"
	mHeur["non-durable_subscriber_-_high_water_mark"] = "non_durable_subscriber_high_water_mark"
	mHeur["non-durable_subscriber_-_low_water_mark"] = "non_durable_subscriber_low_water_mark"
	mHeur["topic_mqput/mqput1_interval_total"] = "topic_mqput_mqput1_interval_total"
	mHeur["interval_total_topic_bytes_put"] = "interval_topic_put_total"
	mHeur["published_to_subscribers_-_message_count"] = "published_to_subscribers_message_count"
	mHeur["published_to_subscribers_-_byte_count"] = "published_to_subscribers_bytes"
	mHeur["non-persistent_-_topic_mqput/mqput1_count"] = "non_persistent_topic_mqput_mqput1_count"
	mHeur["persistent_-_topic_mqput/mqput1_count"] = "persistent_topic_mqput_mqput1_count"
	mHeur["failed_topic_mqput/mqput1_count"] = "failed_topic_mqput_mqput1_count"
	mHeur["mqconn/mqconnx_count"] = "mqconn_mqconnx_count"
	mHeur["failed_mqconn/mqconnx_count"] = "failed_mqconn_mqconnx_count"
	mHeur["concurrent_connections_-_high_water_mark"] = "concurrent_connections_high_water_mark"
	mHeur["mqdisc_count"] = "mqdisc_count"
	mHeur["mqopen_count"] = "mqopen_count"
	mHeur["failed_mqopen_count"] = "failed_mqopen_count"
	mHeur["mqclose_count"] = "mqclose_count"
	mHeur["failed_mqclose_count"] = "failed_mqclose_count"
	mHeur["mqput/mqput1_count"] = "mqput_mqput1_count"
	mHeur["mqput_byte_count"] = "mqput_bytes"
	mHeur["mqput_non-persistent_message_count"] = "mqput_non_persistent_message_count"
	mHeur["mqput_persistent_message_count"] = "mqput_persistent_message_count"
	mHeur["mqput1_non-persistent_message_count"] = "mqput1_non_persistent_message_count"
	mHeur["mqput1_persistent_message_count"] = "mqput1_persistent_message_count"
	mHeur["non-persistent_byte_count"] = "non_persistent_bytes"
	mHeur["persistent_byte_count"] = "persistent_bytes"
	mHeur["queue_avoided_puts"] = "queue_avoided_puts_percentage"
	mHeur["queue_avoided_bytes"] = "queue_avoided_percentage"
	mHeur["lock_contention"] = "lock_contention_percentage"
	mHeur["rolled_back_mqput_count"] = "rolled_back_mqput_count"
	mHeur["mqget_count"] = "mqget_count"
	mHeur["mqget_byte_count"] = "mqget_bytes"
	mHeur["destructive_mqget_non-persistent_message_count"] = "destructive_mqget_non_persistent_message_count"
	mHeur["destructive_mqget_persistent_message_count"] = "destructive_mqget_persistent_message_count"
	mHeur["destructive_mqget_non-persistent_byte_count"] = "destructive_mqget_non_persistent_bytes"
	mHeur["destructive_mqget_persistent_byte_count"] = "destructive_mqget_persistent_bytes"
	mHeur["mqget_browse_non-persistent_message_count"] = "mqget_browse_non_persistent_message_count"
	mHeur["mqget_browse_persistent_message_count"] = "mqget_browse_persistent_message_count"
	mHeur["mqget_browse_non-persistent_byte_count"] = "mqget_browse_non_persistent_bytes"
	mHeur["mqget_browse_persistent_byte_count"] = "mqget_browse_persistent_bytes"
	mHeur["destructive_mqget_fails"] = "destructive_mqget_fails"
	mHeur["destructive_mqget_fails_with_mqrc_no_msg_available"] = "destructive_mqget_fails_with_mqrc_no_msg_available"
	mHeur["destructive_mqget_fails_with_mqrc_truncated_msg_failed"] = "destructive_mqget_fails_with_mqrc_truncated_msg_failed"
	mHeur["mqget_browse_fails"] = "mqget_browse_fails"
	mHeur["mqget_browse_fails_with_mqrc_no_msg_available"] = "mqget_browse_fails_with_mqrc_no_msg_available"
	mHeur["mqget_browse_fails_with_mqrc_truncated_msg_failed"] = "mqget_browse_fails_with_mqrc_truncated_msg_failed"
	mHeur["rolled_back_mqget_count"] = "rolled_back_mqget_count"
	mHeur["messages_expired"] = "expired_messages"
	mHeur["queue_purged_count"] = "queue_purged_count"
	mHeur["average_queue_time"] = "average_queue_time_seconds"
	mHeur["queue_depth"] = "queue_depth"

	// Class: Native HA
	mHeur["synchronous_log_bytes_sent"] = "synchronous_log_sent_bytes"
	mHeur["catch-up_log_bytes_sent"] = "catch_up_log_sent_bytes"
	mHeur["log_write_average_acknowledgement_latency"] = "log_write_average_acknowledgement_latency"
	mHeur["log_write_average_acknowledgement_size"] = "log_write_average_acknowledgement_size"
	mHeur["backlog_bytes"] = "backlog_bytes"
	mHeur["backlog_average_bytes"] = "backlog_average_bytes"
}

// This map will contain only the additional elements where the heuristic version might not be suitable or
// match well-enough to some other implementations like the MQ Cloud package
func fillMapManual() {
	mManual["ram_total_bytes"] = "ram_size_bytes"

	// Don't need the "mq_" on the front
	mManual["mq_trace_file_system_-_bytes_in_use"] = "trace_file_system_in_use_bytes"
	mManual["mq_trace_file_system_-_free_space"] = "trace_file_system_free_space_percentage"
	mManual["mq_errors_file_system_-_bytes_in_use"] = "errors_file_system_in_use_bytes"
	mManual["mq_errors_file_system_-_free_space"] = "errors_file_system_free_space_percentage"
	mManual["mq_fdc_file_count"] = "fdc_files"

	// Flip around some of the elements
	mManual["create_durable_subscription_count"] = "durable_subscription_create_count"
	mManual["alter_durable_subscription_count"] = "durable_subscription_alter_count"
	mManual["resume_durable_subscription_count"] = "durable_subscription_resume_count"
	mManual["delete_durable_subscription_count"] = "durable_subscription_delete_count"

	mManual["create_non-durable_subscription_count"] = "non_durable_subscription_create_count"
	mManual["delete_non-durable_subscription_count"] = "non_durable_subscription_delete_count"

	mManual["failed_create/alter/resume_subscription_count"] = "failed_subscription_create_alter_resume_count"
	mManual["subscription_delete_failure_count"] = "failed_subscription_delete_count"

}
