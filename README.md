# GoGrep

GoGrep is a simple command-line tool written in Go that allows you to search for patterns in files and directories. It provides a similar functionality to the `grep` command.

This tool was made for learning purposes and is way overengineered but gave exposure to popular paradigms in Go. It uses the functional options pattern for flags, Goroutines for performing concurrent searching, and includes a worker pool to experiment with channels

## Installation

To install GoGrep, you need to have Go installed on your system. Then, you can run the following command:

```
go get github.com/williamfedele/gogrep
```

## Usage

To use GoGrep, you can run the following command:

```
gogrep [options] pattern file
```

- `pattern`: The pattern you want to search for.
- `file`: The file(s) or directories you want to search in.

## Options

GoGrep supports the following options:

 - `-c`: Only a count of selected lines is written to standard output.
 - `-e`: The pattern is evaluated as a regular expression.
 - `-i`: Perform case insensitive matching.
 - `-l`: Only print file names with matches to standard output.
 - `-m NUM`: Stop reading a file after NUM matching lines.
 - `-g`: Sets the maximum amount of goroutines to search the files. Default=number of logical CPUs.

## Examples

Here are some examples of how you can use GoGrep:

- Search for a pattern in a single file:

```
gogrep "hello world" myfile.txt
```

- Search for a pattern in multiple files:

```
gogrep "error" log1.txt log2.txt log3.txt
```

- Search for a pattern recursively in a directory:

```
gogrep "TODO" ./src
```

- Search for a pattern ignoring case:

```
gogrep -i "github" README.md
```
