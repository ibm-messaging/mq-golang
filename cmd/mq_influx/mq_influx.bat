
set qMgr=V90_W


set queues="APP.*,MYQ.*"



# And other parameters that may be needed
# See config.go for all recognised flags
set database=MQDB
set userid=admin 
set password=admin
set passwordFile=c:\temp\mqinfluxpw.txt          
set svr=http://klein.hursley.ibm.com:8086
set interval="10"

set ARGS=-ibmmq.queueManager=%qMgr%
set ARGS=%ARGS% -ibmmq.databaseName=%database%
set ARGS=%ARGS% -ibmmq.databaseAddress=%svr%
set ARGS=%ARGS% -ibmmq.databaseUserID=%userid%
set ARGS=%ARGS% -ibmmq.interval=%interval%
set ARGS=%ARGS% -ibmmq.monitoredQueues=%queues%
set ARGS=%ARGS% -ibmmq.pwFile=%passwordFile%
set ARGS=%ARGS% -log.level=info



del %passwordFile%
echo %password% > %passwordFile%
c:\gowork\bin\mq_influx  %ARGS%
