package score

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/h2non/filetype"
	"github.com/sirupsen/logrus"
)

type Options struct {
	OS                []string
	Arch              []string
	Extensions        []string
	Names             []string
	Versions          []string
	Terms             []string
	WeightedTerms     map[string]int
	InvalidOS         []string
	InvalidArch       []string
	InvalidExtensions []string
}

func (o *Options) GetAllStrings() []string {
	var allStrings []string
	allStrings = append(allStrings, o.OS...)
	allStrings = append(allStrings, o.Arch...)
	allStrings = append(allStrings, o.Terms...)
	allStrings = append(allStrings, o.Names...)
	allStrings = append(allStrings, o.Versions...)

	for _, key := range o.Versions {
		allStrings = append(allStrings, fmt.Sprintf("v%s", key))
	}

	return allStrings
}

func Score(names []string, opts *Options) []Sorted { //nolint:gocyclo
	logger := logrus.WithField("function", "score")
	logger.Tracef("names: %v", names)

	var scores = make(map[string]int)

	for _, name := range names {
		var score int
		var scoringValues = make(map[string]int)

		for _, name1 := range opts.Names {
			if name1 == name {
				scores = map[string]int{
					name: 200,
				}
				return SortMapByValue(scores)
			}
		}

		// Note: if it has the word "update" in it, we want to deprioritize it as it's likely an update binary from
		// a rust or go binary distribution
		// TODO: move this out of the function to a weighted term
		scoringValues["update"] = -100
		scoringValues["-keyless.sig"] = -10

		for _, os1 := range opts.OS {
			scoringValues[strings.ToLower(os1)] = 40
		}
		for _, arch := range opts.Arch {
			scoringValues[strings.ToLower(arch)] = 30
		}
		for _, ext := range opts.Extensions {
			scoringValues[strings.ToLower(ext)] = 20
		}
		for _, term := range opts.Terms {
			scoringValues[strings.ToLower(term)] = 10
		}

		for _, os1 := range opts.InvalidOS {
			scoringValues[strings.ToLower(os1)] = -40
		}
		for _, arch := range opts.InvalidArch {
			scoringValues[strings.ToLower(arch)] = -30
		}
		for _, ext := range opts.InvalidExtensions {
			scoringValues[strings.ToLower(ext)] = -20
		}

		for term, weight := range opts.WeightedTerms {
			scoringValues[strings.ToLower(term)] = weight
		}

		for keyMatch, keyScore := range scoringValues {
			if keyScore == 20 { // handle extensions special
				if ext := strings.TrimPrefix(filepath.Ext(strings.ToLower(name)), "."); ext != "" {
					for _, fileExt := range opts.Extensions {
						if filetype.GetType(ext) == filetype.GetType(fileExt) {
							score += keyScore
							break
						}
					}
				}
			} else {
				if strings.Contains(strings.ToLower(name), keyMatch) {
					score += keyScore
				}
			}
		}

		scores[name] = score + calculateAccuracyScore(name, opts.GetAllStrings())
	}

	return SortMapByValue(scores)
}

func removeExtension(filename string) string {
	for {
		newFilename := filename
		newExt := filepath.Ext(newFilename)
		if len(newExt) > 5 || strings.Contains(newExt, "_") {
			break
		}

		newFilename = strings.TrimSuffix(newFilename, newExt)

		if newFilename == filename {
			break
		}

		filename = newFilename
	}

	return filename
}

func calculateAccuracyScore(filename string, knownTerms []string) int {
	logrus.Trace("calculating accuracy score for filename: ", filename)
	filename = removeExtension(filename) // Remove the file extension
	logrus.Trace("filename after removing extension: ", filename)

	// Split the filename by dashes and dots to get individual terms
	terms := strings.FieldsFunc(filename, func(r rune) bool {
		return r == '-' || r == '_'
	})

	// discovered terms
	for i, term := range terms {
		logrus.Tracef("term %d: %s", i, term)
	}

	for i, term := range knownTerms {
		logrus.Tracef("known term %d: %s", i, term)
	}

	// Initialize the score
	score := 0

	// Create a map for quick lookup of known terms
	knownMap := make(map[string]bool)
	for _, term := range knownTerms {
		knownMap[term] = true
	}

	// Check each term in the filename
	for _, term := range terms {
		if filename == term {
			logrus.WithField("filename", filename).Trace("adding point for term: ", term)
			score += 10 // Add points for a direct match
		} else if knownMap[term] {
			logrus.WithField("filename", filename).Trace("adding point for term: ", term)
			score += 2 // Add point for a correct match
		} else {
			logrus.WithField("filename", filename).Trace("subtracting point for term: ", term)
			score += -5 // Add a larger penalty for an unknown term
		}
	}

	return score
}

type Sorted struct {
	Key   string
	Value int
}

func SortMapByValue(m map[string]int) []Sorted {
	var sorted []Sorted

	// Create a slice of key-value pairs
	for k, v := range m {
		sorted = append(sorted, struct {
			Key   string
			Value int
		}{k, v})
	}

	// Sort the slice based on the values in descending order
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Value == sorted[j].Value {
			return sorted[i].Key < sorted[j].Key
		}
		return sorted[i].Value > sorted[j].Value
	})

	return sorted
}
