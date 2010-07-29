include $(GOROOT)/src/Make.$(GOARCH)

TARG=go-mktorrent
GOFILES=\
	bcoding.go\
	mktorrent.go\

include $(GOROOT)/src/Make.pkg
