#!/bin/bash
JSIGN_VERSION='7.1'
JSIGN_SHA256='cfb48b07fdd2ee199bfc9e71d8dccdde67a799c4793602e446c7a101be62b3c4'
JSIGN_URL="https://github.com/ebourg/jsign/releases/download/${JSIGN_VERSION}/jsign-${JSIGN_VERSION}.jar"

set -eux
curl -sSL ${JSIGN_URL} -o jsign.jar
echo ${JSIGN_SHA256} jsign.jar | sha256sum -c -
