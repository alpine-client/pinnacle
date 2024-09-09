#!/bin/bash
JSIGN_VERSION='6.0'
JSIGN_SHA256='05ca18d4ab7b8c2183289b5378d32860f0ea0f3bdab1f1b8cae5894fb225fa8a'
JSIGN_URL="https://github.com/ebourg/jsign/releases/download/${JSIGN_VERSION}/jsign-${JSIGN_VERSION}.jar"

set -eux
curl -sSL ${JSIGN_URL} -o jsign.jar
echo ${JSIGN_SHA256} jsign.jar | sha256sum -c -
