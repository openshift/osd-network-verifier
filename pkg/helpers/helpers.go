package helpers

import (
	_ "embed"
	"errors"
	"math/rand"
	"regexp"
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

//go:embed config/userdata.yaml
var UserdataTemplate string

// RandSeq generates random string with n characters.
func RandSeq(n int) string {
	rand.Seed(time.Now().UnixNano())
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
