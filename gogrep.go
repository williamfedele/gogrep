package main

/*
 * gogrep.go
 * A simple grep-like tool written in Go.
 * Usage: gogrep [OPTIONS] PATTERN FILE...
 * Options:
 * -c: Only a count of selected lines is written to standard output.
 * -e: The pattern is evaluated as a regular expression.
 * -i: Perform case insensitive matching.
 * -l: Only print file names with matches to standard output.
 * -m NUM: Stop reading a file after NUM matching lines.
 * -g: Sets the maximum amount of goroutines to search the files. Default=number of logical CPUs.
 */

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

type Options struct {
	onlyCount            bool
	filesWithMatches     bool
	maxMatches           int
	suppressNormalOutput bool
	numFiles             int
	regexp               bool
	regexpPattern        regexp.Regexp
	caseInsensitive      bool
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

func Regexp(pattern regexp.Regexp) Option {
	return func(o *Options) {
		o.regexp = true
		o.regexpPattern = pattern
	}
}

func CaseInsensitive() Option {
	return func(o *Options) {
		o.caseInsensitive = true
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
		return m.options.regexpPattern.MatchString(input)
	} else {
		return strings.Contains(input, pattern)
	}
}

func NewMatcher(opts ...Option) Matcher {
	options := Options{
		onlyCount:            false,
		filesWithMatches:     false,
		maxMatches:           -1,
		suppressNormalOutput: false,
		numFiles:             1,
		regexp:               false,
		caseInsensitive:      false,
	}
	for _, opt := range opts {
		opt(&options)
	}

	// With certain flags, we don't want to print each matching line
	options.suppressNormalOutput = options.onlyCount || options.filesWithMatches

	return Matcher{
		options: options,
	}
}

func searchFile(
	filePath string,
	matcher Matcher,
	pattern string,
	results chan<- string,
) {

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

			// File search stops once a match is found
			if matcher.options.filesWithMatches {
				break
			}

			if !matcher.options.suppressNormalOutput {
				// The matched pattern is highlighted

				if matcher.options.regexp {
					line = matcher.options.regexpPattern.ReplaceAllString(line, fmt.Sprintf("\x1b[%dm%s\x1b[0m", 32, "$0"))
				} else {
					line = strings.Replace(line, pattern, fmt.Sprintf("\x1b[%dm%s\x1b[0m", 32, pattern), -1)
				}

				if matcher.options.numFiles != 1 {
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

	if matcher.options.onlyCount {
		results <- fmt.Sprintf("%s:%d\n", filePath, lineCount)
	}
	if matcher.options.filesWithMatches && hasMatch {
		results <- fmt.Sprintln(filePath)
	}
}

func worker(jobs <-chan string, results chan<- string, wg *sync.WaitGroup, matcher Matcher, pattern string) {
	defer wg.Done()
	for job := range jobs {
		searchFile(job, matcher, pattern, results)
	}
}

func main() {
	// Parse flags
	var options []Option

	// Optional flags
	onlyCount := flag.Bool("c", false, "Only a count of selected lines is written to standard output.")
	useRegexp := flag.Bool("e", false, "An input line is selected if it matches the pattern evaluated as a regular expression.")
	caseInsensitive := flag.Bool("i", false, "Perform case insensitive matching.")
	filesWithMatches := flag.Bool("l", false, "Only print file names with matches.")
	maxCount := flag.Int("m", -1, "Stop reading a file after m matching lines.")
	maxGoroutines := flag.Int("g", -1, "Sets the maximum amount of goroutines to search the files. Default=amount of files.")

	flag.Parse()

	if *onlyCount {
		options = append(options, OnlyCount())
	}
	if *caseInsensitive {
		options = append(options, CaseInsensitive())
	}
	if *filesWithMatches {
		options = append(options, FilesWithMatches())
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

	// If -e flag is set, compile the pattern as a regular expression
	if *useRegexp {
		re, err := regexp.Compile(pattern)
		if err != nil {
			log.Fatal("Error compiling regular expression:", err)
		}
		options = append(options, Regexp(*re))
	}

	// Determine the number of files to search
	var filePaths []string
	for _, f := range files {
		fileInfo, err := os.Stat(f)
		if err != nil {
			log.Fatal(err)
		}
		// Find all files in a directory and its subdirectories
		if fileInfo.IsDir() {
			err := filepath.WalkDir(f, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() {
					filePaths = append(filePaths, path)
				}
				return nil
			})
			if err != nil {
				log.Fatal(err)
			}
		} else {
			filePaths = append(filePaths, f)
		}
	}
	options = append(options, NumFiles(len(filePaths)))

	// Set the worker pool size. Default is the number of logical CPUs
	poolSize := *maxGoroutines
	if poolSize == -1 {
		poolSize = runtime.NumCPU()
	}

	matcher := NewMatcher(options...)

	var wg sync.WaitGroup
	var printWg sync.WaitGroup
	results := make(chan string)
	jobs := make(chan string, poolSize)

	// Print results from the channel as they arrive
	printWg.Add(1)
	go func() {
		defer printWg.Done()
		for result := range results {
			fmt.Print(result)
		}
	}()

	// Start workers
	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go worker(jobs, results, &wg, matcher, pattern)
	}

	// Launch a goroutine for each file
	for _, file := range filePaths {
		jobs <- file
	}
	// Done sending jobs
	close(jobs)
	// Wait for all workers to finish
	wg.Wait()
	close(results)
	// Wait for the print goroutine to finish
	printWg.Wait()
}
