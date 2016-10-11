#!/bin/bash
while true
do
    # Log load average
    echo "$(date '+TIME:%H:%M:%S') $(uptime)" | tee -a ~/loadagv
    # Log network stat
    echo "$(date '+TIME:%H:%M:%S') $(netstat -i | grep eth0)" | tee -a ~/netstat
    sleep 1
done
