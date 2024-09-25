package practical

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// RegexReplaceFileDeprecate performs a regex match on a file and
// replaces the file when repl is non-empty
func RegexReplaceFile(filepath, regexPattern string, repl string) ([]MatchedLine, error) {
	frr := newFileRegexReplacer(filepath, regexPattern, repl)
	defer frr.close()
	frr.scan()
	frr.replace()
	return frr.result, frr.err
}

type MatchedLine struct {
	FilePath    string
	LineNumber  int
	Matches     []string
	Replace     bool
	Replacement string
}

func (r MatchedLine) String() string {
	var firstMatch string
	if len(r.Matches) > 0 {
		firstMatch = r.Matches[0]
	}
	if r.Replace {
		return fmt.Sprintf("Line %d of %s\nRegex match: %s\nReplacement: %s", r.LineNumber, r.FilePath, firstMatch, r.Replacement)
	}
	return fmt.Sprintf("Line %d of %s\nRegex match: %s", r.LineNumber, r.FilePath, firstMatch)
}

type fileRegexReplacer struct {
	filePath string
	regex    *regexp.Regexp
	repl     string

	file     *os.File
	tempFile *os.File
	scanner  *bufio.Scanner
	writer   *bufio.Writer

	err    error
	result []MatchedLine
}

func newFileRegexReplacer(filepath, pattern, repl string) fileRegexReplacer {
	ret := fileRegexReplacer{filePath: filepath, repl: repl}
	ret.file, ret.err = os.Open(filepath)
	if ret.err != nil {
		ret.err = errors.WithMessagef(ret.err, "failed to open %s", filepath)
		return ret
	}
	ret.scanner = bufio.NewScanner(ret.file)

	if ret.expectReplace() {
		ret.tempFile, ret.err = os.CreateTemp("", "tempfile-*.txt")
		if ret.err != nil {
			ret.err = errors.WithMessagef(ret.err, "failed to create temp file for %s", filepath)
			return ret
		}
		ret.writer = bufio.NewWriter(ret.tempFile)
	}

	ret.regex, ret.err = regexp.Compile(pattern)
	if ret.err != nil {
		ret.err = errors.WithMessagef(ret.err, "invalid regex pattern: %s", pattern)
		return ret
	}

	return ret
}

func (frr *fileRegexReplacer) expectReplace() bool {
	return len(frr.repl) > 0
}

// scan file and capture regex match result, write the replacement content to a new file
func (frr *fileRegexReplacer) scan() {
	if frr.err != nil {
		return
	}

	var lineNumber int
	for frr.scanner.Scan() {
		line := frr.scanner.Text()

		updatedLine := frr.regex.ReplaceAllStringFunc(line, func(match string) string {
			submatches := frr.regex.FindStringSubmatch(match)
			if len(submatches) > 0 {
				result := frr.repl
				if frr.expectReplace() {
					for i := 0; i < len(submatches); i++ {
						placeholderLiteral := fmt.Sprintf(`$%d`, i)
						placeholder := fmt.Sprintf(`\$%d`, i)
						if !strings.Contains(frr.repl, placeholderLiteral) {
							continue
						}
						result = regexp.MustCompile(placeholder).ReplaceAllString(result, submatches[i])
					}
				}
				frr.result = append(frr.result, MatchedLine{
					LineNumber:  lineNumber,
					FilePath:    frr.filePath,
					Matches:     submatches,
					Replace:     frr.expectReplace(),
					Replacement: result,
				})
				return result
			}
			return match
		})

		if frr.expectReplace() {
			_, err := frr.writer.WriteString(updatedLine + "\n")
			if err != nil {
				frr.err = errors.WithMessage(err, "failed to write to temp file")
				return
			}
		}

		lineNumber++
	}

	if err := frr.scanner.Err(); err != nil {
		frr.err = errors.WithStack(err)
		return
	}
}

// replace the old file
func (frr *fileRegexReplacer) replace() {
	if frr.err != nil || !frr.expectReplace() {
		return
	}

	if err := frr.writer.Flush(); err != nil {
		frr.err = errors.WithMessage(err, "error flushing to temp file")
		return
	}
	if err := os.Rename(frr.tempFile.Name(), frr.filePath); err != nil {
		frr.err = errors.WithMessagef(err, "failed to replace original file: %s", frr.filePath)
		return
	}
	if err := os.Chmod(frr.filePath, 0666); err != nil {
		frr.err = errors.WithMessagef(err, "cannot chmod of file %s", frr.filePath)
		return
	}
}

func (frr *fileRegexReplacer) close() {
	if frr.file != nil {
		frr.file.Close()
	}
	if frr.tempFile != nil {
		frr.tempFile.Close()
		os.Remove(frr.tempFile.Name())
	}
}
