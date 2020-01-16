package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/naharp/geosync/osmapi"
	"log"
	"os"
	"path"
	"strings"
)

type Configuration struct {
	OSM_User string
	OSM_Pass string
	OSM_Host string
}

type LogRecord struct {
	UniqueID string
	Name     string
	Amenity  string
	Building string
	LatLng   string
	OsmId    string
}

const (
	UID = iota
	NAME
	AMENITY
	BUILDING
	LATLNG
)

var config Configuration
var quiet bool
var inserts, updates, deletions []LogRecord
var journal map[string]LogRecord
var jounalFile, cfgFile string

func dieIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func loadJson(jsonfp string, obj interface{}, fatalError bool) error {
	file, _ := os.Open(jsonfp)
	defer file.Close()
	decoder := json.NewDecoder(file)
	err := decoder.Decode(obj)
	if err != nil && fatalError {
		log.Fatalf("error loading %s: %v", jsonfp, err)
	}
	return err
}

func saveJson(jsonfp string, obj interface{}, fatalError bool) error {
	file, _ := os.Create(jsonfp)
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "\t")
	err := encoder.Encode(obj)
	if err != nil && fatalError {
		log.Fatalf("error saving %s: %v", jsonfp, err)
	}
	return err
}

func loadCfg() {
	if cfgFile == "" {
		home, _ := os.UserHomeDir()
		cfgFile = path.Join(home, ".geosync.json")
	}

	if _, err := os.Stat(cfgFile); err == nil {
		loadJson(cfgFile, &config, true)
		fmt.Println("Using OSM Username: ", config.OSM_User)
	} else if !quiet {
		fmt.Print("OSM Username : ")
		fmt.Scanln(&config.OSM_User)
		fmt.Print("OSM Password : ")
		fmt.Scanln(&config.OSM_Pass)
		fmt.Print("OSM API Url (press enter for default OSM API) : ")
		fmt.Scanln(&config.OSM_Host)
		if config.OSM_Host == ""{
			config.OSM_Host = "https://api.openstreetmap.org/"
		}
		saveJson(cfgFile, &config, true)
	} else {
		log.Fatal("No config found")
	}
}

func newChangeSet(req *osmapi.MyRequestSt, name string) *osmapi.ChangeSetSt {
	cset, err := req.Changesets(name)
	if err != nil {
		log.Fatal(err)
	}
	cset.Generator("GeoSync v0.1")
	return cset
}

func setNodeInfo(node *osmapi.NodeSt, rec *LogRecord) {
	node.AddTag("name:en", rec.Name)
	node.AddTag("amenity", rec.Amenity)
	node.AddTag("building", rec.Building)
}

func closeChangeSet(cset *osmapi.ChangeSetSt) {
	if err := cset.Close(); err != nil {
		log.Fatal(err)
	}
}

func uploadInserts(req *osmapi.MyRequestSt) {
	cset := newChangeSet(req, "create")
	for i, _ := range inserts {
		latlng := strings.Split(inserts[i].LatLng, ",")
		node, err := cset.NewNode(latlng[0], latlng[1])
		dieIf(err)
		setNodeInfo(node, &inserts[i])
		node.OsmId = fmt.Sprintf("%d", -1-i)
		inserts[i].OsmId = node.OsmId
	}
	newIds, err := cset.MultiUpload()
	dieIf(err)
	for _, rec := range inserts {
		rec.OsmId = newIds[rec.OsmId]
		journal[rec.UniqueID] = rec
		log.Printf("Added node %s (%s: %s)\n", rec.OsmId, rec.UniqueID, rec.LatLng)
	}
	closeChangeSet(cset)
}

func uploadUpdates(req *osmapi.MyRequestSt) {
	cset := newChangeSet(req, "modify")
	for i, _ := range updates {
		latlng := strings.Split(updates[i].LatLng, ",")
		node, err := cset.LoadNode(updates[i].OsmId)
		dieIf(err)
		node.Lat = latlng[0]
		node.Lon = latlng[1]
		setNodeInfo(node, &updates[i])
	}
	newIds, err := cset.MultiUpload()
	dieIf(err)
	for _, rec := range updates {
		if _, ok := newIds[rec.OsmId]; ok {
			journal[rec.UniqueID] = rec
			log.Printf("Updated node %s (%s: %s)\n", rec.OsmId, rec.UniqueID, rec.LatLng)
		}
	}
	closeChangeSet(cset)
}

func uploadDeletions(req *osmapi.MyRequestSt) {
	cset := newChangeSet(req, "delete")
	for i, _ := range deletions {
		node, err := cset.LoadNode(deletions[i].OsmId)
		dieIf(err)
		deletions[i].LatLng = node.Lat + "," + node.Lon
	}
	newIds, err := cset.MultiUpload()
	dieIf(err)
	for _, rec := range deletions {
		if _, ok := newIds[rec.OsmId]; ok {
			delete(journal, rec.UniqueID)
			log.Printf("Deleted node %s (%s: %s)\n", rec.OsmId, rec.UniqueID, rec.LatLng)
		}
	}
	closeChangeSet(cset)
	saveJson(jounalFile, &journal, false)
}

func uploadRecords() {
	req := osmapi.MyRequest(config.OSM_User, config.OSM_Pass)
	req.Url =  config.OSM_Host
	if req == nil {
		log.Fatal("Failed to login")
	}
	uploadInserts(req)
	uploadUpdates(req)
	uploadDeletions(req)
}

func processCSV(csvfp string) {
	f, err := os.Open(csvfp)
	dieIf(err)
	defer f.Close()

	csvr := csv.NewReader(f)
	row, err := csvr.Read()
	if strings.ToLower(fmt.Sprint(row)) != "[uniqueid name amenity building latlng]" {
		log.Fatal("Invalid csv: ", csvfp)
	}
	for {
		row, err := csvr.Read()
		if err != nil {
			break
		}
		rec := LogRecord{UniqueID: row[UID], Name: row[NAME], Amenity: row[AMENITY], Building: row[BUILDING],
			LatLng: row[LATLNG]}
		if lrec, ok := journal[row[UID]]; ok { // already journaled uid
			if lrec.Name == row[NAME] && lrec.Amenity == row[AMENITY] &&
				lrec.Building == row[BUILDING] && lrec.LatLng == row[LATLNG] {
				continue
			}
			rec.OsmId = lrec.OsmId
			if rec.LatLng == "" {
				deletions = append(deletions, rec)
			} else {
				updates = append(updates, rec)
			}
		} else {
			inserts = append(inserts, rec)
		}
	}
	if !quiet {
		var ans string
		fmt.Printf("Found %d insetions, %d updates and %d deletions. Upload (Y/n) ? ",
			len(inserts), len(updates), len(deletions))
		fmt.Scanln(&ans)
		if ans != "Y" && ans != "y" && ans != "" {
			os.Exit(0)
		}
	}
	uploadRecords()
}

func Usage() {
	fmt.Printf("Usage: %s [OPTIONS] csv_file\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.StringVar(&cfgFile, "c", "", "path to configuration file (default: $HOME/.geosync.json)")
	flag.StringVar(&jounalFile, "j", "", "path to jounal file (default: csv_file.log)")
	flag.BoolVar(&quiet, "q", false, "don't prompt for inputs")
	flag.Usage = Usage
	flag.Parse()
	if len(flag.Args()) < 1 {
		Usage()
	} else {
		loadCfg()
		csvFile := flag.Args()[0]

		jounalFile = csvFile + ".log"
		journal = make(map[string]LogRecord)
		loadJson(jounalFile, &journal, false)

		processCSV(csvFile)
	}
}
