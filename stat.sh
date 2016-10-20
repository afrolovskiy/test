#!/bin/bash
INTERVAL=1
LOADAVG_FILE=~/loadavg.10000ws.1000rps
NETSTAT_FILE=~/netstat.10000ws.1000rps
VMSTAT_FILE=~/vmstat.10000ws.1000rps
while true
do
    uptime >> $LOADAVG_FILE
    echo "$(date '+TIME:%H:%M:%S') $(vmstat | tail -n 1)" >> $VMSTAT_FILE
    echo "$(date '+TIME:%H:%M:%S') $(netstat -i | grep eth0)" >> $NETSTAT_FILE
    sleep $INTERVAL
done
