/*
 This package provides methods to parse a shell-like command line string into a list of arguments.

 Words are split on white spaces, respecting quotes (single and double) and the escape character (backslash)
*/
package args

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"unicode"
)

const (
	ESCAPE_CHAR  = '\\'
	OPTION_CHAR  = '-'
	QUOTE_CHARS  = "`'\""
	SYMBOL_CHARS = `|><#{([`
	NO_QUOTE     = unicode.ReplacementChar
)

var (
	BRACKETS = map[rune]rune{
		'{': '}',
		'[': ']',
		'(': ')',
	}
)

type Scanner struct {
	in              *bufio.Reader
	InfieldBrackets bool
}

// Creates a new Scanner with io.Reader as input source
func NewScanner(r io.Reader) *Scanner {
	sc := Scanner{in: bufio.NewReader(r)}
	return &sc
}

// Creates a new Scanner with a string as input source
func NewScannerString(s string) *Scanner {
	sc := Scanner{in: bufio.NewReader(strings.NewReader(s))}
	return &sc
}

// Get the next token from the Scanner, return io.EOF when done
func (scanner *Scanner) NextToken() (s string, delim int, err error) {
	buf := bytes.NewBufferString("")
	first := true
	escape := false
	quote := NO_QUOTE    // invalid character - not a quote
	brackets := []rune{} // stack of open brackets

	for {
		if c, _, e := scanner.in.ReadRune(); e == nil {
			//
			// check escape character
			//
			if c == ESCAPE_CHAR && !escape {
				escape = true
				first = false
				continue
			}

			//
			// if escaping, just add the character
			//
			if escape {
				escape = false
				buf.WriteString(string(c))
				continue
			}

			//
			// checks for beginning of token
			//
			if first {
				if unicode.IsSpace(c) {
					//
					// skip leading spaces
					//
					continue
				}

				first = false

				if strings.ContainsRune(QUOTE_CHARS, c) {
					//
					// start quoted token
					//
					quote = c
					continue
				}

				if b, ok := BRACKETS[c]; ok {
					//
					// start a bracketed session
					//
					delim = int(c)
					brackets = append(brackets, b)
					buf.WriteString(string(c))
					continue
				}

				if strings.ContainsRune(SYMBOL_CHARS, c) {
					//
					// if it's a symbol, return  all the remaining characters
					//
					buf.WriteString(string(c))
					_, err = io.Copy(buf, scanner.in)
					s = buf.String()
					return // (token, delim, err)
				}
			}

			if len(brackets) == 0 {
				//
				// terminate on spaces
				//
				if unicode.IsSpace(c) && quote == NO_QUOTE {
					s = buf.String()
					delim = int(c)
					return // (token, delim, nil)
				}

				//
				// close quote and terminate
				//
				if c == quote {
					quote = NO_QUOTE
					s = buf.String()
					delim = int(c)
					return // (token, delim, nil)
				}

				if scanner.InfieldBrackets {
					if b, ok := BRACKETS[c]; ok {
						//
						// start a bracketed session
						//
						brackets = append(brackets, b)
					}
				}

				//
				// append to buffer
				//
				buf.WriteString(string(c))
			} else {
				//
				// append to buffer
				//
				buf.WriteString(string(c))

				last := len(brackets) - 1

				if quote == NO_QUOTE {
					if c == brackets[last] {
						brackets = brackets[:last] // pop

						if len(brackets) == 0 {
							s = buf.String()
							return // (token, delim, nil)
						}
					} else if strings.ContainsRune(QUOTE_CHARS, c) {
						//
						// start quoted token
						//
						quote = c
					} else if b, ok := BRACKETS[c]; ok {
						brackets = append(brackets, b)
					}
				} else if c == quote {
					quote = NO_QUOTE
				}
			}
		} else {
			if e == io.EOF {
				if buf.Len() > 0 {
					s = buf.String()
					return // (token, 0, nil)
				}
			}
			err = e
			return // ("", 0, io.EOF)
		}
	}

	return
}

