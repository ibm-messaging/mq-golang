# © Copyright IBM Corporation 2019, 2021
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


# This Dockerfile has two separate stages.
#
# The first stage is used to compile the Go program, where we need tools like the Go and C compilers.
# The second stage is a runtime-only container that holds just the things we need to
# execute the compiled program.
#
# Files and directories are copied from the builder container to the runtime container as needed.
#
# The base images are taken from the Red Hat Universal Base Images repository

# Start by setting some global variables that can still be overridden on the build command line.
ARG BASE_IMAGE=registry.access.redhat.com/ubi8/ubi
ARG GOPATH_ARG="/go"

###########################################################
# This starts the BUILD phase
###########################################################
FROM $BASE_IMAGE AS builder

ARG GOPATH_ARG
ENV GOPATH=$GOPATH_ARG \
    ORG="github.com/ibm-messaging"

# Install the Go compiler and some other tools. The version of Go that
# is available from the repository is new enough that we don't need to
# explicitly pull it from Google. Installing Go also gives prereqs like the C compiler.
RUN yum --disableplugin=subscription-manager -y install wget curl tar golang \
  && yum --disableplugin=subscription-manager clean all

# Create a location for the go programs and the MQ installation
RUN mkdir -p $GOPATH/src $GOPATH/bin $GOPATH/pkg \
  && chmod -R 777 $GOPATH \
  && cd /tmp       \
  && mkdir -p /opt/mqm \
  && chmod a+rx /opt/mqm

# Location of the downloadable MQ client package \
ENV RDURL="https://public.dhe.ibm.com/ibmdl/export/pub/software/websphere/messaging/mqdev/redist" \
    RDTAR="IBM-MQC-Redist-LinuxX64.tar.gz" \
    VRMF=9.3.0.0

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

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# Copy the source file over. We also need a go.mod file
# The source for that has a different name in the repo so it doesn't accidentally get
# used. We rename it during this copy.
COPY amqsput.go      $GOPATH_ARG/src
COPY runSample.gomod $GOPATH_ARG/src/go.mod

# Do the actual compile. This will automatically download the ibmmq package
RUN cd $GOPATH_ARG/src && go mod tidy && go build -o $GOPATH_ARG/bin/amqsput amqsput.go

###########################################################
# This starts the RUNTIME phase
###########################################################
# Now that there is a container with the compiled program we can build a smaller
# runtime image. Start from one of the smaller base container images.
FROM registry.access.redhat.com/ubi8/ubi-minimal
ARG GOPATH_ARG

# Copy over the MQ runtime client code. This does preserve the .h files used during compile
# but those are tiny so there's no real space-saving from deleting them here.
COPY --from=builder /opt/mqm /opt/mqm

# Create some directories that may be needed at runtime, depending on the container's
# security environment.
RUN mkdir -p /IBM/MQ/data/errors \
  && mkdir -p /.mqm \
  && chmod -R 777 /IBM \
  && chmod -R 777 /.mqm \
  && mkdir -p /go/bin

# The actual program has all of the Go runtime embedded; we only need the single
# binary along with the MQ client libraries, for it to run.
COPY --from=builder $GOPATH_ARG/bin/amqsput /go/bin/amqsput

# The startup script will set MQSERVER and optionally set more
# environment variables that will be passed to amqsput through this entrypoint.
ENTRYPOINT [ "/go/bin/amqsput" ]
