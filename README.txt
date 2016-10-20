To increase soft limits for app you can use prlimit.
Examples:
1) server run:
prlimit --nofile=20000: ./test-server
1) cient run:
prlimit --nofile=20000: ./test-client