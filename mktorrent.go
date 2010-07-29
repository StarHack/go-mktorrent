package main

import (
	"bufio"
	"fmt"
	"flag"
	"os"
	"crypto/sha1"
	"strings"
)

var (
	announceUrl     = flag.String("a", "", "Announce URL to use")
	fileName        = flag.String("f", "", "File name to process")
	torrentFilename = flag.String("t", "", "Name of the torrent file")
	pieceSize       = flag.Uint("p", defaultPieceSize,
		"Piece size to use for creating the torrent file (Kilobytes)")
)

const (
	version          = "0.1"
	sizeMultiplier   = 1024 // Default value to multiply the size argument by
	defaultPieceSize = 1024 // Default value of the piece size of none is given
)

func headerString() string {
	return ("This is go-mkTorrent version " + version + "\n")
}

func baseName(name string) string {
	res := strings.TrimRightFunc(name, func(c int) bool { return c != '.' })
	if res == "" {
		res = name
	}

	return res
}

func mkPieceString(fName string, sz uint) (r string, l int64) {
	f, err := os.Open(fName, os.O_RDONLY, 0)
	if err != nil {
		panic(err)
	}

	s, err2 := f.Stat()
	if err2 != nil {
		panic(err2)
	}

	l = s.Size
	toRead := sz * sizeMultiplier

	c := make(chan []byte, 3)
	res := make(chan []byte, 0)

	go func() {
		hash := sha1.New()
		for x := range c {
			hash.Write(x)
			res <- hash.Sum()
			hash.Reset()
		}

		close(res)
	}()

	for {
		p := make([]byte, toRead)
		n, err := f.Read(p)
		if n == 0 {
			if err == os.EOF {
				close(c)
				break
			}
		}

		if err != nil {
			panic(err)
		}

		if uint(n) == toRead {
			panic("We are not reading the right amount of bytes from the underlying file")
		}

		c <- p
	}

	r = ""
	for x := range res {
		r += string(x) // TODO: Fix this, it is utterly wrong to convert the []byte to unicode runes,
		// which I think is what happens here.
	}

	return r, l
}

func mkTorrentFile(aUrl string, pStr string, fName string, fSz uint, sz uint, outName string) {
	b_aUrl := BString(aUrl)
	b_fName := BString(fName)
	b_pStr := BString(pStr)
	b_sz := BUint(sz)
	b_f_sz := BUint(sz)

	b_info := map[string]BCode{
		"name":         b_fName,
		"piece length": b_sz,
		"pieces":       b_pStr,
		"length":       b_f_sz,
	}

	m := map[string]BCode{
		"announce": b_aUrl,
		"info":     BMap(b_info),
	}

	f, err := os.Open(outName, os.O_WRONLY|os.O_CREAT|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}

	w := bufio.NewWriter(f)
	BMap(m).renderBCode(w)
	w.Flush()
	f.Close()
}

func main() {
	flag.Parse()
	fmt.Printf(headerString())
	// Flag Testing
	if *announceUrl == "" {
		fmt.Fprintf(os.Stderr, "Must supply an Announce URL parameter (-a)\n")
		os.Exit(1)
	}

	if *fileName == "" {
		fmt.Fprintf(os.Stderr, "Must supply a file name to process (-f)\n")
		os.Exit(1)
	}

	if *torrentFilename == "" {
		// Make a default filename
		*torrentFilename = baseName(*fileName) + ".torrent"
	}

	pieceString, fSize := mkPieceString(*fileName, *pieceSize)
	mkTorrentFile(*announceUrl, pieceString, *fileName, uint(fSize), *pieceSize, *torrentFilename)
}
