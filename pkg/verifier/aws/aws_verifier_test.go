package awsverifier

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	awss "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	ocmlog "github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/openshift/osd-network-verifier/pkg/clients/aws"
	"github.com/openshift/osd-network-verifier/pkg/data/cpu"
	"github.com/openshift/osd-network-verifier/pkg/helpers"
	"github.com/openshift/osd-network-verifier/pkg/mocks"
	"github.com/openshift/osd-network-verifier/pkg/probes/legacy"
	gomock "go.uber.org/mock/gomock"
)

func TestFindUnreachableEndpointsSuccessWithLegacyProbe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	FakeEC2Cli := mocks.NewMockEC2Client(ctrl)
	out := &ec2.GetConsoleOutputOutput{
		InstanceId: awss.String("dummy-instance"),
		// USERDATA BEGIN
		// Using IMAGE : e4d93a35c482
		// Validating route53domains.us-east-1.amazonaws.com:443 ...
		// Success!
		// USERDATA END
		Output: awss.String(`VVNFUkRBVEEgQkVHSU4KVXNpbmcgSU1BR0UgOiBlNGQ5M2EzNWM0ODIKVmFsaWRhdGluZyByb3V0ZTUzZG9tYWlucy51cy1lYXN0LTEuYW1hem9uYXdzLmNvbTo0NDMKVmFsaWRhdGluZyByZWdpc3RyeS5yZWRoYXQuaW86NDQzClZhbGlkYXRpbmcgcXVheS5pbzo0NDMKVmFsaWRhdGluZyBzc28ucmVkaGF0LmNvbTo0NDMKVmFsaWRhdGluZyBzc28ucmVkaGF0LmNvbTo4MApWYWxpZGF0aW5nIHB1bGwucTF3Mi5xdWF5LnJoY2xvdWQuY29tOjQ0MwpWYWxpZGF0aW5nIG9wZW5zaGlmdC5vcmc6NDQzClZhbGlkYXRpbmcgY29uc29sZS5yZWRoYXQuY29tOjQ0MwpWYWxpZGF0aW5nIGNvbnNvbGUucmVkaGF0LmNvbTo4MApWYWxpZGF0aW5nIHF1YXktcmVnaXN0cnkuczMuYW1hem9uYXdzLmNvbTo0NDMKVmFsaWRhdGluZyBjZXJ0LWFwaS5hY2Nlc3MucmVkaGF0LmNvbTo0NDMKVmFsaWRhdGluZyBhcGkuYWNjZXNzLnJlZGhhdC5jb206NDQzClZhbGlkYXRpbmcgaW5mb2d3LmFwaS5vcGVuc2hpZnQuY29tOjQ0MwpWYWxpZGF0aW5nIG1pcnJvci5vcGVuc2hpZnQuY29tOjQ0MwpWYWxpZGF0aW5nIHN0b3JhZ2UuZ29vZ2xlYXBpcy5jb206NDQzClZhbGlkYXRpbmcgYXBpLm9wZW5zaGlmdC5jb206NDQzClZhbGlkYXRpbmcgY2FydC1yaGNvcy1jaS5zMy5hbWF6b25hd3MuY29tOjQ0MwpWYWxpZGF0aW5nIHJlZ2lzdHJ5LmFjY2Vzcy5yZWRoYXQuY29tOjQ0MwpWYWxpZGF0aW5nIGNtLXF1YXktcHJvZHVjdGlvbi1zMy5zMy5hbWF6b25hd3MuY29tOjQ0MwpWYWxpZGF0aW5nIGVjMi5hbWF6b25hd3MuY29tOjQ0MwpWYWxpZGF0aW5nIGlhbS5hbWF6b25hd3MuY29tOjQ0MwpWYWxpZGF0aW5nIHJvdXRlNTMuYW1hem9uYXdzLmNvbTo0NDMKVmFsaWRhdGluZyBzdHMuYW1hem9uYXdzLmNvbTo0NDMKVmFsaWRhdGluZyBldmVudHMucGFnZXJkdXR5LmNvbTo0NDMKVmFsaWRhdGluZyBhcGkuZGVhZG1hbnNzbml0Y2guY29tOjQ0MwpWYWxpZGF0aW5nIG5vc25jaC5pbjo0NDMKVmFsaWRhdGluZyBpbnB1dHMxLm9zZHNlY3VyaXR5bG9ncy5zcGx1bmtjbG91ZC5jb206OTk5NwpWYWxpZGF0aW5nIGh0dHAtaW5wdXRzLW9zZHNlY3VyaXR5bG9ncy5zcGx1bmtjbG91ZC5jb206NDQzClZhbGlkYXRpbmcgb2JzZXJ2YXRvcml1bS5hcGkub3BlbnNoaWZ0LmNvbTo0NDMKVmFsaWRhdGluZyBlYzIuZXUtd2VzdC0xLmFtYXpvbmF3cy5jb206NDQzClZhbGlkYXRpbmcgZWxhc3RpY2xvYWRiYWxhbmNpbmcuZXUtd2VzdC0xLmFtYXpvbmF3cy5jb206NDQzClZhbGlkYXRpbmcgZXZlbnRzLmV1LXdlc3QtMS5hbWF6b25hd3MuY29tOjQ0MwpWYWxpZGF0aW5nIHRhZ2dpbmcudXMtZWFzdC0xLmFtYXpvbmF3cy5jb206NDQzClN1Y2Nlc3MhClVTRVJEQVRBIEVORAo=`),
	}
	FakeEC2Cli.EXPECT().GetConsoleOutput(gomock.Any(), gomock.Any()).Times(1).Return(out, nil)

	cli := AwsVerifier{
		AwsClient: &aws.Client{
			Region: "us-east-1",
		},
	}

	cli.AwsClient.SetClient(FakeEC2Cli)
	cli.Logger = &ocmlog.GlogLogger{}

	err := cli.findUnreachableEndpoints(context.TODO(), "dummy-instance", legacy.Probe{})
	if err != nil {
		t.Errorf("err should be nil when there's success in output, got: %v", err)
	}
}

