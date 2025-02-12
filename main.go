package main

import (
	"fmt"
	"os"
	"encoding/csv"
	"io/ioutil"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"strings"
	"flag"
	"github.com/pkg/errors"
	"path"
)

type GTFSFile struct {
	name string
	fields []string
}

func main() {
	var gtfsDir string
	var dbPath string
	var batchSize int


	flag.StringVar(&gtfsDir, "source", "", "Directory path for gtfs files is passed in source flag")
	flag.StringVar(&dbPath, "db", "", "target db path goes in db flag")
	flag.IntVar(&batchSize, "size", 100000, "batch size can be passed default: 100000")
	flag.Parse()

	if gtfsDir == "" || dbPath == "" {
		panic(errors.New("source and db are params are required"))
	}


	db, err := sql.Open("sqlite3", dbPath)

	checkError(err)

	defer db.Close()

	files, err := getFileNames(gtfsDir)

	checkError(err)

	for _,file := range files {

		filePath := fmt.Sprintf(path.Join(gtfsDir, file))
		data, headers, err := readCSV(filePath, true)

		checkError(err)

		name := strings.Replace(file, ".txt", "", 1)
		db.Exec(getCreateTableQuery(name, headers))
		fmt.Printf("%s created\n", name)

		if batchSize > 0 {
			batchSliceTask(data, batchSize, getMultiInsertTask(db, name))
		} else {
			db.Exec(getMultiInsertQuery(name, data))
		}

	}
}

func getFileNames(dirPath string) (fileNames []string, err error){
	files, err := ioutil.ReadDir(dirPath)

	if err != nil {
		return
	}

	for _, fileInfo := range files {
		if ! fileInfo.IsDir() {
			fileNames = append(fileNames, fileInfo.Name())
		}
	}
	return
}

func readCSV(filePath string, hasHeader bool) (fileData [][]string, header []string, err error) {
	csvFile, err := os.Open(filePath);

	if err != nil {
		return
	}

	defer csvFile.Close()

	reader := csv.NewReader(csvFile)

	if hasHeader {
		reader.FieldsPerRecord = 0
	} else {
		reader.FieldsPerRecord = -1
	}

	fileData, err = reader.ReadAll()

	if err != nil {
		return
	}

	if len(fileData) > 0 && hasHeader {
		header = fileData[0]
		deleteFromSlice(&fileData, 0)
	}
	return
}

func deleteFromSlice(s *[][]string, i int) {
	sl := *s
	*s = append(sl[:i], sl[i + 1:]...)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func getMultiInsertTask (db *sql.DB, name string) (func(data [][]string)){
	return func(data [][]string) {
		query := getMultiInsertQuery(name, data)
		db.Exec(query)
		fmt.Printf("%d records added to %s\n", len(data), name)
	}
}

func getCreateTableQuery(name string, headers []string) (finalQuery string) {
	innerQuery := strings.Join(headers[:], " test, ") + " text"
	finalQuery = fmt.Sprintf("Create table %s (%s);", name, innerQuery)
	return
}

func getMultiInsertQuery(name string, data [][]string) (finalQuery string) {
	var innerQueries []string
	for _, record := range data {
		innerQueries = append(innerQueries[:], "('" + strings.Join(record[:], "', '") + "')")
	}

	finalQuery = fmt.Sprintf("Insert into %s values %s;", name, strings.Join(innerQueries[:], ", "))
	return
}

func batchSliceTask (s [][]string, size int, task func(data [][]string)) {
	var subSlice [][]string
	run := len(s) > 0
	for run {
		if size < len(s) {
			subSlice = s[:size]
			s = s[size:]
			task(subSlice)
		} else {
			task(s)
			run = false
		}
	}
}
