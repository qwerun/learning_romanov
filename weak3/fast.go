package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
)

type User struct {
	Browsers []string `json:"browsers"`
	Company  string   `json:"company"`
	Country  string   `json:"country"`
	Email    string   `json:"email"`
	Job      string   `json:"job"`
	Name     string   `json:"name"`
	Phone    string   `json:"phone"`
}

func FastSearch(out io.Writer) {

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	r := regexp.MustCompile("@")

	uniqueBrowsers := 0
	foundUsers := ""
	users := make([]map[string]interface{}, 0)
	var usersT []User

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		user := make(map[string]interface{})

		// fmt.Printf("%v %v\n", err, line)
		err := json.Unmarshal(scanner.Bytes(), &user)
		if err != nil {
			panic(err)
		}
		users = append(users, user)

		var userT User
		err = json.Unmarshal(scanner.Bytes(), &userT)
		if err != nil {
			panic(err)
		}
		usersT = append(usersT, userT)
	}

	fmt.Printf("%v", usersT)

	seenBrowsers := make([]string, 0, len(users))

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	andr, err := regexp.Compile("Android")
	if err != nil {
		panic(err)
	}
	mse, err := regexp.Compile("MSIE")
	if err != nil {
		panic(err)
	}
	for i, user := range users {
		isAndroid := false
		isMSIE := false
		browsers, ok := user["browsers"].([]interface{})
		if !ok {
			// log.Println("cant cast browsers")
			continue
		}

		for _, browserRaw := range browsers {
			browser, ok := browserRaw.(string)
			if !ok {
				// log.Println("cant cast browser to string")
				continue
			}

			if andr.MatchString(browser) {
				isAndroid = true
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}

			if mse.MatchString(browser) {
				isMSIE = true
				notSeenBefore := true
				for _, item := range seenBrowsers {
					if item == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					// log.Printf("SLOW New browser: %s, first seen: %s", browser, user["name"])
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}
		}

		if !(isAndroid && isMSIE) {
			continue
		}

		// log.Println("Android and MSIE user:", user["name"], user["email"])
		email := r.ReplaceAllString(user["email"].(string), " [at] ")
		foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, user["name"], email)
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))

}
