package main

import (
	"bufio"
	"fmt"
	"go-learning/task3/user"
	"io"
	"os"
	"strings"
)

// вам надо написать более быструю оптимальную этой функции
func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	seenBrowsers := make(map[string]bool)

	var foundUsers strings.Builder
	scanner := bufio.NewScanner(file)

	for i := 0; scanner.Scan(); i++ {

		u := user.User{}
		if err := u.UnmarshalJSON(scanner.Bytes()); err != nil {
			panic(err)
		}

		isAndroid := false
		isMSIE := false
		browsers := u.Browsers

		for _, browser := range browsers {
			hasAndroid := strings.Contains(browser, "Android")
			hasMSIE := strings.Contains(browser, "MSIE")

			if hasAndroid {
				isAndroid = true
				seenBrowsers[browser] = true
			}

			if hasMSIE {
				isMSIE = true
				seenBrowsers[browser] = true
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}

		// log.Println("Android and MSIE user:", user["name"], user["email"])
		email := strings.ReplaceAll(u.Email, "@", " [at] ")
		foundUsers.WriteString(fmt.Sprintf("[%d] %s <%s>\n", i, u.Name, email))
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers.String())
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}
