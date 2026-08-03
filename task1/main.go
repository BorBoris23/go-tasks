package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	return tree(out, path, printFiles, "")
}

func tree(out io.Writer, path string, printFiles bool, prefix string) error {
	items, err := readDirectory(path)
	if err != nil {
		return err
	}

	if items == nil {
		return nil
	}

	sortItems(items)

	visibleItems := getVisibleItems(items, printFiles)

	for i, item := range visibleItems {

		last := i == len(visibleItems)-1

		itemPrefix := makePrefix(prefix, last)

		itemPath := filepath.Join(path, item.Name())

		printItem(out, itemPrefix, item, printFiles)

		if item.IsDir() {
			err := tree(
				out,
				itemPath,
				printFiles,
				makeChildPrefix(prefix, last),
			)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func readDirectory(path string) ([]os.FileInfo, error) {
	object, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer object.Close()

	objectInfo, err := object.Stat()
	if err != nil {
		return nil, err
	}

	if !objectInfo.IsDir() {
		return nil, nil
	}

	items, err := object.Readdir(-1)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func sortItems(items []os.FileInfo) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name() == items[j].Name() {
			return items[i].IsDir()
		}

		return items[i].Name() < items[j].Name()
	})
}

func getVisibleItems(items []os.FileInfo, printFiles bool) []os.FileInfo {
	visibleItems := make([]os.FileInfo, 0)

	for _, item := range items {
		if item.IsDir() || printFiles {
			visibleItems = append(visibleItems, item)
		}
	}

	return visibleItems
}

func printItem(out io.Writer, prefix string, item os.FileInfo, printFiles bool) {
	if item.IsDir() {
		fmt.Fprintln(out, prefix+item.Name())
		return
	}

	if printFiles {
		if item.Size() == 0 {
			fmt.Fprintf(out, "%s%s (empty)\n", prefix, item.Name())
		} else {
			fmt.Fprintf(out, "%s%s (%db)\n", prefix, item.Name(), item.Size())
		}
	}
}

func makeChildPrefix(prefix string, last bool) string {
	if last {
		return prefix + strings.Repeat("\t", 1)
	}

	return prefix + "│" + strings.Repeat("\t", 1)
}

func makePrefix(prefix string, last bool) string {
	branch := "├───"

	if last {
		branch = "└───"
	}

	return prefix + branch
}
