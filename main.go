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
)

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

func removeDuplicates(db *sql.DB, seenChecksums map[string]struct{}, directory string, allFiles int) (int, error) {
	cnt := 0
	pct := 0
	fileCnt := 0

	printOverwrite(fmt.Sprintf("%d/%d (%d%%) - %d files deleted ...", fileCnt, allFiles, pct, cnt))

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileCnt++
			newPct := int(float64(fileCnt*100) / float64(allFiles))
			if newPct > pct {
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
					err := os.Remove(path)
					if err != nil {
						fmt.Println()
						fmt.Printf("Failed to remove duplicate file %s: %v\n", path, err)
						return nil
					}
					cnt++
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
		return cnt, err
	}

	fmt.Println()
	return cnt, nil
}

func main() {
	var directory string
	var mode string

	flag.StringVar(&directory, "i", "", "Directory to scan for duplicates")
	flag.StringVar(&mode, "m", "sql", "Mode: 'sql' or 'mem'")
	flag.Parse()

	if directory == "" {
		fmt.Println("Usage: go run script.go -i /path/to/your/directory [-m sql|mem]")
		os.Exit(1)
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

	cnt, err := removeDuplicates(db, seenChecksums, directory, totalFiles)
	if err != nil {
		fmt.Printf("Error removing duplicates: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%d duplicate files were removed.\n", cnt)
}