func TestFindUnreachableEndpointsNoSuccessWithLegacyProbe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	FakeEC2Cli := mocks.NewMockEC2Client(ctrl)
	out := &ec2.GetConsoleOutputOutput{
		InstanceId: awss.String("dummy-instance"),
		// USERDATA BEGIN
		// Using IMAGE : e4d93a35c482
		// Validating route53domains.us-east-1.amazonaws.com:443
		// USERDATA END
		Output: awss.String(`VVNFUkRBVEEgQkVHSU4KVXNpbmcgSU1BR0UgOiBlNGQ5M2EzNWM0ODIKVmFsaWRhdGluZyByb3V0ZTUzZG9tYWlucy51cy1lYXN0LTEuYW1hem9uYXdzLmNvbTo0NDMKVVNFUkRBVEEgRU5ECg==`),
	}
	FakeEC2Cli.EXPECT().GetConsoleOutput(gomock.Any(), gomock.Any()).Times(1).Return(out, nil)

	cli := AwsVerifier{
		AwsClient: &aws.Client{
			Region: "us-east-1",
		},
	}

	cli.AwsClient.SetClient(FakeEC2Cli)
	cli.Logger = &ocmlog.GlogLogger{}

	err := cli.findUnreachableEndpoints(context.TODO(), "dummy-instance", legacy.Probe{})
	if err != nil {
		t.Errorf("Success! not found, but userdata end exists, err should be nil, got: %v", err)
	}

	if !cli.Output.IsSuccessful() {
		t.Errorf("Success! not found, userdata end exists but no regex match for failure, it means success, got : %v", cli.Output)
	}
}

