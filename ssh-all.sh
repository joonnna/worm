#!/bin/bash
for COMPUTE in $(rocks list host compute | cut -d : -f1 | grep -v HOST)
do
    ssh -f $COMPUTE $* \
        1> >(sed "s/^/$COMPUTE: /") \
        2> >(sed "s/^/$COMPUTE: /" 1>&2)
done
