#!/bin/bash
LOADAVG_FILE=~/loadagv.10000ws.1000rps
NETSTAT_FILE=~/netstat.10000ws.1000rps
while true
do
    # Log load average
    echo "$(date '+TIME:%H:%M:%S') $(uptime)" >> $LOADAVG_FILE
    # Log network stat
    echo "$(date '+TIME:%H:%M:%S') $(netstat -i | grep eth0)" >> $NETSTAT_FILE
    sleep 1
done
