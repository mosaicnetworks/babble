#!/bin/bash

awkcmd="awk -F\"'\" '{print $4}'"

xterm -geometry 50x10+100+100 -fg white -bg blue -xrm 'XTerm.vt100.allowTitleOps: false' -T 'CLIENT 1 INPUT'  -e 'docker attach client1' &
xterm -geometry 50x10+420+100 -fg white -bg blue -xrm 'XTerm.vt100.allowTitleOps: false' -T 'CLIENT 1 MESSAGES' -e 'docker exec client1 tail -f info.log | awk -F'"'"'"'"'"' '"'"'{print $4}'"'"' '  &

xterm -geometry 50x10+100+280 -fg white -bg blue -xrm 'XTerm.vt100.allowTitleOps: false' -T 'CLIENT 2 INPUT' -e "docker attach client2" &
xterm -geometry 50x10+420+280 -fg white -bg blue -xrm 'XTerm.vt100.allowTitleOps: false' -T 'CLIENT 2 MESSAGES' -e 'docker exec client1 tail -f info.log | awk -F'"'"'"'"'"' '"'"'{print $4}'"'"' '  &
