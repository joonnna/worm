#!/bin/bash

for GO in ./*/
do
    #echo $GO
    go install $GO
done
