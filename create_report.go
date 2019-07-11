package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"io/ioutil"
	"os"
	"strings"
)

// ReportRow is information about running docrun on a single markdown file
type ReportRow struct {
	Path string
	SuccessOther   int
	SuccessTrivial int
	FailureOther   int
	FailureMissing int
}

// FullReport is a full collection of docrun results
type FullReport struct {
	Rows []ReportRow
}

func walkRepository(report *FullReport, repo string) {
	rootPath := filepath.Join(os.Getenv("GOPATH"), "src")
	err := filepath.Walk(filepath.Join(rootPath, repo),
		func(path string, info os.FileInfo, err error) error {
			if strings.HasSuffix(path, ".md") {
				// Found a markdown file. Run docrun over it.
				res := docGetResults(path)
				if !res.Empty() {
					// If there was a non-empty result, add it to report.
					row := ReportRow{
						Path: path[len(rootPath) + 1:],
						SuccessOther:   res.CountSuccess - res.CountTrivial,
						SuccessTrivial: res.CountTrivial,
						FailureOther:   res.CountTotal - res.CountSuccess - res.CountMissing,
						FailureMissing: res.CountMissing,
					}
					report.Rows = append(report.Rows, row)
				}
			}
			return nil
		})
	if err != nil {
		panic(err)
	}
}

func createReport() {
	report := FullReport{Rows: []ReportRow{}}
	data, _ := ioutil.ReadFile("manifest.txt")
	text := string(data)
	repos := strings.Split(text, "\n")
	for _, repo := range repos {
		if repo == "" {
			continue
		}
		walkRepository(&report, repo)
	}
	obj, _ := json.MarshalIndent(report, "", " ")
	fmt.Printf("%s\n", string(obj))
}
