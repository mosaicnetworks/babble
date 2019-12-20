
#!/bin/bash

breakCounter=0
for i in `seq 1 100`
do
    go test -run Rejoin > ~/gossip.logs
    if grep "FAIL" ~/gossip.logs 
    then
	echo "FAIL"
	#((breakCounter++))
        echo 'CHECK LOGS'
        exit
    else
        echo $i "OK"
    fi
done
echo "errors: $breakCounter / 100"
