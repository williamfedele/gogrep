package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
)

type Options struct {
	onlyCount            bool
	filesWithMatches     bool
	filesWithoutMatches  bool
	maxMatches           int
	suppressNormalOutput bool
	numFiles             int
	regexp               bool
	caseInsensitive      bool
	invertMatch          bool
}

type Option func(*Options)

func OnlyCount() Option {
	return func(o *Options) {
		o.onlyCount = true
	}
}

func FilesWithMatches() Option {
	return func(o *Options) {
		o.filesWithMatches = true
	}
}

func FilesWithoutMatches() Option {
	return func(o *Options) {
		o.filesWithoutMatches = true
	}
}

func MaxMatches(max int) Option {
	return func(o *Options) {
		o.maxMatches = max
	}
}

func NumFiles(fileCount int) Option {
	return func(o *Options) {
		o.numFiles = fileCount
	}
}

func Regexp() Option {
	return func(o *Options) {
		o.regexp = true
	}
}

func CaseInsensitive() Option {
	return func(o *Options) {
		o.caseInsensitive = true
	}
}

func InvertMatch() Option {
	return func(o *Options) {
		o.invertMatch = true
	}
}

type Matcher struct {
	options Options
}

func (m Matcher) Match(pattern string, input string) bool {
	if m.options.caseInsensitive {
		pattern = strings.ToLower(pattern)
		input = strings.ToLower(input)
	}
	if m.options.regexp {
		matched, err := regexp.MatchString(pattern, input)
		if err != nil {
			return false
		}
		return matched
	} else {
		return strings.Contains(input, pattern)
	}
}

func NewMatcher(opts ...Option) Matcher {
	options := Options{
		onlyCount:            false,
		filesWithMatches:     false,
		filesWithoutMatches:  false,
		maxMatches:           -1,
		suppressNormalOutput: false,
		numFiles:             1,
		regexp:               false,
		caseInsensitive:      false,
	}
	for _, opt := range opts {
		opt(&options)
	}

	// Calculate if normal output is suppressed
	options.suppressNormalOutput = options.onlyCount || options.filesWithMatches || options.filesWithoutMatches

	return Matcher{
		options: options,
	}
}

func searchFile(
	filePath string,
	matcher Matcher,
	pattern string,
	wg *sync.WaitGroup,
	results chan<- string,
) {
	defer wg.Done()

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	hasMatch := false
	lineCount := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		if matcher.Match(pattern, line) {
			hasMatch = true
			lineCount++

			// file search stops once a match is found
			if matcher.options.filesWithMatches {
				break
			}

			if !matcher.options.suppressNormalOutput {
				if matcher.options.numFiles > 1 {
					results <- fmt.Sprintf("%s:%s\n", filePath, line)
				} else {
					results <- fmt.Sprintln(line)
				}
			}

			if matcher.options.maxMatches != -1 && lineCount == matcher.options.maxMatches {
				break
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	if matcher.options.onlyCount || matcher.options.maxMatches != -1 {
		results <- fmt.Sprintf("%s:%d\n", filePath, lineCount)
	}
	if matcher.options.filesWithMatches && hasMatch {
		results <- fmt.Sprintln(filePath)
	}
	if matcher.options.filesWithoutMatches && !hasMatch {
		results <- fmt.Sprintln(filePath)
	}
}

func main() {
	//log.SetLevel(log.DebugLevel)

	var options []Option

	// Optional flags
	onlyCount := flag.Bool("c", false, "Only a count of selected lines is written to standard output.")
	regexp := flag.Bool("e", false, "An input line is selected if it matches any of the specified patterns.")
	caseInsensitive := flag.Bool("i", false, "Perform case insensitive matching.")
	filesWithMatches := flag.Bool("l", false, "Only print files with matches.")
	filesWithoutMatches := flag.Bool("L", false, "Only print files without matches.")
	maxCount := flag.Int("m", -1, "Stop reading a file after m matching lines.")

	flag.Parse()

	if *onlyCount {
		options = append(options, OnlyCount())
	}
	if *regexp {
		options = append(options, Regexp())
	}
	if *caseInsensitive {
		options = append(options, CaseInsensitive())
	}
	if *filesWithMatches {
		options = append(options, FilesWithMatches())
	}
	if *filesWithoutMatches {
		options = append(options, FilesWithoutMatches())
	}
	options = append(options, MaxMatches(*maxCount))

	// Mandatory arguments
	tail := flag.Args()
	// Non-flag args should have a pattern and at least one file
	if len(tail) < 2 {
		log.Fatal("Missing pattern or file.")
	}

	pattern := tail[0]
	files := tail[1:]

	options = append(options, NumFiles(len(files)))

	matcher := NewMatcher(options...)

	var wg sync.WaitGroup
	var printWg sync.WaitGroup
	results := make(chan string)

	// Print results from the channel as they arrive
	printWg.Add(1)
	go func() {
		defer printWg.Done()
		for result := range results {
			fmt.Print(result)
		}
	}()

	// Launch a goroutine for each file
	for _, f := range files {
		wg.Add(1)
		go searchFile(f, matcher, pattern, &wg, results)
	}

	// Wait for all goroutines
	wg.Wait()
	// Close channel
	close(results)
	// Wait for printing to finish before exit
	printWg.Wait()

}