// Return all tokens as an array of strings
func (scanner *Scanner) GetTokens() (tokens []string, err error) {
	tokens, _, err = scanner.getTokens(0)
	return
}

func (scanner *Scanner) GetTokensN(n int) ([]string, string, error) {
	return scanner.getTokens(n)
}

// Return all "option" tokens (tokens that start with "-") and remainder of the line
func (scanner *Scanner) GetOptionTokens() ([]string, string, error) {
	return scanner.getTokens(-1)
}

func (scanner *Scanner) getTokens(max int) ([]string, string, error) {
	tokens := []string{}

	options := max < 0

	for i := 0; max <= 0 || i < max; i++ {
		if options {
			for {
				c, _, err := scanner.in.ReadRune()
				if err == io.EOF {
					return tokens, "", nil
				}
				if err != nil {
					return tokens, "", err
				}

				if c == OPTION_CHAR {
					scanner.in.UnreadRune()
					break
				}

				if !unicode.IsSpace(c) {
					scanner.in.UnreadRune()
					rest, err := ioutil.ReadAll(scanner.in)
					return tokens, string(rest), err
				}

				// skipping spaces until next token
			}
		}

		tok, _, err := scanner.NextToken()
		if err != nil {
			return tokens, "", err
		}

		tokens = append(tokens, tok)
	}

	rest, err := ioutil.ReadAll(scanner.in)
	if err == io.EOF {
		err = nil
	}
	return tokens, strings.TrimSpace(string(rest)), err
}

// GetArgsOption is the type for GetArgs options
type GetArgsOption func(s *Scanner)

// InfieldBrackets enable processing of in-field brackets (i.e. name={"values in brackets"})
func InfieldBrackets() GetArgsOption {
	return func(s *Scanner) {
		s.InfieldBrackets = true
	}
}

func getScanner(line string, options ...GetArgsOption) *Scanner {
	scanner := NewScannerString(line)

	for _, option := range options {
		option(scanner)
	}

	return scanner
}

// Parse the input line into an array of arguments
func GetArgs(line string, options ...GetArgsOption) (args []string) {
	scanner := getScanner(line, options...)
	args, _, _ = scanner.GetTokensN(0)
	return
}

// Parse the input line into an array of max n arguments
func GetArgsN(line string, n int, options ...GetArgsOption) []string {
	scanner := getScanner(line, options...)
	if n > 0 {
		n = n - 1
	}
	args, rest, _ := scanner.GetTokensN(n)
	if rest != "" {
		args = append(args, rest)
	}
	return args
}

func GetOptions(line string, scanOptions ...GetArgsOption) (options []string, rest string) {
	scanner := getScanner(line, scanOptions...)
	options, rest, _ = scanner.GetOptionTokens()
	return
}

type Args struct {
	Options   map[string]string
	Arguments []string
}

func (a Args) GetOption(name, def string) string {
	if val, ok := a.Options[name]; ok {
		return val
	}
	return def
}

func (a Args) GetIntOption(name string, def int) int {
	if val, ok := a.Options[name]; ok {
		n, _ := strconv.Atoi(val)
		return n
	}
	return def
}

func ParseArgs(line string) (parsed Args) {
	parsed = Args{Options: map[string]string{}, Arguments: []string{}}
	args := GetArgs(line)
	if len(args) == 0 {
		return
	}

	for len(args) > 0 {
		arg := args[0]

		if !strings.HasPrefix(arg, "-") {
			break
		}

		args = args[1:]
		if arg == "--" { // stop parsing options
			break
		}

		arg = strings.TrimLeft(arg, "-")
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			key := parts[0]
			value := parts[1]

			parsed.Options[key] = value
		} else {
			parsed.Options[arg] = ""
		}
	}

	parsed.Arguments = args
	return
}

// Create a new FlagSet to be used with ParseFlags
func NewFlags(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)

	flags.Usage = func() {
		fmt.Printf("Usage of %s:\n", name)
		flags.PrintDefaults()
	}

	return flags
}

// Parse the input line through the (initialized) FlagSet
func ParseFlags(flags *flag.FlagSet, line string) error {
	return flags.Parse(GetArgs(line))
}
