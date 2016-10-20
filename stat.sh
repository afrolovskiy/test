#!/bin/bash
INTERVAL=1
LOADAVG_FILE=~/loadavg.10000ws.1000rps
NETSTAT_S_FILE=~/netstat.s.10000ws.1000rps
NETSTAT_I_FILE=~/netstat.i.10000ws.1000rps
NICSTAT_FILE=~/nicstat.10000ws.1000rps
while true
do
    uptime >> $LOADAVG_FILE
    echo "$(date '+TIME:%H:%M:%S') $(netstat -s | grep 'segments retransmited')" >> $NETSTAT_S_FILE
    echo "$(date '+TIME:%H:%M:%S') $(netstat -i | grep eth0)" >> $NETSTAT_I_FILE
    echo "$(date '+TIME:%H:%M:%S') $(nicstat)" >> $NICSTAT_FILE
    sleep $INTERVAL
done
