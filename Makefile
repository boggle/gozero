include $(GOROOT)/src/Make.$(GOARCH)

PKGDIR=$(GOROOT)/pkg/$(GOOS)_$(GOARCH)

TARG=gozero
CGOFILES=gozero.go
CGO_LDFLAGS=-lzmq

include $(GOROOT)/src/Make.pkg

CLEANFILES+=main $(PKGDIR)/$(TARG).a

main: install main.go
	$(GC) main.go
	$(LD) -o $@ main.$O