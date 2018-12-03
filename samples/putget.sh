# Run the amqsput and amqsget samples in sequence, extracting the MsgId
# from the PUT operation and using it to retrieve the message in the GET sample

go run amqsput.go DEV.QUEUE.1 QM1 | tee /tmp/putget.out
id=`grep MsgId /tmp/putget.out | cut -d: -f2`

if [ "$id" != "" ]
then
  echo "Getting MsgId" $id
  go run amqsget.go DEV.QUEUE.1 QM1 $id
fi
