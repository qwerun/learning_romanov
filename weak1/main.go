package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"slices"
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
				er := worker(val, valDir, printFiles)
				if er != nil {
					return er
				}
			}
		}
	}
	return nil
}

func worker(val, valDir string, printFiles bool) error { // [project static zline zzfile.txt]
	resVal := fmt.Sprintf("%v/%v", val, valDir)

	res, err := getDirs(resVal) // добавить доп. параметр, который будет является bool
	// и в зависимости от признака printFiles будет пропускать файлы если -f
	if err != nil {
		return err
	}
	fmt.Println(resVal)
	for _, v := range res {

		if printFiles {
			info, err := os.Stat(v)
			if err != nil {
				return err
			}
			if info.IsDir() {
				er := worker(resVal, v, printFiles)
				if er != nil {
					return er
				}
			}
			return err
		}

		er := worker(resVal, v, printFiles)
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
