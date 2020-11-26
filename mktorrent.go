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
	//"bytes"
	"bufio"
	"fmt"
	"flag"
	"io"
	"path"
	"path/filepath"
	"os"
	"strings"
	//"path/filepath"
	bencode "github.com/jackpal/bencode-go"
)

var (
	comment         = flag.String("c", "", "Comment to use (optional")
	announceUrl     = flag.String("a", "", "Announce URL to use")
	torrentFilename = flag.String("t", "", "Name of the torrent file")
	directory       = flag.String("d", "", "Source directory")
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
	path          [1]string
}

func NewFile(path string, length int64) File {
	var file File
	file.length = length
	file.path[0] = path
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

	//info["files"] = make(map[string]interface{})
	//files, _ := info["files"].(map[string]interface{})
	var files []File
	info["files"] = files
	//files, _ := info["files"].(map[string]interface{})

	return tf, info, files
}
/*
func HashFile(name string, chunksize int) []byte {
	data, err := os.Open(name)
	if err != nil {
		// log.Fatal("Error Reading ", filename, ": ", err)	
	}
	defer data.Close()

	reader := bufio.NewReader(data)
	// buffer := bytes.NewBuffer(make([]byte, 0))
	buffer := make([]byte, chunksize)

	hash := sha1.New()
	var ret []byte
	var sum []byte

	for {
		read, err := reader.Read(buffer);
		if err != nil {
			break;
		}

		hash.Write(buffer[0:read])
		// hash.Write(buffer[0:read]) // only for testing multi file hashes
		//str := fmt.Sprintf("% X", hash.Sum(nil))
		sum = hash.Sum(nil)
		ret = append(ret[:], sum[:]...)
		hash.Reset()
		//ret += str
	}

	// fmt.Println(part)

	if err != io.EOF {
		/// log.Fatal("Error Reading ", filename, ": ", err)	
	} else {
		err = nil
	}

	fmt.Println(ret)

	return ret
}*/

func visit(files* []File) filepath.WalkFunc {
	return func(thepath string, info os.FileInfo, err error) error {
			if info.IsDir() {
				// filepath.Walk(path.Join(*directory, info.Name()), visit(files))
				return nil
			}

			if strings.HasPrefix(info.Name(), ".DS_Store") {
				return nil
			}

			//fmt.Println()
			
			//filename := info.Name()
			filename := thepath[len(*directory) + 1:]

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
		filepath := path.Join(*directory, file.path[0])
		//filepath := path.Join("", file.path[0])
		data, err := os.Open(filepath)
		if err != nil {
			fmt.Println("Error Reading ", filepath, ": ", err)	
		}
		defer data.Close()

		reader := bufio.NewReader(data)
		// buffer := bytes.NewBuffer(make([]byte, 0))
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
				// ret = append(ret[:], sum[:]...)
				//hash.Reset()

				// Set pieceBytes to new value
				pieceBytes = exceedingBytes
			} else {

				hash.Write(buffer[0:read])
				sum = hash.Sum(nil)
				// ret = append(ret[:], sum[:]...)
				// hash.Reset()
			}

			// hash.Write(buffer[0:read]) // only for testing multi file hashes
			//str := fmt.Sprintf("% X", hash.Sum(nil))
			
			

			//ret += str
		}

		// fmt.Println(part)

		if err != io.EOF {
			/// log.Fatal("Error Reading ", filename, ": ", err)	
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



	trfile, _ := os.Open("Just_Dance_2021_Update_v322043.561500_NSW-NiiNTENDO.torrent")
	reader := bufio.NewReader(trfile)
	//var data interface{}
	data, _ := bencode.Decode(reader)



	chunksize := 16384 // 16777216

	torrentFile, info, files := NewTorrentFile()
	torrentFile["announce"] = "https://tntracker.org/announce.php"
	torrentFile["created by"] = "StarHack"
	torrentFile["creation date"] = 1605292896
	// torrentFile.info["info"] = TODO
	info["name"] = *directory
	info["piece length"] = chunksize

	// Add all files in source directory
	err := filepath.Walk(*directory, visit(&files))
	/*
	err := filepath.Walk(*directory, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		files = AddFileToList(files, NewFile(info.Name(), info.Size()))
		return nil
	})
	*/

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

	// log(torrentFile)

	// err := bencode.Unmarshal(reader, &data)

	data2, _ := data.(map[string]interface{})

	fmt.Printf("%+v\n", data2["announce"])
	return
}
