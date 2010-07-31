package main

/*
 * TODO:
 * ; The path handling could take some extra work
 *    - the current setup is a quick hack from Hades.
 * ; Change the default chunk reader size from 1 megabyte.
 * ; Parallellize the code. We can use a pool of workers to carry out hashing for us.
 */

import (
	"container/list"
	"crypto/sha1"
	"bufio"
	"fmt"
	"flag"
	"os"
	"path"
)

var (
	comment         = flag.String("c", "", "Comment to use (optional")
	announceUrl     = flag.String("a", "", "Announce URL to use")
	torrentFilename = flag.String("t", "", "Name of the torrent file")
	pieceSize       = flag.Uint("p", defaultPieceSize,
		"Piece size to use for creating the torrent file (Kilobytes)")
	fileList        = list.New()
	filesVisited    = 0
	multiFile       = false
)

const (
	version          = "0.2"
	worker_count     = 2
	sizeMultiplier   = 1024 // Default value to multiply the size argument by
	defaultPieceSize = 1024 // Default value of the piece size of none is given
)

/* The bufferchunk defines a block for SHA1 checksum encoding. */
type bufferChunk struct {
	p []byte
}

/* The fileBlock defines the information about a new file */
type fileBlock struct {
	path string
	size int64
}

type torrentCreator struct {
	pCh chan *bufferChunk
	pDone chan string
}

func (tc *torrentCreator) VisitFile(path string, fi *os.FileInfo) {
	filesVisited++
	if filesVisited > 1 {
		multiFile = true
	}

	k := new(fileBlock)
	k.path = path
	k.size = fi.Size

	f, err := os.Open(path, os.O_RDONLY, 0)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fileList.PushBack(k)
	fmt.Printf("Visited file %s at path %s\n", fi.Name, path)
	sha1File(f, tc.pCh)
}

func (tc *torrentCreator) VisitDir(path string, f *os.FileInfo) bool {
	multiFile = true

	fmt.Printf("Visited directory %s at path %s\n", f.Name, path)
	return true
}

func sha1File(f *os.File, c chan *bufferChunk) {
	chunkSize := *pieceSize * sizeMultiplier
	buf := bufio.NewReader(f)

	for {
		p := make([]uint8, chunkSize)
		n, err := buf.Read(p)
		if uint(n) < chunkSize {
			if err == os.EOF {
				m := new(bufferChunk)
				m.p = p[0:n]
				c <- m
				break
			}
		}

		if err != nil {
			panic(err)
		}

		m := new(bufferChunk)
		m.p = p

		c <- m
	}
}


func headerString() string {
	return ("This is go-mkTorrent version " + version + "\n")
}

type msg struct {
	i int
	p []byte
	rslice []uint8
}

func hasher(c chan *bufferChunk, done chan string) {
	hash := sha1.New()
	var pushed uint = 0
	lst := list.New()
	chunkSize := *pieceSize * sizeMultiplier
	remaining := chunkSize
	for x := range c {
		p := x.p
		l := uint(len(p))
		for {
			if l < remaining {
				// Not enough for the next piece
				hash.Write(p)
				pushed += l
				remaining -= l
				break
			} else {
				// Enough for the next piece
				hash.Write(p[0:remaining])
				z := hash.Sum()
				lst.PushBack(z)
				hash.Reset()
				pushed = 0
				remaining = chunkSize
				if remaining == l {
					break // Next piece, please
				}
				p = p[remaining:]
			}
		}
	}

	if pushed > 0 {
		z := hash.Sum()
		lst.PushBack(z)
	}

	r := ""
	for x := range lst.Iter() {
		r += string(x.([]byte))
	}

	done <- r
}

func newVisitor() *torrentCreator {
	v := new(torrentCreator)
	v.pCh = make(chan *bufferChunk, 10)
	v.pDone = make(chan string)

	go hasher(v.pCh, v.pDone)
	return v
}

func main() {
	flag.Parse()
	fmt.Printf(headerString())
	// Flag Testing
	if *announceUrl == "" {
		fmt.Fprintf(os.Stderr, "Must supply an Announce URL parameter (-a).\n")
		os.Exit(1)
	}

	if *torrentFilename == "" {
		fmt.Fprintf(os.Stderr, "Must supply a torrent file name to produce.\n")
		os.Exit(1)
	}

	v := newVisitor()
	errC := make(chan os.Error)
	go func() {
		err := <- errC
		panic(err)
	}()

	// Try to open the torrent file. If this fails, it ensures we fail long before the
	// hashing commences.
	f, err := os.Open(*torrentFilename, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0660)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	args := flag.Args()
	for i := range args {
		path.Walk(args[i], v, errC)
	}

	writeTorrent(v, f)
}

func writeTorrent(v *torrentCreator, f *os.File) {
	close(v.pCh)

	aUrl := BString(*announceUrl)
	pStr := BString(<- v.pDone)

	info := map[string]BCode {
		"pieces" : pStr,
	}

	if multiFile {
		var length int64 = 0
		b_list := make([]BCode, filesVisited)
		i := 0
		for fb := range fileList.Iter() {
			b := fb.(*fileBlock)
			length += b.size
			d := map[string]BCode {
				"length" : BUint(b.size),
				"path"   : BString(b.path),
			}
			b_list[i] = BMap(d)
			i++
		}

		info["files"]  = BList(b_list)
		info["length"] = BUint(length)
	} else {
		b := fileList.Front().Value.(*fileBlock)
		info["length"] = BUint(b.size)
		info["name"]   = BString(b.path)
	}

	m := map[string]BCode {
		"announce" : aUrl,
		"info"     : BMap(info),
	}

	if *comment != "" {
		m["comment"] = BString(*comment)
	}

	w := bufio.NewWriter(f)
	BMap(m).renderBCode(w)
	w.Flush()
}
