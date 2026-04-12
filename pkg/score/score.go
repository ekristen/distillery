package score

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/h2non/filetype"
	"github.com/rs/zerolog/log"
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
	InvalidTerms      []string
	InvalidLibrary    []string
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

// getMultiSegmentTerms collects all terms from options that contain delimiters (-/_)
func (o *Options) getMultiSegmentTerms() []string {
	var allTerms []string
	allTerms = append(allTerms, o.OS...)
	allTerms = append(allTerms, o.Arch...)
	allTerms = append(allTerms, o.Terms...)
	allTerms = append(allTerms, o.InvalidOS...)
	allTerms = append(allTerms, o.InvalidArch...)
	allTerms = append(allTerms, o.InvalidTerms...)
	allTerms = append(allTerms, o.InvalidLibrary...)
	allTerms = append(allTerms, o.Versions...)
	for _, v := range o.Versions {
		allTerms = append(allTerms, "v"+v)
	}
	for k := range o.WeightedTerms {
		allTerms = append(allTerms, k)
	}

	var result []string
	for _, term := range allTerms {
		if strings.ContainsAny(term, "-_.") {
			result = append(result, term)
		}
	}
	return result
}

// segmentize splits a filename into lowercase segments, preserving multi-segment
// terms (those containing - or _) as single units via placeholder replacement.
func segmentize(filename string, multiSegmentTerms []string) []string {
	filename = removeExtension(filename)

	sorted := make([]string, len(multiSegmentTerms))
	copy(sorted, multiSegmentTerms)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i]) > len(sorted[j])
	})

	replacements := make(map[string]string)
	modified := strings.ToLower(filename)

	for i, term := range sorted {
		lower := strings.ToLower(term)
		placeholder := fmt.Sprintf("PLACEHOLDER%d", i)
		if strings.Contains(modified, lower) {
			replacements[placeholder] = lower
			modified = strings.ReplaceAll(modified, lower, placeholder)
		}
	}

	segments := strings.FieldsFunc(modified, func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	})

	for i, seg := range segments {
		if original, ok := replacements[seg]; ok {
			segments[i] = original
		}
	}

	return segments
}

// segmentContains checks if any segment matches the term (case-insensitive)
func segmentContains(segments []string, term string) bool {
	lower := strings.ToLower(term)
	for _, seg := range segments {
		if seg == lower {
			return true
		}
	}
	return false
}

func Score(names []string, opts *Options) []Sorted {
	logger := log.With().Str("function", "score").Logger()
	logger.Trace().Msgf("names: %v", names)

	scores := make(map[string]int)
	multiSegTerms := opts.getMultiSegmentTerms()
	allStrings := opts.GetAllStrings()

	for _, name := range names {
		// Exact name match
		for _, n := range opts.Names {
			if n == name {
				scores = map[string]int{name: 200}
				return SortMapByValue(scores)
			}
		}

		segments := segmentize(name, multiSegTerms)
		scores[name] = scoreSegments(name, segments, opts) + calculateAccuracyScore(name, allStrings)
		logger.Trace().Msgf("scoring %s with score %d", name, scores[name])
	}

	return SortMapByValue(scores)
}

// firstSegmentMatch returns weight if any term matches a segment, 0 otherwise
func firstSegmentMatch(segments, terms []string, weight int) int {
	for _, term := range terms {
		if segmentContains(segments, term) {
			return weight
		}
	}
	return 0
}

// allSegmentMatches returns weight * count of matching terms
func allSegmentMatches(segments, terms []string, weight int) int {
	score := 0
	for _, term := range terms {
		if segmentContains(segments, term) {
			score += weight
		}
	}
	return score
}

// extensionMatch returns weight if the file extension MIME-matches any in the list
func extensionMatch(filename string, extensions []string, weight int) int {
	ext := strings.TrimPrefix(filepath.Ext(strings.ToLower(filename)), ".")
	if ext == "" {
		return 0
	}
	for _, fileExt := range extensions {
		if filetype.GetType(ext) == filetype.GetType(fileExt) {
			return weight
		}
	}
	return 0
}

func scoreSegments(name string, segments []string, opts *Options) int {
	score := firstSegmentMatch(segments, opts.OS, 40)
	score += firstSegmentMatch(segments, opts.Arch, 30)
	score += extensionMatch(name, opts.Extensions, 20)
	score += allSegmentMatches(segments, opts.Terms, 10)
	score += firstSegmentMatch(segments, opts.InvalidOS, -40)
	score += firstSegmentMatch(segments, opts.InvalidArch, -30)
	score += extensionMatch(name, opts.InvalidExtensions, -20)
	score += allSegmentMatches(segments, opts.InvalidTerms, -10)
	score += allSegmentMatches(segments, opts.InvalidLibrary, -30)

	// WeightedTerms: substring match preserved (for checksum discovery)
	lowerName := strings.ToLower(name)
	for term, weight := range opts.WeightedTerms {
		if strings.Contains(lowerName, strings.ToLower(term)) {
			score += weight
		}
	}

	return score
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
	log.Trace().Msgf("calculating accuracy score for filename: %s", filename)

	var multiSegTerms []string
	for _, term := range knownTerms {
		if strings.ContainsAny(term, "-_.") {
			multiSegTerms = append(multiSegTerms, term)
		}
	}

	segments := segmentize(filename, multiSegTerms)
	lowerFilename := strings.ToLower(removeExtension(filename))

	for i, term := range segments {
		log.Trace().Msgf("term %d: %s", i, term)
	}

	for i, term := range knownTerms {
		log.Trace().Msgf("known term %d: %s", i, term)
	}

	score := 0

	knownMap := make(map[string]bool)
	for _, term := range knownTerms {
		knownMap[strings.ToLower(term)] = true
	}

	for _, term := range segments {
		currentScore := score
		if lowerFilename == term {
			score += 10
			log.Trace().Str("filename", filename).Int("current", currentScore).Int("new", score).Msgf("adding points (10) for term: %s", term)
		} else if knownMap[term] {
			score += 2
			log.Trace().Str("filename", filename).Int("current", currentScore).Int("new", score).Msgf("adding points (2) for term: %s", term)
		} else {
			score -= 5
			log.Trace().Str("filename", filename).Int("current", currentScore).Int("new", score).Msgf("subtracting points (5) for term: %s", term)
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
