# go-mktorrent

This is a fork of go-mktorrent by TODO

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
./go-mktorrent -a https://my-tracker.org/announce.php -t test3.torrent -d The_Inchtabokatables_-_Inchtomanie-1992-CMG -p 16384 -c "hello world" -cb "torrentClient"
```