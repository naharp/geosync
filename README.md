GeoSync
=======

GeoSync is a small commandline application to keep coordinates of places provided in a csv file in sync with OpenStreetMap. 

It uploads a place to OSM and saves returned IDs to a local journal file for tracking modifications. A synced entry can be deleted by setting the `latlng` field empty. 

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


