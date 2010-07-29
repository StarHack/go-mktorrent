include $(GOROOT)/src/Make.$(GOARCH)

TARG=gomktorrent
GOFILES=\
	bcoding.go\
	mktorrent.go\

include $(GOROOT)/src/Make.cmd
