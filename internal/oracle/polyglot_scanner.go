package oracle

import (
	"fmt"
	"regexp"

	"github.com/fuzzspec/fuzzspec/internal/config"
)

// LanguageSignature defines panic and crash patterns for a specific programming language runtime.
type LanguageSignature struct {
	Language string
	Patterns []*regexp.Regexp
}

// DefaultPolyglotSignatures contains built-in crash regex patterns across major backend languages.
var DefaultPolyglotSignatures = []LanguageSignature{
	{
		Language: "Go",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)panic:\s*runtime error`),
			regexp.MustCompile(`goroutine \d+ \[running\]:`),
			regexp.MustCompile(`(?i)invalid memory address or nil pointer dereference`),
			regexp.MustCompile(`(?i)runtime\.goexit`),
		},
	},
	{
		Language: "Python",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`Traceback \(most recent call last\):`),
			regexp.MustCompile(`(?i)(ZeroDivisionError|KeyError|TypeError|IndexError|AttributeError|ValueError):`),
			regexp.MustCompile(`pydantic_core\._pydantic_core\.ValidationError`),
			regexp.MustCompile(`(?i)sqlalchemy\.exc\.`),
		},
	},
	{
		Language: "NodeJS/TypeScript",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`TypeError: Cannot read properties of (undefined|null)`),
			regexp.MustCompile(`UnhandledPromiseRejection`),
			regexp.MustCompile(`at Object\.<anonymous> \(.+:\d+:\d+\)`),
			regexp.MustCompile(`ReferenceError: \w+ is not defined`),
		},
	},
	{
		Language: "Java/Kotlin",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`java\.lang\.(NullPointerException|ArrayIndexOutOfBoundsException|IllegalArgumentException|ArithmeticException)`),
			regexp.MustCompile(`org\.springframework\.web\.bind\.MethodArgumentNotValidException`),
			regexp.MustCompile(`at org\.springframework\.`),
			regexp.MustCompile(`jakarta\.validation\.ValidationException`),
		},
	},
	{
		Language: "PHP/Laravel",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`Fatal error: Uncaught TypeError:`),
			regexp.MustCompile(`Illuminate\\Database\\QueryException`),
			regexp.MustCompile(`PDOException: SQLSTATE\[`),
			regexp.MustCompile(`Symfony\\Component\\ErrorHandler\\Error\\FatalError`),
		},
	},
	{
		Language: "CSharp/.NET",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`System\.NullReferenceException:`),
			regexp.MustCompile(`Microsoft\.AspNetCore\.Diagnostics\.ExceptionHandlerMiddleware`),
			regexp.MustCompile(`at System\..+ in <.+>:\w+`),
		},
	},
	{
		Language: "Rust",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`thread '.+' panicked at`),
			regexp.MustCompile(`called .Result::unwrap\(\). on an .Err. value`),
			regexp.MustCompile(`fatal runtime error: panicking`),
		},
	},
	{
		Language: "Ruby/Rails",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`ActiveRecord::StatementInvalid:`),
			regexp.MustCompile(`NoMethodError \(undefined method`),
			regexp.MustCompile(`ActionController::RoutingError`),
		},
	},
	{
		Language: "Database/Generic",
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)syntax error at or near`),
			regexp.MustCompile(`(?i)You have an error in your SQL syntax`),
			regexp.MustCompile(`(?i)MongoServerError: `),
			regexp.MustCompile(`(?i)ORA-\d{5}: `),
		},
	},
}

// PolyglotScanner scans HTTP responses for internal stack traces, DB errors, and language panics.
type PolyglotScanner struct {
	signatures []LanguageSignature
}

// NewPolyglotScanner initializes the scanner with built-in and custom language signatures.
func NewPolyglotScanner(customLangs []config.CustomLang) *PolyglotScanner {
	sigs := append([]LanguageSignature{}, DefaultPolyglotSignatures...)

	for _, cl := range customLangs {
		var compiled []*regexp.Regexp
		for _, pattern := range cl.StacktracePatterns {
			re, err := regexp.Compile(pattern)
			if err == nil {
				compiled = append(compiled, re)
			}
		}
		if len(compiled) > 0 {
			sigs = append(sigs, LanguageSignature{
				Language: cl.DisplayName,
				Patterns: compiled,
			})
		}
	}

	return &PolyglotScanner{signatures: sigs}
}

// ScanResponse inspects the response body for any language panic or stack trace match.
func (s *PolyglotScanner) ScanResponse(body string) (bool, string, string) {
	if len(body) == 0 {
		return false, "", ""
	}

	for _, sig := range s.signatures {
		for _, pattern := range sig.Patterns {
			if loc := pattern.FindStringIndex(body); loc != nil {
				matchedSnippet := body[loc[0]:loc[1]]
				return true, sig.Language, fmt.Sprintf("Detected %s stack trace / unhandled panic signature: '%s'", sig.Language, matchedSnippet)
			}
		}
	}

	return false, "", ""
}
