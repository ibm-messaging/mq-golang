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

# This simple script builds a Docker container whose purpose is simply
# to compile the libraries and sample programs, and then to copy those
# outputs to a local temporary directory.
# In order to run the sample programs you will still need a copy of
# the MQ runtime libraries in the environment.

GOPATH="/go"

TAG="mq-golang-samples-gobuild"
# Assume repo tags have been created in a sensible order
VER=`git tag -l | sort | tail -1 | sed "s/^v//g"`
if [ -z "$VER" ]
then
  VER="latest"
fi
echo "Building container $TAG:$VER"

# Build a container that has all the pieces needed to compile the Go programs for MQ
docker build --build-arg GOPATH_ARG=$GOPATH -t $TAG:$VER .
rc=$?

if [ $rc -eq 0 ]
then
  # Run the image to do the compilation and extract the files
  # from it into local directories mounted into the container.
  OUTBINDIR=$HOME/tmp/mq-golang-samples/bin
  OUTPKGDIR=$HOME/tmp/mq-golang-samples/pkg
  rm -rf $OUTBINDIR $OUTPKGDIR >/dev/null 2>&1
  mkdir -p $OUTBINDIR $OUTPKGDIR

  # The container will be run as the current user to ensure files
  # written back to the host image are owned by that person instead of root.
  uid=`id -u`
  gid=`id -g`

  # Mount an output directory
  # Delete the container once it's done its job
  docker run --rm \
          --user $uid:$gid \
          -v $OUTBINDIR:$GOPATH/bin \
          -v $OUTPKGDIR:$GOPATH/pkg \
          $TAG:$VER
  echo "Compiled samples should now be in $OUTBINDIR"
fi
