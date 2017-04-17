#!/bin/bash
for GO in *.go
do
    go build $GO
done
