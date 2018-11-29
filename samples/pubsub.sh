# Run the amqspub and amqssub samples in sequence - start the subscriber
# first and in the background. Give it a chance to start. Then run the
# publisher

go run amqssub.go GO.TEST.TOPIC QM1 &
sleep 1
go run amqspub.go GO.TEST.TOPIC QM1
wait
