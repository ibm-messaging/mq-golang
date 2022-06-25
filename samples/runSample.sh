# This is an example of running one of the sample programs in a container.

function latestSemVer {
  (for x in $*
  do
    echo $x | sed "s/^v//g"
  done) | sort -n | tail -1
}

# Assume repo tags have been created in a sensible order. Find the mq-golang
# version in the root go.mod file and the current Git tag for this repo.
# Then pick the latest version to create the Docker tag
for m in ../go.mod ../../go.mod
do
  if [ -r $m ]
  then
   VERDEP=`cat $m | awk '/mq-golang/ {print $2}' `
  fi
done
VERREPO=`git tag -l 2>/dev/null| sort | tail -1 `

VER=`latestSemVer $VERDEP $VERREPO`
if [ -z "$VER" ]
then
  VER="latest"
fi

# Basic name for the container
TAG=mq-golang-sample-amqsput
echo Building container with tag $TAG:$VER

# Build the container which includes compilation of the program. Can set the FROM
# environment variable outside this script to choose which base image to work from.
# The UBI variant is built on the Red Hat Universal Base Image set of containers; the
# default is built on Ubuntu/Debian containers.
if [ "$FROM" = "UBI" ]
then
  dfile="runSample.ubi.Dockerfile"
else
  dfile="runSample.deb.Dockerfile"
fi

# Setting NOCACHE environment variable will force a complete rebuild of the container
# which might be useful for some testing
if [ ! -z "$NOCACHE" ]
then
  nocache="--no-cache"
else
  nocache=""
fi
docker build $nocache -t $TAG:$VER -f  $dfile .

if [ $? -eq 0 ]
then
  # This line grabs a currently active IPv4 address for this machine. It's probably
  # not what you want to use for a real system but it's useful for testing. "localhost"
  # does not necessarily work inside the container so we need a real address.
  addr=`ip -4 addr | grep "state UP" -A2 | grep inet | tail -n1 | awk '{print $2}' | cut -f1 -d'/'`
  echo "Local address is $addr"
  port="1414"

  if [ ! -z "addr" ]
  then
    # Run the container. Can override default command line values in amqsput via
    # env vars here.
    docker run -e MQSERVER="SYSTEM.DEF.SVRCONN/TCP/$addr($port)" \
       -e QUEUE=DEV.QUEUE.1 \
       -e QMGR=QM1 \
       $TAG:$VER
  else
    echo "Cannot find a working address for this system"
    exit 1
  fi
fi
