#!/bin/bash

# © Copyright IBM Corporation 2019
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

# Script to build the Go libraries and sample programs from within a Docker container
# This script is INSIDE the container and used as the entrypoint.

export PATH="${PATH}:/usr/lib/go-${GOVERSION}/bin:/go/bin"
export CGO_CFLAGS="-I/opt/mqm/inc/"
export CGO_LDFLAGS_ALLOW="-Wl,-rpath.*"
export GOCACHE=/tmp/.gocache

echo "Running as " `id`

# Build the libraries so they can be used by other programs
cd $GOPATH/src

echo "Using compiler:"
go version

for pkg in $ORG/$REPO/ibmmq $ORG/$REPO/mqmetric
do
  lib=`basename $pkg`
  echo "Building package: $lib"
  go install  $pkg
done

# And do the sample program builds into the bin directory
cd $GOPATH
srcdir=src/$ORG/$REPO/samples

for samp in $srcdir/*.go
do
  exe=`basename $samp .go`
  echo "Building program: $exe"
  go build -o bin/$exe $samp
done

echo "Building program: mqitest"
go build -o bin/mqitest $srcdir/mqitest/mqitest.go

