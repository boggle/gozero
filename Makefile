include $(GOROOT)/src/Make.$(GOARCH)

PKGDIR=$(GOROOT)/pkg/$(GOOS)_$(GOARCH)

TARG=zmq
CGOFILES=utils.go zmq.go
CGO_CFLAGS=-I. -I "$(GOROOT)/include"
CGO_LDFLAGS=-lzmq
GOFMT=$(GOROOT)/bin/gofmt -tabwidth=2 -spaces=true -tabindent=false -w 

include $(GOROOT)/src/Make.pkg

# include $(GOROOT)/src/pkg/goprotobuf.googlecode.com/hg/Make.protobuf

CLEANFILES+=main $(PKGDIR)/$(TARG).a

main: install main.go
	$(GC) main.go
	$(LD) -o $@ main.$O

format: 
	$(GOFMT) utils.go
	$(GOFMT) zmq.go
	$(GOFMT) main.go
