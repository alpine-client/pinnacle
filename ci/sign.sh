#!/bin/bash

# Usage: ./sign.sh <path_to_exe>

# Check if the path to the executable is provided
if [ -z "$1" ]; then
  echo "Usage: $0 <path_to_exe>"
  exit 1
fi

# Set the path to the executable
EXE_PATH="$1"

# Ensure required environment variables are set
if [ -z "$GCP_KEYSTORE" ] || [ -z "$GCP_KEY_ALIAS" ]; then
  echo "Error: GCP_KEYSTORE and GCP_KEY_ALIAS environment variables must be set."
  exit 1
fi

# Sign the executable
{
  java -jar jsign.jar \
    --storetype GOOGLECLOUD \
    --storepass "$(gcloud auth print-access-token)" \
    --keystore "$GCP_KEYSTORE" \
    --alias "$GCP_KEY_ALIAS" \
    --certfile pinnacle-certificate.pem \
    --tsmode RFC3161 \
    --tsaurl http://timestamp.globalsign.com/tsa/r6advanced1 \
    "$EXE_PATH"
} &> /dev/null

if [ $? -eq 0 ]; then
  echo "Successfully signed $EXE_PATH."
else
  echo "Failed to sign $EXE_PATH."
  exit 1
fi
