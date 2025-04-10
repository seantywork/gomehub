#!/bin/bash



if ! docker network ls | grep -q gomehubnet0
then
    docker network create --driver=bridge gomehubnet0
fi


if ! docker images | grep -q localhost/gomehub
then 
    docker build -t gomehub:latest .
fi


docker run --rm \
    --name gomehub --network gomehubnet0 \
    -p 8000:8000 \
    -p 8002:8002 \
    -p 8004:8004 \
    -p 8006:8006/udp \
    -p 31000-31100:31000-31100/udp \
    localhost/gomehub /bin/bash -c 'sleep 3 && ./gomehub.out'
