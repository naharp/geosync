release:
	GOOS=windows GOARCH=386 go build -ldflags="-s -w"
	7za a geosync-windows.zip geosync.exe
	go build -ldflags="-s -w"
	7za a geosync-linux-amd64.zip geosync