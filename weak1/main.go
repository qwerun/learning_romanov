package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"slices"
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
	s, err := getDirs(path)
	if err != nil {
		return err
	}

	for _, val := range s {
		info, err := os.Stat(val)
		if err != nil {
			return err
		}
		if info.IsDir() {
			v, err := getDirs(val)
			if err != nil {
				log.Fatal(err)
			}
			for _, valDir := range v {
				er := worker(out, val, valDir, printFiles)
				if er != nil {
					return er
				}
			}
		}
	}
	return nil
}

func worker(out io.Writer, val, valDir string, printFiles bool) error { // [project static zline zzfile.txt]
	resVal := fmt.Sprintf("%v/%v", val, valDir)

	res, err := getDirs(resVal)
	if err != nil {
		return err
	}

	if !printFiles { //printFiles or !printFiles
		info, err := os.Stat(resVal)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return err
		}
	}

	aboba := len(strings.Split(resVal, "/")) - 1
	_, err = fmt.Fprintf(out, "%v───%s\n", aboba, resVal)
	if err != nil {
		return err
	}
	for _, v := range res {

		er := worker(out, resVal, v, printFiles)
		if er != nil {
			return er
		}
	}
	return nil
}

func getDirs(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return nil, nil
	}
	d, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	s, err := d.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	slices.Sort(s)
	return s, nil
}
