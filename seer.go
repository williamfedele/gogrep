package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/log"
)

type MatcherOptions struct {
	regexp          bool
	caseInsensitive bool
}

type Option func(*MatcherOptions)

func CaseSensitive() Option {
	return func(o *MatcherOptions) {
		o.caseInsensitive = true
	}
}

func Regexp() Option {
	return func(o *MatcherOptions) {
		o.regexp = true
	}
}

type Matcher struct {
	options MatcherOptions
}

func makeMatcher(opts ...Option) *Matcher {
	options := MatcherOptions{
		caseInsensitive: false,
		regexp:          false,
	}
	for _, opt := range opts {
		opt(&options)
	}
	return &Matcher{
		options: options,
	}
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

func main() {
	var options []Option

	// optional flags
	onlyCount := flag.Bool("c", false, "Only a count of selected lines is written to standard output.")
	regexp := flag.Bool("e", false, "An input line is selected if it matches any of the specified patterns.")
	caseInsensitive := flag.Bool("i", false, "Perform case insensitive matching.")

	flag.Parse()

	if *caseInsensitive {
		options = append(options, CaseSensitive())
	}

	if *regexp {
		options = append(options, Regexp())
	}

	// non-flag args
	tail := flag.Args()
	if len(tail) < 2 {
		log.Fatal("Missing pattern or file.")
	}

	pattern := tail[0]
	files := tail[1:]

	matcher := makeMatcher(options...)

	lineCount := 0
	for _, f := range files {
		file, err := os.Open(f)
		defer file.Close()
		if err != nil {
			log.Fatal(err)
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) == 0 {
				continue
			}
			if matcher.Match(pattern, line) {
				if *onlyCount {
					lineCount++
				} else {
					fmt.Println(line)
				}

			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

	}
	if *onlyCount {
		fmt.Println(lineCount)
	}

}
