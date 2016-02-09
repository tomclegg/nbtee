README.md: *.go
	go get github.com/robertkrimen/godocdown/godocdown
	$(GOPATH)/bin/godocdown >README.md~
	mv README.md~ README.md
