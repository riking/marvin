#!/bin/bash
go build -v github.com/riking/marvin/cmd/slacktest 2>&1
if [ "$?" != 0 ]; then
    echo "Failed to compile."
    exit 127
fi
mv slacktest $HOME/go/bin/ 2>&1