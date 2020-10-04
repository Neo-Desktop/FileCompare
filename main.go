package main

import (
	"bufio"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type DataFile struct {
	Filename string
	Filepath string
	Filesize int64
}

type DataStore struct {
	Files map[string][]DataFile
	Paths []string
}

var (
	ds           *DataStore
	dupe, unique int
)

func reinit() {
	ds = &DataStore{
		Files: make(map[string][]DataFile),
		Paths: make([]string, 0),
	}
	dupe, unique = 0, 0
}

func main() {

	reinit()

	fmt.Println("File Catalog")
	fmt.Println("---------------------")
	stdin := bufio.NewReader(os.Stdin)

	catalogue := choseCatalogue(stdin)
	loadCatalogue(catalogue)

	for {
		fmt.Println("Next path to scan (q to quit)")
		fmt.Print("-> ")

		searchDir, _ := stdin.ReadString('\n')
		searchDir = strings.Replace(searchDir, "\r", "", -1)
		searchDir = strings.Replace(searchDir, "\n", "", -1)

		if len(searchDir) == 0 || strings.ToLower(searchDir) == "q" {
			break
		}

		err := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {

			if f != nil && f.IsDir() {
				return nil
			}

			hashError, hash := hashFile(path)
			if hashError != nil {
				return hashError
			}
			file := DataFile{
				filepath.Base(path),
				filepath.Dir(path),
				f.Size(),
			}

			if ds.Files[hash] == nil {
				ds.Files[hash] = make([]DataFile, 0)
				// log.Printf("+[%s] - %s", hash, path)
				unique++
			} else {
				dupe++
				log.Printf("![%d] %s", len(ds.Files[hash])+1, path)
			}
			ds.Files[hash] = append(ds.Files[hash], file)

			return err

			// end file loop
		})
		if err != nil {
			panic(err)
		}

		update := ""
		for update == "" {
			fmt.Printf("Job finished, %d duplicates found, update catalogue?\n", dupe)
			fmt.Printf("[y/N/q]-> ")
			update, _ = stdin.ReadString('\n')
			update = strings.Replace(update, "\r", "", -1)
			update = strings.Replace(update, "\n", "", -1)
			update = strings.ToLower(update)

			if update == "y" {
				saveCatalogue(catalogue)
			} else if len(update) == 0 || update == "n" {
				fmt.Println("Discarding changes...")
				reinit()
				loadCatalogue(catalogue)
				break
			} else if update == "q" {
				fmt.Println("Proceeding to exit...")
				break
			} else {
				update = ""
			}
		}
		if update == "q" {
			break
		}

		// end main loop
	}

	fmt.Println("Save text summary of duplicate files?")
	fmt.Printf("[y/N]-> ")
	summary, _ := stdin.ReadString('\n')
	summary = strings.Replace(summary, "\r", "", -1)
	summary = strings.Replace(summary, "\n", "", -1)
	summary = strings.ToLower(summary)

	if summary == "y" {
		saveSummary("duplicates.csv")
	} else {
		log.Println("Exiting...")
	}
}

func saveSummary(path string) {
	fileExists, err := os.Stat(path)
	summaryFile, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer summaryFile.Close()

	if fileExists == nil {
		summaryFile.WriteString(fmt.Sprintf("\"%s\",\"%s\",\"%s\",\"%s\"\n", "hash", "filepath", "bytes", "copies"))
	}

	for hash, files := range ds.Files {
		if len(files) > 1 {
			// _, err := summaryFile.WriteString(fmt.Sprintf("%s - %s Duplicates", hash, len(files)))
			for _, file := range files {
				summaryFile.WriteString(fmt.Sprintf("\"%s\",\"%s\",\"%d\",\"%d\"\n", hash, filepath.Join(file.Filepath, file.Filename), file.Filesize, len(files)))
			}
		}
	}

	log.Printf("Saved summary to: %s\n", path)
}

func hashFile(filePath string) (error, string) {
	file, err := os.Open(filePath)
	if err != nil {
		return err, ""
	}
	defer file.Close()

	hash := md5.New()
	_, err = io.Copy(hash, file)

	return nil, hex.EncodeToString(hash.Sum(nil))
}

func choseCatalogue(stdin *bufio.Reader) string {
	fmt.Println("Path to catalogue file")
	fmt.Print("[./save.gob]-> ")

	catalogue, _ := stdin.ReadString('\n')
	catalogue = strings.Replace(catalogue, "\r", "", -1)
	catalogue = strings.Replace(catalogue, "\n", "", -1)

	if catalogue == "" {
		catalogue = "save.gob"
	}

	return catalogue
}

func saveCatalogue(path string) {
	flags := os.O_TRUNC | os.O_RDWR | os.O_EXCL
	file, err := os.Stat(path)
	if file == nil {
		flags |= os.O_CREATE
	}

	dataFile, err := os.OpenFile(path, flags, 0644)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer dataFile.Close()

	dataEncoder := gob.NewEncoder(dataFile)
	err = dataEncoder.Encode(*ds)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	log.Printf("File Saved: %s\n", path)
}

func loadCatalogue(path string) {
	file, err := os.Stat(path)
	if file == nil {
		saveCatalogue(path)
	}
	dataFile, err := os.OpenFile(path, os.O_RDWR|os.O_EXCL, 0644)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer dataFile.Close()

	dataDecoder := gob.NewDecoder(dataFile)
	err = dataDecoder.Decode(ds)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	log.Printf("File Loaded: %s\n", path)

}
