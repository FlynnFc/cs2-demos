package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	demoinfocs "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
	"github.com/tealeg/xlsx"
)

type FinalScore struct {
	CT int
	T  int
}

type TeamDetails struct {
	Name  string
	Score int
}

type PlayerStats struct {
	ID      uint64
	Name    string `json:"name"`
	Rounds  int
	Matches int
	Kills   int `json:"kills"`
	Deaths  int `json:"deaths"`
	Damage  int
}

type BasicMatchDetails struct {
	Team1 TeamDetails
	Team2 TeamDetails
	Stats []PlayerStats
}

var paths []string
var failed []string

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
)

func main() {
	// ANSI escape codes for text colors

	playerStatsMap := []PlayerStats{}
	var root string

	//User enters path of folder
	fmt.Print("Please enter the path for the demo folder: ")
	fmt.Scan(&root)
	fmt.Println("Path=", root)

	err := filepath.WalkDir(root, visitFile)
	if err != nil {
		fmt.Printf("error walking the path %v: %v\n", root, err)
	}

	fmt.Printf(Blue+"You are parsing %d files. Estimated time: %d - %d seconds\n"+Reset, len(paths), len(paths)-2, len(paths)+4)
	maxWorkers := 10
	overall := time.Now()
	pathChannel := make(chan string, len(paths))
	resultChannel := make(chan []PlayerStats, len(paths))

	for i := 0; i < maxWorkers; i++ {
		go func() {
			for path := range pathChannel {
				fmt.Printf(Yellow+"Parsing %s\n"+Reset, path)
				start := time.Now()
				game := demoParsing(path)
				elapsed := time.Since(start)
				fmt.Printf(Green+"%s took %s\n"+Reset, path, elapsed)
				resultChannel <- game
			}
		}()
	}

	for _, path := range paths {
		pathChannel <- path
	}

	close(pathChannel)

	for i := 0; i < len(paths); i++ {
		playerStatsMap = append(playerStatsMap, <-resultChannel...)
	}

	fmt.Println(Green + "\nAll demos processed!" + Reset)
	fmt.Println()

	elapsed := time.Since(overall)

	if len(failed) > 1 {
		fmt.Printf(Red+"%d out of %d demos were invalid and were ignored.\n"+Reset, len(failed), len(paths))
	}
	fmt.Printf(Green+"Parsing %d demos took %s\n"+Reset, len(paths)-len(failed), elapsed)
	mergedData := make(map[uint64]PlayerStats)

	for _, item := range playerStatsMap {
		if existing, found := mergedData[item.ID]; found {
			var newRounds = existing.Rounds + item.Rounds
			var newKills = existing.Kills + item.Kills
			var newDeaths = existing.Deaths + item.Deaths
			var newDamage = existing.Damage + item.Damage
			var matchesPlayed = existing.Matches + item.Matches
			mergedData[item.ID] = PlayerStats{Rounds: newRounds, Kills: newKills, Deaths: newDeaths, Damage: newDamage, Name: existing.Name, ID: existing.ID, Matches: matchesPlayed}
		} else {
			// If the item is not a duplicate, add it to the map
			mergedData[item.ID] = item
		}
	}

	mergedArray := make([]PlayerStats, 0, len(mergedData))
	for _, value := range mergedData {
		mergedArray = append(mergedArray, value)
	}
	excelExporter(mergedArray)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func playerStatsCalc(player *common.Player, totalRounds int) PlayerStats {
	var output = PlayerStats{ID: player.SteamID64, Name: player.Name, Rounds: totalRounds, Kills: player.Kills(), Deaths: player.Deaths(), Damage: player.TotalDamage(), Matches: 1}
	return output
}

func excelExporter(allPlayers []PlayerStats) {
	fmt.Println(Yellow + "Building spreadsheet..." + Reset)
	file := xlsx.NewFile()
	sheet, err := file.AddSheet("Basicstats")
	checkError(err)

	//
	//Header row initialising
	//
	headerRow := sheet.AddRow()
	headerRow.AddCell().Value = "ID"
	headerRow.AddCell().Value = "Name"
	headerRow.AddCell().Value = "Matches"
	headerRow.AddCell().Value = "Rounds"
	headerRow.AddCell().Value = "Kills"
	headerRow.AddCell().Value = "Deaths"
	headerRow.AddCell().Value = "Damage"

	for _, player := range allPlayers {
		var id = strconv.FormatUint(player.ID, 10)
		row := sheet.AddRow()
		row.AddCell().SetString(id)
		row.AddCell().SetString(player.Name)
		row.AddCell().SetInt(player.Matches)
		row.AddCell().SetInt(player.Rounds)
		row.AddCell().SetInt(player.Kills)
		row.AddCell().SetInt(player.Deaths)
		row.AddCell().SetFloat(float64(player.Damage))
	}

	err = file.Save("epicstats.xlsx")
	checkError(err)
	fmt.Println(Green + "Spreadsheet done" + Reset)
}

func demoParsing(path string) []PlayerStats {
	f, err := os.Open(path)
	checkError(err)

	defer f.Close()
	output := []PlayerStats{}
	p := demoinfocs.NewParser(f)
	defer p.Close()

	//Game ended
	p.RegisterEventHandler(func(e events.AnnouncementWinPanelMatch) {

		allPlayers := p.GameState().Participants().Playing()
		score := FinalScore{p.GameState().TeamCounterTerrorists().Score(), p.GameState().TeamTerrorists().Score()}

		totalRounds := score.CT + score.T

		if score.CT < 12 && score.T < 12 {
			//Not 1 half has been completed so match is likely void
			failed = append(failed, path)
			fmt.Printf(Red+"Demo %s not valid. Too few rounds player\n"+Reset, path)
			return
		}

		for _, player := range allPlayers {
			stats := playerStatsCalc(player, totalRounds)
			output = append(output, stats)
		}

	})

	// Parse the whole demo
	err = p.ParseToEnd()

	if err != nil {
		failed = append(failed, path)
		return output
	}
	return output
}

func visitFile(fp string, fi os.DirEntry, err error) error {
	if err != nil {
		fmt.Println(err) // can't walk here,
		return nil       // but continue walking elsewhere
	}
	if fi.IsDir() {
		return nil // not a file. ignore.
	}
	paths = append(paths, fp)
	return nil
}
