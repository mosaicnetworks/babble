#!/bin/bash

xterm -geometry 50x10+100+100 -bg blue -xrm 'XTerm.vt100.allowTitleOps: false' -T 'CLIENT 1 INPUT'  -e 'docker attach client1' & 
xterm -geometry 50x10+420+100 -bg blue -xrm 'XTerm.vt100.allowTitleOps: false' -T 'CLIENT 1 MESSAGES' -e "docker exec client1 watch -n 1 cat messages.txt" &

xterm -geometry 50x10+100+280 -bg blue -xrm 'XTerm.vt100.allowTitleOps: false' -T 'CLIENT 2 INPUT' -e "docker attach client2" &
xterm -geometry 50x10+420+280 -bg blue -xrm 'XTerm.vt100.allowTitleOps: false' -T 'CLIENT 2 MESSAGES' -e "docker exec client2 watch -n 1 cat messages.txt" &  
