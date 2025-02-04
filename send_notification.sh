#!/bin/bash

# Define the URL of the server to send the request to
TELEGRAM_HERMES_URL="http://localhost:10000/notification?sender=$HOSTNAME"

# Get the arg ument (status message)
if [ -n "$1" ]; then
    MESSAGE=$1
else
    echo -n "Enter message: "
    read MESSAGE
fi


# Send the request using curl
RESPONSE=$(curl -s -o  -w "%{http_code}" /dev/null -X POST \
-d "$MESSAGE" \
"$TELEGRAM_HERMES_URL")

echo "Notification sent"