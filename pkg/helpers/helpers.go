package helpers

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Enumerated type representing the platform underlying the cluster-under-test
const (
	PlatformAWS           = "aws"           // deprecated: use PlatformAWSClassic
	PlatformGCP           = "gcp"           // deprecated: use PlatformGCPClassic
	PlatformHostedCluster = "hostedcluster" // deprecated: use PlatformAWSHCP
	PlatformAWSClassic    = "aws-classic"
	PlatformGCPClassic    = "gcp-classic"
	PlatformAWSHCP        = "aws-hcp"
)

// Enumerated type representing CPU architectures
const (
	ArchX86 = "x86"
	ArchARM = "arm"
)

// RandSeq generates random string with n characters.
func RandSeq(n int) string {
	b := make([]rune, n)
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// PollImmediate calls the condition function at the specified interval up to the specified timeout
// until the condition function returns true or an error
func PollImmediate(interval time.Duration, timeout time.Duration, condition func() (bool, error)) error {
	var totalTime time.Duration = 0

	for totalTime < timeout {
		cond, err := condition()
		if err != nil {
			return err
		}

		if cond {
			return nil
		}

		time.Sleep(interval)
		totalTime += interval
	}

	return errors.New("timed out waiting for the condition")
}

// IPPermissionsEquivalent compares two AWS IpPermissions (used in security group rules)
// similarly to reflect.DeepEqual, except it only compares IP(v6) ports and CIDR ranges
// and ignores UserIdGroupPairs, PrefixListIds, and any Description fields
func IPPermissionsEquivalent(a ec2Types.IpPermission, b ec2Types.IpPermission) bool {
	if *a.FromPort != *b.FromPort || *a.ToPort != *b.ToPort || *a.IpProtocol != *b.IpProtocol {
		return false
	}

	if len(a.IpRanges) != len(b.IpRanges) || len(a.Ipv6Ranges) != len(b.Ipv6Ranges) {
		return false
	}

	for _, ipRangeA := range a.IpRanges {
		foundEquivalentIPRange := false
		// Nested loop necessary to check for out-of-order IpRanges
		for _, ipRangeB := range b.IpRanges {
			if *ipRangeA.CidrIp == *ipRangeB.CidrIp {
				// We only need to find one IpRange in b that matches ipRangeA, so break here
				foundEquivalentIPRange = true
				break
			}
		}
		if !foundEquivalentIPRange {
			return false
		}
	}

	for _, ipv6RangeA := range a.Ipv6Ranges {
		foundEquivalentIPv6Range := false
		// Nested loop necessary to check for out-of-order Ipv6Ranges
		for _, ipv6RangeB := range b.Ipv6Ranges {
			if *ipv6RangeA.CidrIpv6 == *ipv6RangeB.CidrIpv6 {
				// We only need to find one Ipv6Range in b that matches ipv6RangeA, so break here
				foundEquivalentIPv6Range = true
				break
			}
		}
		if !foundEquivalentIPv6Range {
			return false
		}
	}

	return true
}

func GetPlatformType(platformType string) (string, error) {
	switch platformType {
	case PlatformAWS, PlatformAWSClassic:
		return "aws", nil
	case PlatformGCP, PlatformGCPClassic:
		return "gcp", nil
	case PlatformHostedCluster, PlatformAWSHCP:
		return "hostedcluster", nil
	default:
		return "", errors.New("invalid platform type")
	}
}

// The following regular expressions are used in fixLeadingZerosInJSON. They'll be used
// hundreds of times per verifier run, so we declare them globally to avoid unnecessary
// recompilation
var reJSONIntsWithLeadingZero = regexp.MustCompile(`":\s*0+[^,.]+[,}]`)
var reDigits = regexp.MustCompile(`0*(\d+)`)
var reBracketedISO8601Timestamps = regexp.MustCompile(`\[[\d-]+T[\d:.]+]`)

// fixLeadingZerosInJSON attempts to detect unsigned integers containing leading zeros
// (e.g, 061 or 000) in strings containing raw JSON and replace them with spec-compliant
// de-zeroed equivalents. Leading zeros are invalid in JSON, but curl v7.76 and below
// contain a bug (github.com/curl/curl/issues/6905) that emits them in status codes.
// Note that strContainingJSON can contain substrings that are not JSON, but
// such substrings will be subjected to the same regexes and may therefore be modified
// unintentionally
func FixLeadingZerosInJSON(strContainingJSON string) string {
	return reJSONIntsWithLeadingZero.ReplaceAllStringFunc(
		strContainingJSON,
		func(substrContainingNum string) string {
			return string(reDigits.ReplaceAll([]byte(substrContainingNum), []byte("$1"))[:])
		},
	)
}

// RemoveTimestamps attempts to detect and remove the bracketed ISO-8601 timestamps that
// AWS inexplicably inserts into the output of ec2.GetConsoleOutput() whenever a line
// exceeds a certain length. This function simply returns the input string after passing
// it through regexp.ReplaceAllLiteralString()
func RemoveTimestamps(strContainingTimestamps string) string {
	return reBracketedISO8601Timestamps.ReplaceAllLiteralString(strContainingTimestamps, "")
}

// ExtractRequiredVariablesDirective looks for a "directive line" in a YAML string resembling:
// # network-verifier-required-variables=VAR_X,VAR_Y,VAR_Z
// If such a string is found, the comma-separated values after the '=' are transformed into a
// slice of strings, e.g., ["VAR_X", "VAR_Y", "VAR_Z"]. That slice is returned alongside the
// original string, minus any directive lines. Only variables listed in the first (leftmost)
// directive line will be extracted/returned, but all directive lines will be removed
func ExtractRequiredVariablesDirective(yamlStr string) (string, []string) {
	reDirective := regexp.MustCompile(
		`(?m)^[ \t]*#[ \t]*network-verifier-required-variables[ \t]*=[ \t]*([\w,]+)[ \t]*$`,
	)

	submatches := reDirective.FindStringSubmatch(yamlStr)
	// submatches will be either nil or a 2-str slice (full directive line, comma-separated values)
	if len(submatches) < 2 {
		return yamlStr, []string{}
	}

	// Erase the directive line and split comma-separated vars string into slice
	directivelessYAMLStr := reDirective.ReplaceAllLiteralString(yamlStr, "")
	requiredVariables := strings.Split(strings.TrimSpace(submatches[1]), ",")
	return directivelessYAMLStr, requiredVariables
}

// ValidateProvidedVariables returns an error if either (a.) providedVarMap contains a key also present in
// presetVarMap, or (b.) requiredVarSlice contains a value not present in the union of providedVarMap's keys
// and presetVarMap's keys. IOW, this returns nil as long as providedVarMap.keys ∩ presetVarMap.keys = ∅ and
// requiredVarSlice ⊆ (providedVarMap.keys ∪ presetVarMap.keys)
func ValidateProvidedVariables(providedVarMap map[string]string, presetVarMap map[string]string, requiredVarSlice []string) error {
	// Error if user tries to set preset variables
	for providedVarName := range providedVarMap {
		if _, isPreset := presetVarMap[providedVarName]; isPreset {
			return fmt.Errorf("must not overwrite preset user-data variable %s", providedVarName)
		}
	}

	// Error if required variables not set
	for _, requiredVarName := range requiredVarSlice {
		// Ignore requiredVar if pre-set
		if _, isPreset := presetVarMap[requiredVarName]; isPreset {
			continue
		}
		if providedValue, isProvided := providedVarMap[requiredVarName]; !isProvided || providedValue == "" {
			return fmt.Errorf("must specify non-empty value for required user-data variable %s", requiredVarName)
		}
	}
	return nil
}

// CutBetween returns the part of s between startingToken and endingToken. If startingToken and/or
// endingToken cannot be found in s, or if there are no characters between the two tokens, this
// returns an empty string (""). If there are multiple occurrances of each token, the largest possible
// part of s will be returned (i.e., everything between the leftmost startingToken and the rightmost
// endingToken, a.k.a. greedy matching)
func CutBetween(s string, startingToken string, endingToken string) string {
	escapedStartingToken := regexp.QuoteMeta(startingToken)
	escapedEndingToken := regexp.QuoteMeta(endingToken)
	reCutBetween := regexp.MustCompile(escapedStartingToken + `([\s\S]*)` + escapedEndingToken)
	matches := reCutBetween.FindStringSubmatch(s)
	// matches will be nil or a single-element slice if tokens are missing from str
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// DurationToBareSeconds tries to parse a given string as a duration (e.g., "1m30s") and return the
// total floating-point number of seconds in the duration. Failing that, it tries to return the left-
// most number in the given string. Failing that (or given an empty string, NaN, or infinity), it
// returns 0
func DurationToBareSeconds(possibleDurationStr string) float64 {
	// Return 0 for empty strings
	if strings.TrimSpace(possibleDurationStr) == "" {
		return 0
	}

	// Try to parse it as a time.Duration
	if parsedDuration, err := time.ParseDuration(possibleDurationStr); err == nil {
		// Success! So just return a floating point number of seconds
		return parsedDuration.Seconds()
	}

	// Try to just pull out any number
	reFloat := regexp.MustCompile(`-?\d+(\.\d*)?`)
	if reFloat.MatchString(possibleDurationStr) {
		f, _ := strconv.ParseFloat(reFloat.FindString(possibleDurationStr), 64)
		return f
	}

	// possibleDurationStr looks nothing like a duration: fall back to 0
	return 0
}
