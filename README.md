GeoSync
=======

GeoSync is a small commandline application to keep coordinates of places provided in a csv file in sync with OpenStreetMap. 

It uploads a place to OSM and saves returned IDs to a local journal file for tracking modifications. A synced entry can be deleted by setting the `latlng` field empty. 

Download: [Windows Build](https://github.com/naharp/geosync/releases/download/v0.0.1/geosync-windows.zip) &bull;
[Linux Build (64bit)](https://github.com/naharp/geosync/releases/download/v0.0.1/geosync-linux-amd64.zip)
## Usage ##
```bash
$ geosync -h
Usage: geosync [OPTIONS] csv_file
  -c string
        path to configuration file (default: $HOME/.geosync.json)
  -j string
        path to jounal file (default: csv_file.log)
  -q    don't prompt for inputs
  
```  

## Building ##

Any system with Golang 1.13+ can build this package 
```
go 
```

