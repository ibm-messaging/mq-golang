# Run the amqspub and amqssub samples in sequence - start the subscriber
# first and in the background. Give it a chance to start. Then run the
# publisher

go run amqssub.go dev/GoTopic QM1 &
sleep 1
go run amqspub.go dev/GoTopic QM1
wait
