(cd modules/weblogin; go-bindata -nomemcopy -debug -pkg weblogin layout.html assets/ templates/; goimports -w .) ; go install -v ./cmd/slacktest
