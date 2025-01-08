package main

import (
	"crypto/sha256"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var version = "1.2.1"

func printOverwrite(message string) {
	fmt.Printf("\r%s", message)
}

func initDB() (*sql.DB, error) {
	dbPath := "./copycure.db"
	err := os.Remove(dbPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS files (
		checksum TEXT PRIMARY KEY,
		path TEXT NOT NULL
	);
	`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func computeSHA256(db *sql.DB, filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	checksum := fmt.Sprintf("%x", hash.Sum(nil))

	return checksum, nil
}

func countFiles(directory string) (int, error) {
	var totalFiles int
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalFiles++
		}
		return nil
	})
	return totalFiles, err
}
func deletePath(path string, noconfirm bool) (bool, error) {
	var response string
	deleted := false
	var err error
	if !noconfirm {
		fmt.Printf("\nDo you want to delete %s? (y/n): ", path)
		fmt.Scanln(&response)
	} else {
		response = "y"
	}
	if strings.ToLower(response) == "y" {
		err = os.Remove(path)
		if err != nil {
			fmt.Println()
			fmt.Printf("Failed to remove duplicate file %s: %v\n", path, err)
		} else {
			deleted = true
		}
	} else {
		fmt.Printf("Skipped %s\n", path)
	}
	return deleted, err
}

func removeDuplicates(db *sql.DB, seenChecksums map[string]struct{}, directory string, allFiles int, noConfirm bool, exclude []string, deleteEmpty bool, listMode bool) (int, int, error) {
	cnt := 0
	dbl := 0
	pct := 0
	fileCnt := 0
	if !listMode {
		printOverwrite(fmt.Sprintf("%d/%d (%d%%) - %d files deleted ...", fileCnt, allFiles, pct, cnt))
	}

	deleteSizeLimit := int64(10)
	if deleteEmpty {
		deleteSizeLimit = int64(-1)
	}

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !containsSubstring(path, exclude) && info.Size() > deleteSizeLimit {
			fileCnt++
			newPct := int(float64(fileCnt*100) / float64(allFiles))
			if newPct > pct && !listMode {
				pct = newPct
				printOverwrite(fmt.Sprintf("%d/%d (%d%%) - %d files deleted ...", fileCnt, allFiles, pct, cnt))
			}

			checksum, err := computeSHA256(db, path)
			if err != nil {
				fmt.Println()
				fmt.Printf("Failed to compute checksum for file %s: %v\n", path, err)
				return nil
			}

			if db != nil {
				var existingPath string
				err = db.QueryRow("SELECT path FROM files WHERE checksum = ?", checksum).Scan(&existingPath)
				if err == sql.ErrNoRows {
					// No duplicate found
					_, err = db.Exec("INSERT INTO files (checksum, path) VALUES (?, ?)", checksum, path)
					if err != nil {
						fmt.Println("failed inserting checksum for file %s: %v\n", path, err)
					}
				} else if err != nil {
					fmt.Println()
					fmt.Printf("Failed to query database for file %s: %v\n", path, err)
					return nil
				} else {
					if listMode {
						fmt.Println(path)
					} else {
						deleted, _ := deletePath(path, noConfirm)
						if deleted {
							cnt++
						}
					}
					dbl++
				}
			} else if seenChecksums != nil {
				if _, exists := seenChecksums[checksum]; exists {
					err := os.Remove(path)
					if err != nil {
						fmt.Println()
						fmt.Printf("Failed to remove duplicate file %s: %v\n", path, err)
						return nil
					}
					cnt++
				} else {
					seenChecksums[checksum] = struct{}{}
				}
			}

		}
		return nil
	})

	if err != nil {
		return cnt, dbl, err
	}

	fmt.Println()
	return cnt, dbl, nil
}

func containsSubstring(path string, exclude []string) bool {
	for _, s := range exclude {
		if s == "" {
			continue
		}
		if strings.Contains(path, s) {
			return true
		}
	}
	return false
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func main() {
	var directory string
	var mode string
	var excludeParam string
	flag.StringVar(&directory, "i", "", "Directory to scan for duplicates")
	flag.StringVar(&mode, "m", "sql", "Mode: 'sql' or 'mem'")
	_ = flag.Bool("y", false, "Delete files without asking")
	_ = flag.Bool("e", false, "Delete empty files too. Empty files all look the same to copycure. The Default is to ignore them")
	_ = flag.Bool("l", false, "List the duplicates instead of deleting them. -y is automatically assumed to be set")
	flag.StringVar(&excludeParam, "x", "", "Comma separated list of partial filenames to exclude (e.g. -e .venv/,.git/)")
	flag.Parse()
	exclude := strings.Split(excludeParam, ",")

	noConfirm := isFlagPassed("y")
	deleteEmpty := isFlagPassed("e")
	listMode := isFlagPassed("l")

	if directory == "" {
		fmt.Printf("CopyCure %v (written by Sammy Fischer)\n", version)
		fmt.Println("Usage: copycure -i /path/to/your/directory [-m sql|mem] [-y] [-x aaa,bbb] [-e] [-l]\n" +
			" -m {sql|mem} : method to store known checksums. \n\tsql: use a temporary sqllite database  mem: store in and array in RAM\n" +
			" -y : remove files without asking\n" +
			" -x {comma separated list} : exclude any file whose path contains one of the list values\n\t(e.g. -e .venv,.git ignores any path containing .venv or .git)}\n" +
			" -e : remove duplicate empty files (size==0). Default is to ignore them.\n" +
			" -l : only list the full path to the duplicates found without deleting them.\n")
		os.Exit(1)
	}

	if !listMode {
		fmt.Printf("CopyCure %v (written by Sammy Fischer)\n", version)
	}

	var db *sql.DB
	var seenChecksums map[string]struct{}

	switch mode {
	case "sql":
		var err error
		db, err = initDB()
		if err != nil {
			fmt.Printf("Error initializing database: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()
	case "mem":
		seenChecksums = make(map[string]struct{})
	default:
		fmt.Println("Invalid mode. Use 'sql' or 'mem'.")
		os.Exit(1)
	}

	totalFiles, err := countFiles(directory)
	if err != nil {
		fmt.Printf("Error counting files: %v\n", err)
		os.Exit(1)
	}

	cnt, dbl, err := removeDuplicates(db, seenChecksums, directory, totalFiles, noConfirm, exclude, deleteEmpty, listMode)
	if err != nil {
		fmt.Printf("Error removing duplicates: %v\n", err)
		os.Exit(1)
	}

	if !listMode {
		fmt.Printf("%d duplicates found, %d files were removed.\n", dbl, cnt)
	}
}
