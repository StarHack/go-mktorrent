include $(GOROOT)/src/Make.$(GOARCH)

TARG=gomktorrent
GOFILES=\
	bcoding.go\
	mktorrent.go\

include $(GOROOT)/src/Make.cmd

torrenttest: clean-torrent mktorrent-test go-mktorrent-test

clean-torrent:
	rm test.torrent

mktorrent-test:
	/usr/bin/time mktorrent -a 'http://www.foo.com' -l 20 test

go-mktorrent-test:
	/usr/bin/time ./gomktorrent -a 'http://www.foo.com' -t 'out.torrent' test