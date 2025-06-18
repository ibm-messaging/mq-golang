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

ARG BASE_IMAGE=registry.access.redhat.com/ubi8/go-toolset:1.21
FROM $BASE_IMAGE AS builder

ARG GOPATH_ARG="/go"
ARG GOARCH=amd64
ARG MQARCH=X64

ENV GOPATH=$GOPATH_ARG \
    ORG="github.com/ibm-messaging"

# Make sure we've got permissions inside the container
USER 0

# Create location for the git clone and MQ installation
RUN mkdir -p $GOPATH/src $GOPATH/bin $GOPATH/pkg \
  && chmod -R 777 $GOPATH \
  && mkdir -p $GOPATH/src/$ORG \
  && cd /tmp       \
  && mkdir -p /opt/mqm \
  && chmod a+rx /opt/mqm

# Location of the downloadable MQ client package
ARG RDURL_ARG="https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqdev/redist"
ENV RDURL=${RDURL_ARG} \
    RDTAR="IBM-MQC-Redist-Linux${MQARCH}.tar.gz" \
    VRMF=9.4.3.0

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

# Insert the script that will do the build
COPY --chmod=777 buildInDocker.sh $GOPATH

# Copy the rest of the source tree from this directory into the container
# And make sure it's readable by the id that will run the compiles (not just root)
ENV  REPO="mq-golang"
COPY --chmod=0777 . $GOPATH/src/$ORG/$REPO

# Set the entrypoint to the script that will do the compilation
WORKDIR $GOPATH
ENTRYPOINT [ "./buildInDocker.sh" ]
