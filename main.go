package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func printOverwrite(message string) {
	fmt.Printf("\r%s", message)
}

func computeSHA256(filePath string) (string, error) {
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

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
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

func removeDuplicates(directory string, allFiles int) (int, error) {
	seenChecksums := make(map[string]struct{})
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

			checksum, err := computeSHA256(path)
			if err != nil {
				fmt.Println()
				fmt.Printf("Failed to compute checksum for file %s: %v\n", path, err)
				return nil
			}

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
		return nil
	})

	if err != nil {
		return cnt, err
	}

	fmt.Println()
	return cnt, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run script.go /path/to/your/directory")
		os.Exit(1)
	}

	directory := os.Args[1]
	totalFiles, err := countFiles(directory)
	if err != nil {
		fmt.Printf("Error counting files: %v\n", err)
		os.Exit(1)
	}

	cnt, err := removeDuplicates(directory, totalFiles)
	if err != nil {
		fmt.Printf("Error removing duplicates: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%d duplicate files were removed.\n", cnt)
}

//TIP See GoLand help at <a href="https://www.jetbrains.com/help/go/">jetbrains.com/help/go/</a>.
// Also, you can try interactive lessons for GoLand by selecting 'Help | Learn IDE Features' from the main menu.
