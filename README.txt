To increase soft limits for app you can use prlimit.
Examples:
1) server run:
prlimit --nofile=20000: ./test-server
1) cient run:
prlimit --nofile=20000: ./test-client

Monitoring:
CPU monitoring
vmstat 1

Memory monitoring
vmstat 1
dmesg (call at the end of script work)

Network interface monitoring
sar -n DEV 1
sar -n EDEV 1
netstat -i
netstat -s | grep 'segments retransmited'
nicstat

Run monitor test-server:
source stat.sh
vmstat 1 > ~/vmstat.10000ws.1000rps
sar -n DEV 1 > ~/sar.dev.10000ws.1000rps
sar -n EDEV 1 > ~/sar.dev.10000ws.1000rps
