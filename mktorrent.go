package main

/*
 * TODO:
 * ; The path handling could take some extra work
 *    - the current setup is a quick hack from Hades.
 * ; Eliminate the copying that goes on in the default case. If a complete slice can
 *   be stolen right off of the array, do that rather than copying.
 */

import (
	"container/list"
	"crypto/sha1"
	"bufio"
	"fmt"
	"flag"
	"io"
	"os"
	"path/filepath"
)

var (
	comment         = flag.String("c", "", "Comment to use (optional")
	announceUrl     = flag.String("a", "", "Announce URL to use")
	torrentFilename = flag.String("t", "", "Name of the torrent file")
	pieceSize       = flag.Uint("p", defaultPieceSize,
		"Piece size to use for creating the torrent file (Kilobytes)")
	fileList     = list.New()
	piecesMap    = make(map[uint][]byte, 400)
	filesVisited = 0
	multiFile    = false
)

const (
	version          = "0.2"
	workerCount      = 2
	sizeMultiplier   = 1024 // Default value to multiply the size argument by
	defaultPieceSize = 1024 // Default value of the piece size of none is given
)

type bufferChunk struct {
	chunk []byte
	n uint
}

/* The fileBlock defines the information about a new file */
type fileBlock struct {
	path string
	size int64
}

type torrentCreator struct {
	pCh   chan []byte
	pDone chan []byte
	hCh   chan *bufferChunk
}

func (tc *torrentCreator) VisitFile(path string, fi *os.FileInfo) {
	filesVisited++
	if filesVisited > 1 {
		multiFile = true
	}

	k := new(fileBlock)
	k.path = path
	k.size = (*fi).Size()

	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fileList.PushBack(k)
	fmt.Printf("Visited file %s at path %s\n", (*fi).Name(), path)
	sha1File(f, tc.pCh)
}

func (tc *torrentCreator) VisitDir(path string, f *os.FileInfo) bool {
	multiFile = true

	fmt.Printf("Visited directory %s at path %s\n", (*f).Name(), path)
	return true
}

func sha1File(f *os.File, c chan []byte) {
	chunkSize := uint(4096 * 1024)
	buf := bufio.NewReader(f)

	for {
		p := make([]uint8, chunkSize)
		n, err := buf.Read(p)
		if uint(n) < chunkSize {
			if err == io.EOF {
				c <- p[0:n]
				break
			}
		}

		if err != nil {
			panic(err)
		}

		c <- p
	}
}


func headerString() string {
	return ("This is go-mkTorrent version " + version + "\n")
}

func pieceAdder(res chan []byte) chan *bufferChunk {
	fmt.Printf("Adding a piece\n")
	c := make(chan *bufferChunk, 3)
	go func() {
		i := 0
		for m := range c {
			piecesMap[m.n] = m.chunk
			i++
		}

		r := make([]byte, i*20)
		for k, v:= range piecesMap {
			copy(r[k*20:k*20+20], v)
		}

		res <- r
	}()

	return c
}

func hashWorker(in chan *bufferChunk, out chan *bufferChunk, done chan bool) {
	hash := sha1.New()
	for m := range in {
		hash.Write(m.chunk)
		m.chunk = hash.Sum(m.chunk) // TODO?
		out <- m
		hash.Reset()
	}

	done <- true
}

func hasher(c chan []byte, hashC chan *bufferChunk, res chan []byte) {

	fmt.Printf("Hashing a piece\n")

	out := pieceAdder(res)
	workerDone := make([]chan bool, workerCount)
	for i := range workerDone {
		workerDone[i] = make(chan bool)
		go hashWorker(hashC, out, workerDone[i])
	}

	chunkSize := *pieceSize * sizeMultiplier
	point := uint(0)
	chunk := make([]byte, chunkSize)

	n := uint(0)
	for x := range c {
		for {
			if chunk == nil {
				chunk = make([]byte, chunkSize)
			}

			l := uint(len(x))
			remaining := chunkSize - point
			if l < remaining {
				// Not enough for the next piece
				copy(chunk[point:(point+l)], x)
				point += l
				break
			} else {
				// Enough for the next piece
				copy(chunk[point:(point+remaining)], x[0:remaining])
				bc := new(bufferChunk)
				bc.chunk = chunk
				bc.n = n
				n++
				hashC <- bc
				chunk = nil
				point = 0
				if l == remaining {
					break
				} else {
					x = x[remaining:]
				}
			}
		}
	}

	if chunk != nil {
		bc := new(bufferChunk)
		bc.chunk = chunk[0:point]
		bc.n = n
		n++
		hashC <- bc
	}

	close(hashC)

	for i := range workerDone {
		<-workerDone[i]
	}
	close(out)
}

func newVisitor() (*torrentCreator, chan []byte) {
	v := new(torrentCreator)
	v.pCh = make(chan []byte, 3)
	v.hCh = make(chan *bufferChunk, 3)

	res := make(chan []byte)
	go hasher(v.pCh, v.hCh, res)
	return v, res
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
		fmt.Fprintf(os.Stderr, "Must supply a torrent file name to produce (-t).\n")
		os.Exit(1)
	}

	v, resC := newVisitor()
	errC := make(chan error)
	go func() {
		err := <-errC
		panic(err)
	}()

	// Try to open the torrent file. If this fails, it ensures we fail long before the
	// hashing commences.
	//f, err := os.Open(*torrentFilename, os.O_WRONLY|os.O_CREAT|os.O_TRUNC, 0660) // TODO
	f, err := os.Create(*torrentFilename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	args := flag.Args()
	for i := range args {
		/// path.Walk(args[i], v, errC)      // TODO?
		filepath.Walk(args[i], func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			} else {
				v.VisitFile(info.Name(), &info)
			}
			
			return nil
		})
	}

	writeTorrent(v, f, resC)
}

func writeTorrent(v *torrentCreator, f *os.File, res chan []byte) {
	close(v.pCh)

	aUrl := BString(*announceUrl)
	pStr := BBytes(<-res)

	chunkSize := BUint(*pieceSize * sizeMultiplier)
	info := map[string]BCode{
		"pieces" : pStr,
		"piece length" : chunkSize,
	}

	fmt.Printf("Pieces: %s\n", pStr)



	if multiFile {
		var length int64 = 0
		b_list := make([]BCode, filesVisited)
		i := 0
		// for fb := range fileList.Iter() { // TODO?
		for fb := fileList.Front(); fb != nil; fb = fb.Next() {
			b := fb.Value.(*fileBlock)
			length += b.size
			d := map[string]BCode{
				"length": BUint(b.size),
				"path":   BString(b.path),
			}
			b_list[i] = BMap(d)
			i++
		}

		info["files"] = BList(b_list)
		info["length"] = BUint(length)
	} else {
		b := fileList.Front().Value.(*fileBlock)
		info["length"] = BUint(b.size)
		info["name"] = BString(b.path)
	}

	m := map[string]BCode{
		"announce": aUrl,
		"info":     BMap(info),
	}

	if *comment != "" {
		m["comment"] = BString(*comment)
	}

	w := bufio.NewWriter(f)
	BMap(m).renderBCode(w)
	w.Flush()
}
