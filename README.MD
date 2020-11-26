# go-mktorrent

This is a fork of [go-mktorrent](https://github.com/0xD3ADB33F/go-mktorrent) by **0xD3ADB33F**

Unfortunately I changed quite a lot compared to the original codebase (sorry). We now rely on the [bencode-go](https://github.com/jackpal/bencode-go) package by **jackpal**.

## Features

- Create a torrent file from source directories (`-d` parameter)
- Supports multiple files
- Supports sub directories
- Filters `.DS_Store` files (macOS)

## Building

```
go get .
go build .
```

## Usage

### Supported Command Line Parameters

```
 -a https://my-tracker.org/announce.php
 -d My_Release_Directory-GROUP
 -t output.torrent
 -p 16384
 -c "My torrent comment"
-cb "MyTorrentGenerator"
```


```
./go-mktorrent -a https://my-tracker.org/announce.php -t test3.torrent -d My_Release_Directory-GROUP -p 16384 -c "hello world" -cb "torrentClient"
```