func TestIpPermissionFromURL(t *testing.T) {
	type args struct {
		urlStr      string
		description string
	}
	tests := []struct {
		name    string
		args    args
		want    *ec2Types.IpPermission
		wantErr bool
	}{
		{
			name: "IPv4 happy path",
			args: args{
				urlStr:      "http://1.2.3.4:567",
				description: "test4",
			},
			want: &ec2Types.IpPermission{
				FromPort:   awss.Int32(567),
				ToPort:     awss.Int32(567),
				IpProtocol: awss.String("tcp"),
				IpRanges: []ec2Types.IpRange{
					{
						CidrIp:      awss.String("1.2.3.4/32"),
						Description: awss.String("test4"),
					},
				},
			},
		},
		{
			name: "IPv6 happy path",
			args: args{
				urlStr:      "http://[ff06::c3]:567",
				description: "test6",
			},
			want: &ec2Types.IpPermission{
				FromPort:   awss.Int32(567),
				ToPort:     awss.Int32(567),
				IpProtocol: awss.String("tcp"),
				Ipv6Ranges: []ec2Types.Ipv6Range{
					{
						CidrIpv6:    awss.String("ff06::c3/128"),
						Description: awss.String("test6"),
					},
				},
			},
		},
		{
			name: "Inferred port",
			args: args{
				urlStr:      "https://10.0.8.8",
				description: "testi",
			},
			want: &ec2Types.IpPermission{
				FromPort:   awss.Int32(443),
				ToPort:     awss.Int32(443),
				IpProtocol: awss.String("tcp"),
				IpRanges: []ec2Types.IpRange{
					{
						CidrIp:      awss.String("10.0.8.8/32"),
						Description: awss.String("testi"),
					},
				},
			},
		},
		{
			name: "Good https fqdn",
			args: args{
				urlStr:      "https://example.fqdn.test.com",
				description: "test-fqdn",
			},
			want: &ec2Types.IpPermission{
				FromPort:   awss.Int32(443),
				ToPort:     awss.Int32(443),
				IpProtocol: awss.String("tcp"),
				IpRanges: []ec2Types.IpRange{
					{
						CidrIp:      awss.String("0.0.0.0/0"),
						Description: awss.String("test-fqdn"),
					},
				},
			},
		},
		{
			name: "Good http fqdn",
			args: args{
				urlStr:      "http://example.fqdn.test.com",
				description: "test-fqdn2",
			},
			want: &ec2Types.IpPermission{
				FromPort:   awss.Int32(80),
				ToPort:     awss.Int32(80),
				IpProtocol: awss.String("tcp"),
				IpRanges: []ec2Types.IpRange{
					{
						CidrIp:      awss.String("0.0.0.0/0"),
						Description: awss.String("test-fqdn2"),
					},
				},
			},
		},
		{
			name: "Good fqdn with port",
			args: args{
				urlStr:      "http://example.fqdn.test.com:7654",
				description: "test-fqdn3",
			},
			want: &ec2Types.IpPermission{
				FromPort:   awss.Int32(7654),
				ToPort:     awss.Int32(7654),
				IpProtocol: awss.String("tcp"),
				IpRanges: []ec2Types.IpRange{
					{
						CidrIp:      awss.String("0.0.0.0/0"),
						Description: awss.String("test-fqdn3"),
					},
				},
			},
		},
		{
			name: "Bad fqdn",
			args: args{
				urlStr:      "http://example.b>d.fqdn.test.com:8080",
				description: "teste",
			},
			wantErr: true,
		},
		{
			name: "Missing URL scheme",
			args: args{
				urlStr:      "example.bad.fqdn.test.com:8080",
				description: "teste",
			},
			wantErr: true,
		},
		{
			name: "Error on inferring non-http(s) scheme",
			args: args{
				urlStr:      "ssh://example.com",
				description: "teste",
			},
			wantErr: true,
		},
		{
			name: "Error on bad URL",
			args: args{
				urlStr:      "not a URL",
				description: "teste",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ipPermissionFromURL(tt.args.urlStr, tt.args.description)
			if (err != nil) != tt.wantErr {
				t.Errorf("ipPermissionFromURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ipPermissionFromURL() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_ipPermissionSetFromURLs(t *testing.T) {
	type args struct {
		urlStrs           []string
		descriptionPrefix string
	}
	tests := []struct {
		name    string
		args    args
		want    []ec2Types.IpPermission
		wantErr bool
	}{
		{
			name: "single-URL happy path",
			args: args{
				urlStrs:           []string{"http://1.2.3.4:567"},
				descriptionPrefix: "single test: ",
			},
			want: []ec2Types.IpPermission{
				{
					FromPort:   awss.Int32(567),
					ToPort:     awss.Int32(567),
					IpProtocol: awss.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awss.String("1.2.3.4/32"),
							Description: awss.String("single test: http://1.2.3.4:567"),
						},
					},
				},
			},
		},
		{
			name: "multiple unique URL happy path",
			args: args{
				urlStrs:           []string{"http://1.2.3.4:567", "https://8.9.10.11:1213"},
				descriptionPrefix: "multi-unique test: ",
			},
			want: []ec2Types.IpPermission{
				{
					FromPort:   awss.Int32(567),
					ToPort:     awss.Int32(567),
					IpProtocol: awss.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awss.String("1.2.3.4/32"),
							Description: awss.String("multi-unique test: http://1.2.3.4:567"),
						},
					},
				},
				{
					FromPort:   awss.Int32(1213),
					ToPort:     awss.Int32(1213),
					IpProtocol: awss.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awss.String("8.9.10.11/32"),
							Description: awss.String("multi-unique test: https://8.9.10.11:1213"),
						},
					},
				},
			},
		},
		{
			name: "multiple equivalent URLs",
			args: args{
				urlStrs:           []string{"http://1.2.3.4:567", "https://1.2.3.4:567"},
				descriptionPrefix: "multi-equivalent test: ",
			},
			want: []ec2Types.IpPermission{
				{
					FromPort:   awss.Int32(567),
					ToPort:     awss.Int32(567),
					IpProtocol: awss.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awss.String("1.2.3.4/32"),
							Description: awss.String("multi-equivalent test: http://1.2.3.4:567"),
						},
					},
				},
			},
		},
		{
			name: "multiple identical URLs",
			args: args{
				urlStrs:           []string{"http://1.2.3.4:567", "http://1.2.3.4:567"},
				descriptionPrefix: "multi-identical test: ",
			},
			want: []ec2Types.IpPermission{
				{
					FromPort:   awss.Int32(567),
					ToPort:     awss.Int32(567),
					IpProtocol: awss.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awss.String("1.2.3.4/32"),
							Description: awss.String("multi-identical test: http://1.2.3.4:567"),
						},
					},
				},
			},
		},
		{
			name: "multiple domain URLs",
			args: args{
				urlStrs:           []string{"http://proxy.example.org:567", "https://proxy.example.org:567"},
				descriptionPrefix: "multi-identical test: ",
			},
			want: []ec2Types.IpPermission{
				{
					FromPort:   awss.Int32(567),
					ToPort:     awss.Int32(567),
					IpProtocol: awss.String("tcp"),
					IpRanges: []ec2Types.IpRange{
						{
							CidrIp:      awss.String("0.0.0.0/0"),
							Description: awss.String("multi-identical test: http://proxy.example.org:567"),
						},
					},
				},
			},
		},
		{
			name: "domain URLs overlapping with default SG set",
			args: args{
				urlStrs:           []string{"http://proxy.example.org:80", "https://proxy.example.org:443"},
				descriptionPrefix: "multi-identical test: ",
			},
			want: []ec2Types.IpPermission{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ipPermissionSetFromURLs(tt.args.urlStrs, tt.args.descriptionPrefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("ipPermissionSetFromURLs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ipPermissionSetFromURLs() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestAwsVerifier_selectInstanceType uses a mock EC2 API client to test the logic used for selecting
// an instance type and CPU architecture based on user inputs and programmed defaults
func TestAwsVerifier_selectInstanceType(t *testing.T) {
	x86DefaultInstanceType, _ := cpu.ArchX86.DefaultInstanceType(helpers.PlatformAWS)
	armDefaultInstanceType, _ := cpu.ArchARM.DefaultInstanceType(helpers.PlatformAWS)

	type MockInstanceInfo struct {
		CPUArchitecture ec2Types.ArchitectureType
		Hypervisor      ec2Types.InstanceTypeHypervisor
	}

	tests := []struct {
		name              string
		inputInstanceType string
		inputCPUArch      cpu.Architecture
		// mockInstanceInfo defines the answer that the mock EC2 API will give to any calls to
		// ec2.DescribeInstanceTypes(). Leaving this nil means we don't expect any API calls in
		// this test case
		mockInstanceInfo   *MockInstanceInfo
		expectInstanceType string
		expectCPUArch      cpu.Architecture
		expectErr          bool
	}{
		{
			name:               "nothing requested",
			expectInstanceType: x86DefaultInstanceType,
			expectCPUArch:      cpu.ArchX86,
		},
		{
			name:               "X86 requested",
			expectInstanceType: x86DefaultInstanceType,
			expectCPUArch:      cpu.ArchX86,
		},
		{
			name:               "ARM requested",
			inputCPUArch:       cpu.ArchARM,
			expectInstanceType: armDefaultInstanceType,
			expectCPUArch:      cpu.ArchARM,
		},
		{
			name:               "valid X86 type requested",
			inputInstanceType:  "t3.nano",
			mockInstanceInfo:   &MockInstanceInfo{ec2Types.ArchitectureTypeX8664, ec2Types.InstanceTypeHypervisorNitro},
			expectInstanceType: "t3.nano",
			expectCPUArch:      cpu.ArchX86,
		},
		{
			name:               "valid ARM type requested",
			inputInstanceType:  "t4g.nano",
			mockInstanceInfo:   &MockInstanceInfo{ec2Types.ArchitectureTypeArm64, ec2Types.InstanceTypeHypervisorNitro},
			expectInstanceType: "t4g.nano",
			expectCPUArch:      cpu.ArchARM,
		},
		{
			name:               "non-Nitro type requested",
			inputInstanceType:  "c4.large",
			mockInstanceInfo:   &MockInstanceInfo{ec2Types.ArchitectureTypeX8664, ec2Types.InstanceTypeHypervisorXen},
			expectInstanceType: x86DefaultInstanceType,
			expectCPUArch:      cpu.ArchX86,
		},
		{
			name:              "Nitro type with unsupported CPU requested",
			inputInstanceType: "mac1.metal",
			mockInstanceInfo:  &MockInstanceInfo{ec2Types.ArchitectureTypeX8664Mac, ec2Types.InstanceTypeHypervisorNitro},
			expectErr:         true,
		},
		{
			name:              "invalid type requested",
			inputInstanceType: "foobar",
			expectErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AwsVerifier{
				Logger:    &ocmlog.GlogLogger{},
				AwsClient: &aws.Client{},
			}

			// Set up mock EC2 API client
			mockController := gomock.NewController(t)
			defer mockController.Finish()
			mockEC2Client := mocks.NewMockEC2Client(mockController)

			// Only expect API calls if an instance type was requested and needs to be validated
			if tt.inputInstanceType != "" {
				if tt.mockInstanceInfo != nil {
					// Mock EC2 API returns a valid answer with mocked instance info
					expectedDescribeInstanceTypesInput := &ec2.DescribeInstanceTypesInput{
						InstanceTypes: []ec2Types.InstanceType{ec2Types.InstanceType(tt.inputInstanceType)},
					}
					mockDescribeInstanceTypesOutput := &ec2.DescribeInstanceTypesOutput{
						InstanceTypes: []ec2Types.InstanceTypeInfo{
							{
								Hypervisor:   tt.mockInstanceInfo.Hypervisor,
								InstanceType: ec2Types.InstanceType(tt.inputInstanceType),
								ProcessorInfo: &ec2Types.ProcessorInfo{
									SupportedArchitectures: []ec2Types.ArchitectureType{tt.mockInstanceInfo.CPUArchitecture},
								},
							},
						},
					}
					mockEC2Client.EXPECT().DescribeInstanceTypes(gomock.Any(), expectedDescribeInstanceTypesInput).Times(1).Return(
						mockDescribeInstanceTypesOutput,
						nil,
					)
				} else {
					// Mock EC2 API always answers DescribeInstanceTypes() calls with "not valid"
					mockEC2Client.EXPECT().DescribeInstanceTypes(gomock.Any(), gomock.Any()).Times(1).Return(
						nil,
						fmt.Errorf("(MOCK) not a valid instance type"),
					)
				}
			}
			a.AwsClient.SetClient(mockEC2Client)

			gotInstanceType, gotCPUArch, err := a.selectInstanceType(context.TODO(), tt.inputInstanceType, tt.inputCPUArch)
			if (err != nil) != tt.expectErr {
				t.Errorf("AwsVerifier.selectInstanceType() error = %v, wantErr %v", err, tt.expectErr)
				return
			}
			if gotInstanceType != tt.expectInstanceType {
				t.Errorf("AwsVerifier.selectInstanceType() gotInstanceType = %v, want %v", gotInstanceType, tt.expectInstanceType)
			}
			if gotCPUArch != tt.expectCPUArch {
				t.Errorf("AwsVerifier.selectInstanceType() gotCPUArch = %v, want %v", gotCPUArch, tt.expectCPUArch)
			}
		})
	}
}
