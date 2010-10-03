include $(GOROOT)/src/Make.inc

PKGDIR=$(GOROOT)/pkg/$(GOOS)_$(GOARCH)

TARG=zmq
CGOFILES=zmq.go
CGO_CFLAGS=-I. -I"$(GOROOT)/include" -I"../coffer" -I/usr/local/include
CGO_LDFLAGS=-lzmq
GOFMT=$(GOROOT)/bin/gofmt -tabwidth=4 -spaces=true -tabindent=false -w 

include $(GOROOT)/src/Make.pkg

# include $(GOROOT)/src/pkg/goprotobuf.googlecode.com/hg/Make.protobuf

CLEANFILES+=clsrv $(PKGDIR)/$(TARG).a

again: clean install clsrv

clsrv: install clsrv.go
	$(GC) clsrv.go
	$(LD) -o $@ clsrv.$O

format: 
	$(GOFMT) zmq.go
	$(GOFMT) clsrv.go
