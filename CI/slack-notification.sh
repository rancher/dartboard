#!/bin/bash

# Slack notification script for the dartboard e2e (deploy -> load -> Qase k6 suite) pipeline.
# Sends one summary message per build, reporting the aggregate Qase test run results.

set -e

# Function to send Slack notification
send_slack_notification() {
	local status="$1"
	local message="$2"
	local bot_token="$3"
	local channel="$4"

	if [ -z "$bot_token" ] || [ -z "$channel" ]; then
		echo "DARTBOARD_SLACK_BOT_TOKEN and DARTBOARD_SLACK_CHANNEL must be set"
		return 1
	fi

	echo $channel

	local payload
	payload=$(cat <<EOF
{
  "channel": "$channel",
  "text": "$message",
  "username": "Dartboard Test Reporter"
}
EOF
)

	# Bounded: this runs in the post block, so a stalled Slack call would hold
	# the build open against the outer timeout rather than the few seconds a
	# notification is worth.
	local response
	if ! response=$(curl --fail --silent --show-error -X POST \
		--connect-timeout 10 --max-time 30 --retry 2 --retry-delay 3 \
		-H "Content-type: application/json; charset=utf-8" \
		-H "Authorization: Bearer $bot_token" \
		--data "$payload" \
		"https://slack.com/api/chat.postMessage"); then
		echo "Failed to send Slack notification (HTTP/network error)"
		return 1
	fi

	# Slack returns HTTP 200 even for errors (e.g. invalid_auth, channel_not_found)
	if [[ "$response" != *'"ok":true'* ]]; then
		echo "Failed to send Slack notification (API error): $response"
		return 1
	fi

	return 0
}

# Emit a "• *Label:* value" message line, or nothing when the value is unavailable.
# Fields are skipped gracefully so the message stays correct as fields become optional.
append_field() {
	local label="$1"
	local value="$2"

	if [ -n "$value" ] && [ "$value" != "Unknown" ]; then
		printf '• *%s:* %s\\n' "$label" "$value"
	fi
}

# Main execution
send_jenkins_e2e_notification() {
	local build_status="$1"
	local build_number="${BUILD_NUMBER:-Unknown}"
	local build_url="${BUILD_URL:-}"
	local rancher_url="${RANCHER_URL:-}"

	# Get Slack bot token and channel from Secrets Manager
	local slack_bot_token="${DARTBOARD_SLACK_BOT_TOKEN:-}"
	local slack_channel="${DARTBOARD_SLACK_CHANNEL:-}"

	local emoji="✅"
	local status_text="SUCCESS"

	if [ "$build_status" = "FAILURE" ] || [ "$build_status" = "FAILED" ]; then
		emoji="❌"
		status_text="FAILED"
	elif [ "$build_status" = "UNSTABLE" ]; then
		emoji="⚠️"
		status_text="UNSTABLE"
	elif [ "$build_status" = "ABORTED" ]; then
		emoji="❌"
		status_text="ABORTED"
	elif [ "$build_status" = "SUCCESS" ] || [ "$build_status" = "PASSED" ]; then
		emoji="✅"
		status_text="SUCCESS"
	fi

	# Format the build number as a link
	local build_link="link"
	if [ "$build_number" != "Unknown" ]; then
		build_link="#$build_number"
	fi
	if [ -n "$build_url" ]; then
		build_link="<$build_url|$build_link>"
	fi

	local message="Build - $build_link - $status_text $emoji\n"

	# Format the Rancher version as a link
	local rancher_ver="${RANCHER_VERSION:-}"
	local rancher_link="link"
	if [ -n "$rancher_ver" ]; then
		rancher_link="$rancher_ver"
	fi
	if [ -n "$rancher_url" ]; then
		rancher_link="<$rancher_url|$rancher_link>"
	fi

	#format access log link here
	local access_log_url="${ACCESS_LOG_URL:-}"
	local access_log_link="link"
	if [ -n "$access_log_url" ]; then
		access_log_link="<$access_log_url|Access log>"
	fi

	local k8s_ver="${KUBERNETES_VERSION:-}"
	message+="Rancher: $rancher_link (on $k8s_ver) - $access_log_link \n"

	# Qase test run summary, published by the qase-k6-cli 'runstats' subcommand.
	local run_id="${QASE_RUN_ID:-${QASE_TESTOPS_RUN_ID:-}}"
	if [ -z "$run_id" ] && [ -n "${QASE_RUN_URL:-}" ]; then
		run_id="${QASE_RUN_URL##*/}"
	fi

	if [[ "${QASE_RUN_TOTAL:-}" =~ ^[0-9]+$ &&
		"${QASE_RUN_PASSED:-}" =~ ^[0-9]+$ &&
		"${QASE_RUN_FAILED:-}" =~ ^[0-9]+$ &&
		"${QASE_RUN_EXCEEDED_THRESHOLDS:-}" =~ ^[0-9]+$ ]]; then
		message+=$(append_field "Passed" "${QASE_RUN_PASSED}/${QASE_RUN_TOTAL}")
		message+=$(append_field "Threshold Exceeded" "${QASE_RUN_EXCEEDED_THRESHOLDS}")
		message+=$(append_field "Failed" "${QASE_RUN_FAILED}")

		local skipped="${QASE_RUN_SKIPPED:-}"
		if [ -z "$skipped" ]; then
			local executed=$((QASE_RUN_PASSED + QASE_RUN_FAILED + QASE_RUN_EXCEEDED_THRESHOLDS))
			if [ "$executed" -le "$QASE_RUN_TOTAL" ]; then
				skipped=$((QASE_RUN_TOTAL - executed))
			else
				skipped="0"
			fi
		fi
		message+=$(append_field "Skipped" "$skipped")
	else
		echo "Skipping Qase run stats: required QASE_RUN_* values are unavailable or invalid."
	fi

	local qase_test_run=""
	if [ -n "$run_id" ] && [ -n "${QASE_RUN_URL:-}" ]; then
		qase_test_run="<${QASE_RUN_URL}|${run_id}>"
	elif [ -n "$run_id" ]; then
		qase_test_run="${run_id}"
	elif [ -n "${QASE_RUN_URL:-}" ]; then
		qase_test_run="<${QASE_RUN_URL}|link>"
	fi
	message+=$(append_field "Qase Test Run" "$qase_test_run")

	message+="• *Timestamp:* $(date -u '+%Y-%m-%d %H:%M:%S UTC')"

	echo "Sending Slack notification for $build_status build..."
	if send_slack_notification "$build_status" "$message" "$slack_bot_token" "$slack_channel"; then
		echo "Slack notification sent successfully"
		return 0
	else
		echo "Failed to send Slack notification"
		return 1
	fi
}

# Execute main function with build status
send_jenkins_e2e_notification "$1"
