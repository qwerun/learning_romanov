package forEasyJson

import (
	"bufio"
	"fmt"
	"github.com/mailru/easyjson"
	"io"
	"os"
	"regexp"
)

//easyjson:json
type User struct {
	Browsers []string `json:"browsers"`
	Email    string   `json:"email"`
	Name     string   `json:"name"`
}

var filePath string = "./data/users.txt"

func FastEasyJson(out io.Writer) {

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	r := regexp.MustCompile("@")

	uniqueBrowsers := 0
	foundUsers := ""

	var usersT []User

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var userT User
		err = easyjson.Unmarshal(scanner.Bytes(), &userT)
		if err != nil {
			panic(err)
		}
		usersT = append(usersT, userT)
	}

	seenBrowsers := make([]string, 0, len(usersT))

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

	for g, userT := range usersT {
		isAndroid := false
		isMSIE := false

		for _, browser := range userT.Browsers {
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
		email := r.ReplaceAllString(userT.Email, " [at] ")
		foundUsers += fmt.Sprintf("[%d] %s <%s>\n", g, userT.Name, email)
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}
