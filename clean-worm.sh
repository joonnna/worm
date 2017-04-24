#!/bin/bash
USER=$(whoami)
set -x
./ssh-all.sh killall -9 -u $USER
./ssh-all.sh sudo rm -r /tmp/wormgate-$USER
