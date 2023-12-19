package helpers

import (
	_ "embed"
	"errors"
	"math/rand"
	"time"

	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
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

	for idx, ipRange := range a.IpRanges {
		if *ipRange.CidrIp != *b.IpRanges[idx].CidrIp {
			return false
		}
	}

	for idx, ipv6Range := range a.Ipv6Ranges {
		if *ipv6Range.CidrIpv6 != *b.Ipv6Ranges[idx].CidrIpv6 {
			return false
		}
	}

	return true
}

// Enumerated type representing the platform underlying the cluster-under-test
const (
	PlatformAWS           string = "aws"
	PlatformGCP           string = "gcp"
	PlatformHostedCluster string = "hostedcluster"
)
