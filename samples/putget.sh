# Run the amqsput and amqsget samples in sequence, extracting the MsgId
# from the PUT operation and using it to retrieve the message in the GET sample

# We don't get to see the output from the amqsput program as it's filtered to
# extract the MsgId
id=`go run amqsput.go DEV.QUEUE.1 QM1 | grep MsgId | cut -d: -f2`

if [ "$id" != "" ]
then
  echo "Getting MsgId" $id
  go run amqsget.go DEV.QUEUE.1 QM1 $id
fi
