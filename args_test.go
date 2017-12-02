package args

import (
	"fmt"
	"testing"
)

const (
	TEST_STRING  = `the   quick 	  "brown  'fox'"  jumps 'o v e r' \"the\"\ lazy dog`
	PARSE_STRING = "-l --number=42 -where=here -- -not-an-option- one two three # a comment \n next line"

	TEST_BRACKETS = `some stuff in "quotes" and {"brackets":[1, 'help', (2+3)]} {{"a":1,"b":2},{"c":3}} x={"value":"with brakets", a=[1, "2", 3.14, {"another": "field"}]}`
)

func TestScanner(test *testing.T) {
	scanner := NewScannerString(TEST_STRING)

	for {
		token, delim, err := scanner.NextToken()
		if err != nil {
			test.Log(err)
			break
		}

		test.Logf("%q %q", delim, token)
	}
}

func TestScannerInfieldBrackets(test *testing.T) {
	scanner := NewScannerString(TEST_BRACKETS)
	scanner.InfieldBrackets = true

	for {
		token, delim, err := scanner.NextToken()
		if err != nil {
			test.Log(err)
			break
		}

		test.Logf("%q %q", delim, token)
	}
}

func TestGetArgs(test *testing.T) {

	test.Logf("%q", GetArgs(TEST_STRING))
}

func TestGetArgsN(test *testing.T) {

	args := GetArgsN(TEST_STRING, 3)
	test.Logf("%q", args)
}

func TestGetOptions(test *testing.T) {

	options, rest := GetOptions(PARSE_STRING)
	test.Logf("%q %q", options, rest)
}

func TestParseArgs(test *testing.T) {

	test.Logf("%q", ParseArgs(PARSE_STRING))
}

func TestBrackets(test *testing.T) {

	for i, a := range GetArgs(TEST_BRACKETS) {
		fmt.Println(i, a)
	}
}

func TestBracketsInfield(test *testing.T) {

	for i, a := range GetArgs(TEST_BRACKETS, InfieldBrackets()) {
		fmt.Println(i, a)
	}
}

func ExampleGetArgs() {
	s := `one two three "double quotes" 'single quotes' arg\ with\ spaces "\"quotes\" in 'quotes'" '"quotes" in \'quotes'"`

	for i, arg := range GetArgs(s) {
		fmt.Println(i, arg)
	}
	// Output:
	// 0 one
	// 1 two
	// 2 three
	// 3 double quotes
	// 4 single quotes
	// 5 arg with spaces
	// 6 "quotes" in 'quotes'
	// 7 "quotes" in 'quotes
}

func ExampleParseArgs() {
	arguments := "-l --number=42 -where=here -- -not-an-option- one two three |pipers piping"

	parsed := ParseArgs(arguments)

	fmt.Println("options:", parsed.Options)
	fmt.Println("arguments:", parsed.Arguments)
	// Output:
	// options: map[l: number:42 where:here]
	// arguments: [-not-an-option- one two three |pipers piping]
}

func ExampleParseFlags() {
	arguments := "-l --number=42 -where=here -- -not-an-option- one two three"

	flags := NewFlags("args")

	list := flags.Bool("l", false, "list something")
	num := flags.Int("number", 0, "a number option")
	where := flags.String("where", "", "a string option")

	if err := ParseFlags(flags, arguments); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("list:", *list)
		fmt.Println("num:", *num)
		fmt.Println("where:", *where)
		fmt.Println("args:", flags.Args())
	}
	// Output:
	// list: true
	// num: 42
	// where: here
	// args: [-not-an-option- one two three]
}
