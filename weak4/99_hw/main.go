package main

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Row struct {
	ID         int    `xml:"id"`
	Age        int    `xml:"age"`
	FirstName  string `xml:"first_name"`
	LastName   string `xml:"last_name"`
	Gender     string `xml:"gender"`
	About      string `xml:"about"`
	Registered string `xml:"registered"`
	Name       string `xml:"-"`
}

type Root struct {
	XMLName xml.Name `xml:"root"`
	Rows    []Row    `xml:"row"`
}

func main() {

	//http.HandleFunc("/search", Search)
	//if err := http.ListenAndServe(":3000", nil); err != nil {
	//	log.Fatalf("Server failed: %v", err)
	//}

	//go func() {
	//	http.HandleFunc("/search", Search)
	//	if err := http.ListenAndServe(":3000", nil); err != nil {
	//		log.Fatalf("Server failed: %v", err)
	//	}
	//}()

	//time.Sleep(100 * time.Millisecond)

	//sc := SearchClient{
	//	AccessToken: "mycooltoken123",
	//	URL:         "http://localhost:3000/search",
	//}
	//cases := []SearchRequest{
	//	{
	//		Limit:      10,
	//		Offset:     0,
	//		Query:      "Boy",
	//		OrderField: "Name",
	//		OrderBy:    OrderByAsIs,
	//	},
	//	{
	//		Limit:      10,
	//		Offset:     0,
	//		Query:      "ga",
	//		OrderField: "Name",
	//		OrderBy:    OrderByDesc,
	//	},
	//}
	//
	//for i, cs := range cases {
	//	r, err := sc.FindUsers(cs)
	//	if err != nil {
	//		log.Fatalf("FindUsers Error: %v, caseNum: %v", err, i)
	//	}
	//
	//	fmt.Printf("main Result: %+v\n", r.Users)
	//}
	//
	//select {}
}

func SearchServerErrors(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	mockError := queryParams.Get("query")

	switch mockError {
	case "timeout":
		time.Sleep(2 * time.Second)
	case "500":
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	case "invalid_json":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 123, "name": "test, 2}`))
		return
	case "400_unknown":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		errResp := SearchErrorResponse{Error: ""}
		json.NewEncoder(w).Encode(errResp)
		return
	case "400_cant_unpack":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`[]`))
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("AccessToken")
	if token != "mycooltoken123" {
		http.Error(w, "Bad token", http.StatusUnauthorized)
	}

	queryParams := r.URL.Query()
	maxIterations := 10000
	orderField := queryParams.Get("order_field")
	orderByStr := queryParams.Get("order_by")
	query := queryParams.Get("query")
	limitStr := queryParams.Get("limit")
	offsetStr := queryParams.Get("offset")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 0
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}
	orderBy, err := strconv.Atoi(orderByStr)
	if err != nil {
		orderBy = OrderByAsIs
	}
	dataName := "dataset.xml"

	file, err := os.Open(dataName)

	if err != nil {
		http.Error(w, "Ошибка при открытии файла", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	var root Root
	decoder := xml.NewDecoder(file)
	if err := decoder.Decode(&root); err != nil {
		http.Error(w, "Ошибка при декодировании XML", http.StatusInternalServerError)
		return
	}
	var filteredRows []Row
	cnt := 0

	for i, row := range root.Rows {
		if i < offset {
			continue
		}
		if (limit == 0 && cnt == maxIterations) || (limit > 0 && cnt == limit) {
			break
		}
		root.Rows[i].Name = row.FirstName + row.LastName
		if !(strings.Contains(strings.ToLower(root.Rows[i].Name), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(root.Rows[i].About), strings.ToLower(query))) {
			continue
		}
		filteredRows = append(filteredRows, root.Rows[i])
		cnt++
	}

	switch orderField {
	case "Id", "Age", "Name":
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		errResp := SearchErrorResponse{Error: "ErrorBadOrderField"}
		json.NewEncoder(w).Encode(errResp)
		return
	}

	if orderBy != OrderByAsIs {
		sort.SliceStable(filteredRows, func(i, j int) bool {
			switch orderField {
			case "Id":
				if orderBy == OrderByAsc {
					return filteredRows[i].ID < filteredRows[j].ID
				} else if orderBy == OrderByDesc {
					return filteredRows[i].ID > filteredRows[j].ID
				}
			case "Age":
				if orderBy == OrderByAsc {
					return filteredRows[i].Age < filteredRows[j].Age
				} else if orderBy == OrderByDesc {
					return filteredRows[i].Age > filteredRows[j].Age
				}
			case "Name":
				if orderBy == OrderByAsc {
					return filteredRows[i].Name < filteredRows[j].Name
				} else if orderBy == OrderByDesc {
					return filteredRows[i].Name > filteredRows[j].Name
				}
			}
			return false
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filteredRows)
}
