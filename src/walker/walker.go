package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/tkrajina/gpxgo/gpx"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"pg-walker/src/config"
	"strconv"
	"time"
)

var properties map[string]string
var cwd string
var timezone string

type moveInstruction struct {
	lat      float64
	long     float64
	distance float64
	time     float64
}

type DeviceInfo []struct {
	Name         string `json:"name"`
	Display_name string `json:"display_name"`
	Udid         string `json:"udid"`
}

func main() {
	var err error
	cwd, err = os.Getwd()
	if err != nil {
		log.Fatal("error trying to get the current working directory")
	}

	properties, err = config.LoadProperties(cwd + "/pg-walker.properties")
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) > 1 {
		run(os.Args[1])
	} else {
		run("loop")
	}
}

func run(city string) {
	log.Println(properties)
	gpxBytes, err := os.ReadFile(cwd + "/src/res/" + city + ".xml")
	check(err)
	gpxData, err := gpx.ParseBytes(gpxBytes)
	check(err)
	initRoute(calcInformationSet(gpxData.Tracks[0].Segments[0].Points))
}

func initRoute(instructions []moveInstruction) {
	calcRouteDuration(instructions)
	timezone = getTimezone(instructions[0].lat, instructions[0].long)
	var udid = getDeviceUdid()
	for i := 0; i < len(instructions); i++ {
		values := map[string]interface{}{
			"lat":  instructions[i].lat,
			"lng":  instructions[i].long,
			"udid": udid}
		jsonData, err := json.Marshal(values)
		check(err)
		printInfos(i, instructions)
		http.Post("http://localhost:49215/set_location", "application/json", bytes.NewBuffer(jsonData))
		time.Sleep(time.Duration(instructions[i].time) * time.Second)

		if i+1 == len(instructions) {
			i = -1
		}
	}
}

func getTimezone(lat float64, long float64) string {
	resp, err := http.Get("http://api.timezonedb.com/v2.1/get-time-zone?key=" + properties["timezone_api_key"] + "&format=json&by=position&lat=" + fmt.Sprint(lat) + "&lng=" + fmt.Sprint(long))
	if err != nil {
		return ""
	}
	body, err := ioutil.ReadAll(resp.Body)
	check(err)
	sb := string(body)
	var result map[string]string
	json.Unmarshal([]byte(sb), &result)
	return result["zoneName"]
}

func getDeviceUdid() string {
	resp, err := http.Get("http://localhost:49215/get_devices")
	check(err)
	body, err := ioutil.ReadAll(resp.Body)
	check(err)
	sb := string(body)
	deviceData := &DeviceInfo{}
	err = json.Unmarshal([]byte(sb), deviceData)
	check(err)
	return ((*deviceData)[0].Udid)
}

func calcInformationSet(coords []gpx.GPXPoint) []moveInstruction {
	var instructionSet []moveInstruction
	for k := range coords {
		if len(coords) != k+1 {
			moveInstruction := moveInstruction{
				lat:  coords[k].Latitude,
				long: coords[k].Longitude,
			}
			moveInstruction.distance = math.Round(gpx.Distance2D(coords[k].Latitude, coords[k].Longitude, coords[k+1].Latitude, coords[k+1].Longitude, false))
			moveInstruction.time = calcTravelTime(moveInstruction.distance)
			instructionSet = append(instructionSet, moveInstruction)
		}
	}
	return instructionSet
}

func calcRouteDuration(instructions []moveInstruction) {
	var sum = 0.0
	for e := range instructions {
		sum += instructions[e].time
	}
	var duration = secondsToMinutes(int(sum))
	fmt.Println("Duration of Route: " + duration)
	lineSeparator()
}

func calcTravelTime(distance float64) float64 {
	speed, err := strconv.ParseFloat(properties["speed"], 32)
	if err != nil {
		speed = 3
	}
	return math.Round(distance / speed)
}

func printInfos(e int, instructions []moveInstruction) {
	printTime(timezone)
	printWaypointCount(e, len(instructions))
	printLatLng(instructions[e].lat, instructions[e].long)
	fmt.Println("Time before moving to next waypoint: ")
	fmt.Println(fmt.Sprint(instructions[e].time) + " seconds")
	lineSeparator()
}

func printTime(timezone string) {
	if timezone != "" {
		now := time.Now()
		loc, _ := time.LoadLocation(timezone)
		fmt.Println(now.In(loc).Format(time.RFC822))
	} else {
		fmt.Println(time.Now().Format(time.RFC822))
	}
}

func printWaypointCount(current int, max int) {
	fmt.Println(getWaypointCount(current, max) + " Moving to waypoint with coordinates: ")
}

func getWaypointCount(val int, max int) string {
	return "(" + strconv.Itoa(val+1) + "/" + strconv.Itoa(max) + ")"
}

func printLatLng(lat float64, long float64) {
	fmt.Print(fmt.Sprint(lat) + "," + fmt.Sprint(long) + "\n")
}

func secondsToMinutes(inSeconds int) string {
	minutes := inSeconds / 60
	seconds := inSeconds % 60
	str := fmt.Sprint(minutes) + ":" + fmt.Sprint(seconds)
	return str
}

func lineSeparator() {
	fmt.Println("--------------------------------------")
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
