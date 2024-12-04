#!/bin/bash

# Define the URL of the server to send the request to
TELEGRAM_HERMES_URL="http://localhost:10000/notification?sender_id=$HOSTNAME"

# Check if the argument is provided
if [ -z "$1" ]; then
echo "Error: No argument provided to the script."
exit 1;
fi

# Get the argument (status message)
STATUS_MESSAGE="$1"

# Construct the JSON payload
JSON_PAYLOAD=$(jq -n --arg text "$STATUS_MESSAGE" '{"text": $text}')

# Send the request using curl
RESPONSE=$(curl -s -o  -w "%{http_code}" /dev/null -X POST \
-H "Content-Type: application/json" \
-d "$JSON_PAYLOAD" \
"$TELEGRAM_HERMES_URL")

echo "Notification sent"