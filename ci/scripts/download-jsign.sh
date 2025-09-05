#!/bin/bash
JSIGN_VERSION='7.2'
JSIGN_SHA256='9a99673bb011cc1d7faf00bc840a6b333fc8a9b596098da2f92946b68297f067'
JSIGN_URL="https://github.com/ebourg/jsign/releases/download/${JSIGN_VERSION}/jsign-${JSIGN_VERSION}.jar"

set -eux
curl -sSL ${JSIGN_URL} -o jsign.jar
echo ${JSIGN_SHA256} jsign.jar | sha256sum -c -
