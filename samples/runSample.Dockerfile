# © Copyright IBM Corporation 2019, 2020
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ARG BASE_IMAGE=ubuntu:19.10
FROM $BASE_IMAGE

ARG GOPATH_ARG="/go"

ENV GOVERSION=1.13 \
    GOPATH=$GOPATH_ARG \
    ORG="github.com/ibm-messaging"

# Install the Go compiler and Git
RUN export DEBIAN_FRONTEND=noninteractive \
  && bash -c 'source /etc/os-release; \
     echo "deb http://archive.ubuntu.com/ubuntu/ ${UBUNTU_CODENAME} main restricted" > /etc/apt/sources.list; \
     echo "deb http://archive.ubuntu.com/ubuntu/ ${UBUNTU_CODENAME}-updates main restricted" >> /etc/apt/sources.list; \
     echo "deb http://archive.ubuntu.com/ubuntu/ ${UBUNTU_CODENAME}-backports main restricted universe" >> /etc/apt/sources.list; \
     echo "deb http://archive.ubuntu.com/ubuntu/ ${UBUNTU_CODENAME} universe" >> /etc/apt/sources.list; \
     echo "deb http://archive.ubuntu.com/ubuntu/ ${UBUNTU_CODENAME}-updates universe" >> /etc/apt/sources.list;' \
  && apt-get update \
  && apt-get install -y --no-install-recommends \
    golang-${GOVERSION} \
    git \
    ca-certificates \
    curl \
    tar \
    bash \
    go-dep \
    build-essential \
  && rm -rf /var/lib/apt/lists/*

# Create location for the go programs and the MQ installation
RUN mkdir -p $GOPATH/src $GOPATH/bin $GOPATH/pkg \
  && chmod -R 777 $GOPATH \
  && mkdir -p /opt/mqm \
  && chmod a+rx /opt/mqm

# Location of the downloadable MQ client package \
ENV RDURL="https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqdev/redist" \
    RDTAR="IBM-MQC-Redist-LinuxX64.tar.gz" \
    VRMF=9.2.1.0

# Install the MQ client from the Redistributable package. This also contains the
# header files we need to compile against. Setup the subset of the package
# we are going to keep - the genmqpkg.sh script removes unneeded parts
ENV genmqpkg_incnls=1 \
    genmqpkg_incsdk=1 \
    genmqpkg_inctls=1

RUN cd /opt/mqm \
 && curl -LO "$RDURL/$VRMF-$RDTAR" \
 && tar -zxf ./*.tar.gz \
 && rm -f ./*.tar.gz \
 && bin/genmqpkg.sh -b /opt/mqm

# We need the Go compiler in our PATH
ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/lib/go-$GOVERSION/bin

# Copy the source file over. We also need a go.mod file
# The source for that has a different name in the repo so it doesn't accidentally get
# used and we rename it during this copy.
COPY amqsput.go $GOPATH_ARG/src
COPY runSample.gomod $GOPATH_ARG/src/go.mod

# Do the actual compile. This will automatically download the ibmmq package
RUN cd $GOPATH_ARG/src && go build -o $GOPATH_ARG/bin/amqsput amqsput.go

# The startup script will set MQSERVER and optionally set
# more environment variables that will be passed to amqsput through this entrypoint.
ENTRYPOINT /go/bin/amqsput $QUEUE $QMGR
