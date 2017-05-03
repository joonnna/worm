
PROGRAM EXTRACTION:
----
This project follows the go code organization as described here: https://golang.org/doc/code.html

1. set GOPATH: export GOATH=~/path/to/gopath
2. mkdir -p $GOPATH/src/github.com/joonnna/worm
3. cd $GOPATH/src/github.com/joonnna/src
4. git clone github.com/joonnna/worm
5. To persist the GOPATH setting add the same line to either your ~/.bashrc or ~/.zshrc

If your downloading from zip and not cloning from github, simply extract
the worm folder and contents into github.com/joonnna/ 

BUILD PROGRAM:
----
1. Launch the build.sh script, all binaries(wormgate, segment, visualizer) will be placed
in the "$GOPATH/bin/" folder. 

RUNNING PROGRAM:
----
(sp is the segment port, wp is the wormgateport)

1. Start wormgates on all nodes by using the ssh-all.sh script
Example: ssh-all.sh "$GOPATH/bin/wormgate" -wp=:16000

2. Start visualizer
Example: $GOPATH/bin/visualize -wp=:16000 -sp=:16050

3. Start segments
Example: $GOPATH/bin/segment -wp=:16000 -sp=:16050 -mode=spread -host=compute-1-1
This will spread the segment to compute-1-1, default mode is run, which will execute
the segment on the current node.

