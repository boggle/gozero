include $(GOROOT)/src/Make.$(GOARCH)

PKGDIR=$(GOROOT)/pkg/$(GOOS)_$(GOARCH)

TARG=zmq
CGOFILES=utils.go zmq.go
CGO_CFLAGS=-I. -I "$(GOROOT)/include"
CGO_LDFLAGS=-lzmq
GOFMT=$(GOROOT)/bin/gofmt -tabwidth=2 -spaces=true -tabindent=false -w 

include $(GOROOT)/src/Make.pkg

# include $(GOROOT)/src/pkg/goprotobuf.googlecode.com/hg/Make.protobuf

CLEANFILES+=clsrv $(PKGDIR)/$(TARG).a

again: clean install clsrv

clsrv: install clsrv.go
	$(GC) clsrv.go
	$(LD) -o $@ clsrv.$O

format: 
	$(GOFMT) utils.go
	$(GOFMT) zmq.go
	$(GOFMT) clsrv.go
