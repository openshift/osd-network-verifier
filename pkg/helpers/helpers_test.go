package helpers

import (
	_ "embed"
	"testing"

	awsTools "github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestIPPermissionsEquivalent(t *testing.T) {
	type args struct {
		a ec2Types.IpPermission
		b ec2Types.IpPermission
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "identical",
			args: args{
				a: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/0"),
							Description: awsTools.String("foo"),
						},
					},
				},
				b: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/0"),
							Description: awsTools.String("foo"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "equivalent diff descriptions",
			args: args{
				a: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/0"),
							Description: awsTools.String("foo"),
						},
					},
				},
				b: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/0"),
							Description: awsTools.String("bar"),
						},
					},
				},
			},
			want: true,
		},
		{
			name: "not equivalent port",
			args: args{
				a: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(8080),
					ToPort:     awsTools.Int32(8080),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/0"),
							Description: awsTools.String("foo"),
						},
					},
				},
				b: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/0"),
							Description: awsTools.String("foo"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "not equivalent cidr",
			args: args{
				a: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.1.3/32"),
							Description: awsTools.String("foo"),
						},
					},
				},
				b: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/0"),
							Description: awsTools.String("foo"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "not equivalent range len",
			args: args{
				a: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/32"),
							Description: awsTools.String("foo"),
						},
						{
							CidrIp:      awsTools.String("0.0.1.3/32"),
							Description: awsTools.String("bar"),
						},
					},
				},
				b: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awsTools.String("0.0.0.0/32"),
							Description: awsTools.String("foo"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "not equivalent v6",
			args: args{
				a: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					Ipv6Ranges: []ec2Types.Ipv6Range{
						{
							CidrIpv6:    awsTools.String("ff06::c3/128"),
							Description: awsTools.String("foo"),
						},
					},
				},
				b: ec2Types.IpPermission{
					FromPort:   awsTools.Int32(80),
					ToPort:     awsTools.Int32(80),
					IpProtocol: awsTools.String("tcp"),
					Ipv6Ranges: []ec2Types.Ipv6Range{
						{
							CidrIpv6:    awsTools.String("ff06::c5/128"),
							Description: awsTools.String("foo"),
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IPPermissionsEquivalent(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("IPPermissionsEquivalent() = %v, want %v", got, tt.want)
			}
		})
	}
}
