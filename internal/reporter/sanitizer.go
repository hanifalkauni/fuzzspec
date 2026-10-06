package reporter

import (
	"regexp"
	"strings"
)

var (
	tokenRegex       = regexp.MustCompile(`(?i)(Bearer\s+)[A-Za-z0-9\-\._~\+\/]+=*`)
	apiKeyRegex      = regexp.MustCompile(`(?i)(api[_-]?key[:=]\s*["']?)[A-Za-z0-9\-\._~]{8,}(["']?)`)
	passwordRegex    = regexp.MustCompile(`(?i)("password"\s*:\s*")[^"]+(")`)
	secretRegex      = regexp.MustCompile(`(?i)("?(?:client_secret|access_token|refresh_token|secret_key|secret)"?\s*[:=]\s*["']?)[A-Za-z0-9\-\._~]{8,}(["']?)`)
	openAIKeyRegex   = regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`)
	anthropicKeyRegex = regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{20,}`)
	awsKeyRegex      = regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
	githubPatRegex   = regexp.MustCompile(`ghp_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{50,}`)
	privateKeyRegex  = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]+?-----END [A-Z ]*PRIVATE KEY-----`)
	dbConnRegex      = regexp.MustCompile(`(?i)(postgres|mysql|mongodb(?:\+srv)?|redis):\/\/[^:]+:[^@]+@`)
)

// SanitizeText scrubs sensitive credentials and PII from log/report strings.
func SanitizeText(input string) string {
	if len(input) == 0 {
		return input
	}

	sanitized := tokenRegex.ReplaceAllString(input, "${1}[REDACTED_TOKEN]")
	sanitized = apiKeyRegex.ReplaceAllString(sanitized, "${1}[REDACTED_API_KEY]${2}")
	sanitized = passwordRegex.ReplaceAllString(sanitized, "${1}[REDACTED_PASSWORD]${2}")
	sanitized = secretRegex.ReplaceAllString(sanitized, "${1}[REDACTED_SECRET]${2}")
	sanitized = openAIKeyRegex.ReplaceAllString(sanitized, "sk-[REDACTED_OPENAI_KEY]")
	sanitized = anthropicKeyRegex.ReplaceAllString(sanitized, "sk-ant-[REDACTED_ANTHROPIC_KEY]")
	sanitized = awsKeyRegex.ReplaceAllString(sanitized, "[REDACTED_AWS_KEY]")
	sanitized = githubPatRegex.ReplaceAllString(sanitized, "[REDACTED_GITHUB_TOKEN]")
	sanitized = privateKeyRegex.ReplaceAllString(sanitized, "[REDACTED_PRIVATE_KEY]")
	sanitized = dbConnRegex.ReplaceAllString(sanitized, "${1}://[REDACTED_USER]:[REDACTED_PASS]@")

	return sanitized
}

// SanitizeHeaders creates a safe copy of HTTP headers with masked tokens.
func SanitizeHeaders(headers map[string][]string) map[string][]string {
	safe := make(map[string][]string)
	for k, v := range headers {
		lowerK := strings.ToLower(k)
		if lowerK == "authorization" || lowerK == "x-api-key" || lowerK == "cookie" || lowerK == "set-cookie" || lowerK == "proxy-authorization" {
			safe[k] = []string{"[REDACTED]"}
		} else {
			safe[k] = v
		}
	}
	return safe
}
