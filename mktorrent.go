package main

import (
	"container/list"
	"crypto/sha1"
	"bufio"
	"fmt"
	"flag"
	"io"
	"path"
	"path/filepath"
	"os"
	"strings"
	bencode "github.com/jackpal/bencode-go"
)

var (
	comment         = flag.String("c", "", "Comment to use (optional")
	announceUrl     = flag.String("a", "", "Announce URL to use")
	torrentFilename = flag.String("t", "", "Name of the torrent file")
	directory       = flag.String("d", "", "Source directory")
	createdBy       = flag.String("cb", "StarHack", "Created by")
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

type TorrentFile struct {
	announce   		string
	created_by 		string
	creation_data int64
	info          map[string]interface{}
}

type File struct {
	length 				int64
	path          []string
}

func NewFile(path string, length int64) File {
	var file File
	file.length = length

	if strings.Contains(path, "/") {
		file.path = strings.Split(path, "/")
	} else {
		file.path = []string{path}
	}

	return file
}

func AddFileToList(files* []File, file File) {
	target := make([]File, len(*files)+1)
	copy(target, *files)
	target[len(*files)] = file
	*files = target
}

func NewTorrentFile() (map[string]interface{}, map[string]interface{}, []File) {
	var tf = make(map[string]interface{})
	
	tf["info"] = make(map[string]interface{})
	info, _ := tf["info"].(map[string]interface{})

	var files []File
	info["files"] = files

	return tf, info, files
}

func visit(files* []File) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
			// Ignore directories (we only need to hash files) and .DS_Store files (macOS)
			if info.IsDir() || strings.HasPrefix(info.Name(), ".DS_Store") {
				return nil
			}

			filename := path[len(*directory) + 1:]

			AddFileToList(files, NewFile(filename, info.Size()))
			return nil
	}
}

func HashFiles(files []File, chunksize int) []byte {
	hash := sha1.New()
	var ret []byte
	var sum []byte

	pieceBytes := 0

	for _, file := range files {
		filepath := path.Join(*directory, "")
		for _, subpath := range file.path {
			filepath = path.Join(filepath, subpath)
		}

		data, err := os.Open(filepath)
		if err != nil {
			fmt.Println("Error Reading ", filepath, ": ", err)	
		}
		defer data.Close()

		reader := bufio.NewReader(data)
		buffer := make([]byte, chunksize)

		for {
			read, err := reader.Read(buffer);
			
			if err != nil {
				break;
			}

			pieceBytes = pieceBytes + read

			if pieceBytes > chunksize {
				exceedingBytes := pieceBytes - chunksize
				remainingPieceBytes := chunksize - (pieceBytes - read)

				// Write remaining bytes to fill up piece
				hash.Write(buffer[0:remainingPieceBytes])
				sum = hash.Sum(nil)
				ret = append(ret[:], sum[:]...)
				hash.Reset()

				// Write exceeding bytes into new piece
				hash.Write(buffer[remainingPieceBytes:])
				sum = hash.Sum(nil)

				// Set pieceBytes to new value
				pieceBytes = exceedingBytes
			} else {
				hash.Write(buffer[0:read])
				sum = hash.Sum(nil)
			}
		}


		if err != io.EOF {
			// log.Fatal("Error Reading " + filename + ": " + err)	
		} else {
			err = nil
		}

	}

	ret = append(ret[:], sum[:]...)

	return ret
}

func headerString() string {
	return "This is go-mkTorrent version " + version + "\n"
}

func log(obj interface{}) {
  fmt.Printf("%+v\n", obj)
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

	if *directory == "" {
		fmt.Fprintf(os.Stderr, "Must supply a source directory (-d).\n")
		os.Exit(1)
	}

	if *pieceSize % 2 != 0 || *pieceSize < 32 || *pieceSize > 16384 {
		fmt.Fprintf(os.Stderr, "Piece size must be a power of two and be between 32 KB and 16 MiB (16384 KB).\n")
		os.Exit(1)
	}

	// chunksize := 16384 // 16777216
	chunksize := int(*pieceSize * sizeMultiplier)

	torrentFile, info, files := NewTorrentFile()
	torrentFile["announce"] = *announceUrl
	torrentFile["created by"] = *createdBy
	torrentFile["creation date"] = 1605292896
	torrentFile["comment"] = *comment
	info["name"] = *directory
	info["piece length"] = chunksize

	// Add all files in source directory
	err := filepath.Walk(*directory, visit(&files))

	if err != nil {
		panic(err)
	}


	info["files"] = files

	tst := HashFiles(files, chunksize)

	info["pieces"] = tst
	info["private"] = 1

	writeFile, _ := os.Create("test2.torrent")
	writer := bufio.NewWriter(writeFile)
	bencode.Marshal(writer, torrentFile)
	writer.Flush()

	return
}
