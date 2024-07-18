package main

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas/types"
	"reflect"
	"testing"
)

type DescribeRegionsClientMock struct {
	regions []ec2Types.Region
	err     error
}

func (d DescribeRegionsClientMock) DescribeRegions(context.Context, *ec2.DescribeRegionsInput, ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error) {
	return &ec2.DescribeRegionsOutput{Regions: d.regions}, d.err
}

func Test_getEnabledRegions(t *testing.T) {
	tests := []struct {
		name                  string
		describeRegionsClient DescribeRegionsClient
		want                  []ec2Types.Region
		wantErr               bool
	}{
		{
			name:                  "error when fetching enabled regions",
			describeRegionsClient: DescribeRegionsClientMock{err: errors.New("fail")},
			wantErr:               true,
		},
		{
			name:                  "successfully returns no regions",
			describeRegionsClient: DescribeRegionsClientMock{regions: []ec2Types.Region{}},
			want:                  []ec2Types.Region{},
		},
		{
			name:                  "successfully returns one region",
			describeRegionsClient: DescribeRegionsClientMock{regions: []ec2Types.Region{{RegionName: aws.String("us-east-1")}}},
			want:                  []ec2Types.Region{{RegionName: aws.String("us-east-1")}},
		},
		{
			name: "successfully returns multiple regions",
			describeRegionsClient: DescribeRegionsClientMock{
				regions: []ec2Types.Region{
					{RegionName: aws.String("us-east-1")},
					{RegionName: aws.String("us-east-2")},
					{RegionName: aws.String("us-west-1")},
					{RegionName: aws.String("us-west-2")},
				},
			},
			want: []ec2Types.Region{
				{RegionName: aws.String("us-east-1")},
				{RegionName: aws.String("us-east-2")},
				{RegionName: aws.String("us-west-1")},
				{RegionName: aws.String("us-west-2")},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getEnabledRegions(tt.describeRegionsClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("getEnabledRegions() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getEnabledRegions() got = %v, want %v", got, tt.want)
			}
		})
	}
}

type GetServiceQuotaClientMock struct {
	quotaValue float64
	err        error
}

func (g GetServiceQuotaClientMock) GetServiceQuota(context.Context, *servicequotas.GetServiceQuotaInput, ...func(*servicequotas.Options)) (*servicequotas.GetServiceQuotaOutput, error) {
	return &servicequotas.GetServiceQuotaOutput{
		Quota: &types.ServiceQuota{
			Value: aws.Float64(g.quotaValue),
		},
	}, g.err
}

func Test_getPublicAMIServiceQuota(t *testing.T) {
	tests := []struct {
		name                string
		servicequotasClient GetServiceQuotaClient
		want                int
		wantErr             bool
	}{
		{
			name:                "error when fetching image quota",
			servicequotasClient: GetServiceQuotaClientMock{err: errors.New("fail")},
			wantErr:             true,
		},
		{
			name:                "successfully returns integer quota",
			servicequotasClient: GetServiceQuotaClientMock{quotaValue: 20},
			want:                20,
		},
		{
			name:                "successfully returns decimal quota",
			servicequotasClient: GetServiceQuotaClientMock{quotaValue: 13.2},
			want:                13,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPublicAMIServiceQuota(tt.servicequotasClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPublicAMIServiceQuota() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("getPublicAMIServiceQuota() got = %v, want %v", got, tt.want)
			}
		})
	}
}

type DescribeImagesClientMock struct {
	images []ec2Types.Image
	err    error
}

func (d DescribeImagesClientMock) DescribeImages(context.Context, *ec2.DescribeImagesInput, ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error) {
	return &ec2.DescribeImagesOutput{Images: d.images}, d.err
}

func Test_getPublicImages(t *testing.T) {
	tests := []struct {
		name      string
		ec2Client DescribeImagesClient
		want      []ec2Types.Image
		wantErr   bool
	}{
		{
			name:      "error when fetching images",
			ec2Client: DescribeImagesClientMock{err: errors.New("fail")},
			wantErr:   true,
		},
		{
			name:      "successfully return a single image",
			ec2Client: DescribeImagesClientMock{images: []ec2Types.Image{{ImageId: aws.String("abcd1234")}}},
			want:      []ec2Types.Image{{ImageId: aws.String("abcd1234")}},
		},
		{
			name:      "successfully returns no images",
			ec2Client: DescribeImagesClientMock{images: []ec2Types.Image{}},
			want:      []ec2Types.Image{},
		},
		{
			name: "successfully returns multiple images",
			ec2Client: DescribeImagesClientMock{
				images: []ec2Types.Image{
					{ImageId: aws.String("a")},
					{ImageId: aws.String("b")},
					{ImageId: aws.String("c")},
					{ImageId: aws.String("d")},
				},
			},
			want: []ec2Types.Image{
				{ImageId: aws.String("a")},
				{ImageId: aws.String("b")},
				{ImageId: aws.String("c")},
				{ImageId: aws.String("d")},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPublicImages(tt.ec2Client)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPublicImages() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getPublicImages() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sortImages(t *testing.T) {
	tests := []struct {
		name                            string
		images                          []ec2Types.Image
		architecture                    string
		tag                             string
		arm64ExpectedFilteredImages     []ec2Types.Image
		legacyx86ExpectedFilteredImages []ec2Types.Image
		x86ExpectedFilteredImages       []ec2Types.Image
		untaggedExpectedImages          []ec2Types.Image
	}{
		{
			name: "filter arm64 images by matching architecture and tag",
			images: []ec2Types.Image{
				{ImageId: aws.String("a"), Architecture: "arm64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("rhel-arm64")}}},
				{ImageId: aws.String("b"), Architecture: "x86_64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("legacy-x86_64")}}},
				{ImageId: aws.String("c"), Architecture: "x86_64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("rhel-x86_64")}}},
				{ImageId: aws.String("d"), Architecture: "arm64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("rhel-arm64")}}},
			},
			architecture: "arm64",
			tag:          "rhel-arm64",
			arm64ExpectedFilteredImages: []ec2Types.Image{
				{ImageId: aws.String("a"), Architecture: "arm64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("rhel-arm64")}}},
				{ImageId: aws.String("d"), Architecture: "arm64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("rhel-arm64")}}},
			},
			legacyx86ExpectedFilteredImages: []ec2Types.Image{
				{ImageId: aws.String("b"), Architecture: "x86_64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("legacy-x86_64")}}},
			},
			x86ExpectedFilteredImages: []ec2Types.Image{
				{ImageId: aws.String("c"), Architecture: "x86_64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("rhel-x86_64")}}},
			},
		},
		{
			name: "filter arm64 Images with missing tag",
			images: []ec2Types.Image{
				{ImageId: aws.String("a"), Architecture: "arm64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("")}}},
				{ImageId: aws.String("b"), Architecture: "x86_64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("legacy-x86_64")}}},
				{ImageId: aws.String("c"), Architecture: "x86_64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("rhel-x86_64")}}},
				{ImageId: aws.String("d"), Architecture: "arm64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("rhel-arm64")}}},
			},
			architecture: "arm64",
			tag:          "rhel-arm64",
			arm64ExpectedFilteredImages: []ec2Types.Image{
				{ImageId: aws.String("d"), Architecture: "arm64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("rhel-arm64")}}},
			},
			legacyx86ExpectedFilteredImages: []ec2Types.Image{
				{ImageId: aws.String("b"), Architecture: "x86_64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("legacy-x86_64")}}},
			},
			x86ExpectedFilteredImages: []ec2Types.Image{
				{ImageId: aws.String("c"), Architecture: "x86_64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("rhel-x86_64")}}},
			},
		},
		{
			name: "filters untagged images",
			images: []ec2Types.Image{
				{ImageId: aws.String("a"), Architecture: "arm64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("")}}},
				{ImageId: aws.String("b"), Architecture: "x86_64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("legacy-x86_64")}}},
				{ImageId: aws.String("c"), Architecture: "x86_64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("rhel-x86_64")}}},
				{ImageId: aws.String("d"), Architecture: "arm64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("rhel-arm64")}}},
				{ImageId: aws.String("e"), Architecture: "arm64"},
			},
			architecture: "arm64",
			tag:          "rhel-arm64",
			arm64ExpectedFilteredImages: []ec2Types.Image{
				{ImageId: aws.String("d"), Architecture: "arm64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("rhel-arm64")}}},
			},
			legacyx86ExpectedFilteredImages: []ec2Types.Image{
				{ImageId: aws.String("b"), Architecture: "x86_64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("legacy-x86_64")}}},
			},
			x86ExpectedFilteredImages: []ec2Types.Image{
				{ImageId: aws.String("c"), Architecture: "x86_64", Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("rhel-x86_64")}}},
			},
			untaggedExpectedImages: []ec2Types.Image{
				{ImageId: aws.String("e"), Architecture: "arm64"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arm64Images, legacyx86Images, x86Images, untaggedImages := sortImages(tt.images)
			if !reflect.DeepEqual(arm64Images, tt.arm64ExpectedFilteredImages) {
				t.Errorf("sortImages() got = %v, want %v", arm64Images, tt.arm64ExpectedFilteredImages)
			} else if !reflect.DeepEqual(legacyx86Images, tt.legacyx86ExpectedFilteredImages) {
				t.Errorf("sortImages() got = %v, want %v", legacyx86Images, tt.legacyx86ExpectedFilteredImages)
			} else if !reflect.DeepEqual(x86Images, tt.x86ExpectedFilteredImages) {
				t.Errorf("sortImages() got = %v, want %v", x86Images, tt.x86ExpectedFilteredImages)
			} else if !reflect.DeepEqual(untaggedImages, tt.untaggedExpectedImages) {

			}
		})
	}
}

func Test_canDeleteImages(t *testing.T) {
	testCases := []struct {
		name                string
		numOfImagesToDelete int
		arm64Images         []ec2Types.Image
		legacyx86Images     []ec2Types.Image
		x86Images           []ec2Types.Image
		expected            bool
	}{
		{
			name:                "Delete 3 images when rhel-arm64: 3, legacy-x86_64: 2, rhel-x86_64: 2",
			numOfImagesToDelete: 3,
			arm64Images: []ec2Types.Image{
				{ImageId: aws.String("a")},
				{ImageId: aws.String("b")},
				{ImageId: aws.String("c")}},
			legacyx86Images: []ec2Types.Image{
				{ImageId: aws.String("d")},
				{ImageId: aws.String("e")}},
			x86Images: []ec2Types.Image{
				{ImageId: aws.String("f")},
				{ImageId: aws.String("g")}},
			expected: true,
		},
		{
			name:                "Delete 2 images when rhel-arm64: 3, legacy-x86_64: 2, rhel-x86_64: 1",
			numOfImagesToDelete: 2,
			arm64Images: []ec2Types.Image{
				{ImageId: aws.String("a")},
				{ImageId: aws.String("b")},
				{ImageId: aws.String("c")}},
			legacyx86Images: []ec2Types.Image{
				{ImageId: aws.String("d")},
				{ImageId: aws.String("e")}},
			x86Images: []ec2Types.Image{
				{ImageId: aws.String("f")}},
			expected: true,
		},
		{
			name:                "Delete 1 image when rhel-arm64: 1, legacy-x86_64: 2, rhel-x86_64: 2",
			numOfImagesToDelete: 1,
			arm64Images:         []ec2Types.Image{},
			legacyx86Images: []ec2Types.Image{
				{ImageId: aws.String("a")},
				{ImageId: aws.String("b")}},
			x86Images: []ec2Types.Image{
				{ImageId: aws.String("c")},
				{ImageId: aws.String("d")}},
			expected: true,
		},
		{
			name:                "Delete 2 image when rhel-arm64: 0, legacy-x86_64: 1, rhel-x86_64: 1",
			numOfImagesToDelete: 2,
			arm64Images:         []ec2Types.Image{},
			legacyx86Images: []ec2Types.Image{
				{ImageId: aws.String("a")}},
			x86Images: []ec2Types.Image{
				{ImageId: aws.String("b")}},
			expected: false,
		},
		{
			name:                "Delete 3 image when rhel-arm64: 1, legacy-x86_64: 2, rhel-x86_64: 1",
			numOfImagesToDelete: 3,
			arm64Images: []ec2Types.Image{
				{ImageId: aws.String("a")}},
			legacyx86Images: []ec2Types.Image{
				{ImageId: aws.String("b")},
				{ImageId: aws.String("c")}},
			x86Images: []ec2Types.Image{
				{ImageId: aws.String("d")}},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := canDeleteImages(tc.numOfImagesToDelete, tc.arm64Images, tc.legacyx86Images, tc.x86Images)
			if result != tc.expected {
				t.Errorf("Expected %v but got %v for test case '%s'", tc.expected, result, tc.name)
			}
		})
	}
}

func Test_findImagesToDelete(t *testing.T) {
	tests := []struct {
		name              string
		numImagesToDelete int
		arm64Images       []ec2Types.Image
		legacyx86Images   []ec2Types.Image
		x86Images         []ec2Types.Image
		want              []ec2Types.Image
		wantErr           bool
	}{
		{
			name:              "0 images to delete",
			numImagesToDelete: 0,
		},
		{
			name:              "1 image to delete with arm64 most populated",
			numImagesToDelete: 1,
			arm64Images: []ec2Types.Image{
				{ImageId: aws.String("test1"), CreationDate: aws.String("2024-12-26T15:04:05.000Z")},
				{ImageId: aws.String("test2"), CreationDate: aws.String("2024-12-26T15:04:06.000Z")},
				{ImageId: aws.String("test3"), CreationDate: aws.String("2024-12-26T15:04:07.000Z")},
			},
			want: []ec2Types.Image{{ImageId: aws.String("test1"), CreationDate: aws.String("2024-12-26T15:04:05.000Z")}},
		},
		{
			name:              "1 image to delete with legacy-x86 most populated",
			numImagesToDelete: 1,
			legacyx86Images: []ec2Types.Image{
				{ImageId: aws.String("test1"), CreationDate: aws.String("2024-12-26T15:04:05.000Z")},
				{ImageId: aws.String("test2"), CreationDate: aws.String("2024-12-26T15:04:06.000Z")},
				{ImageId: aws.String("test3"), CreationDate: aws.String("2024-12-26T15:04:07.000Z")},
			},
			want: []ec2Types.Image{{ImageId: aws.String("test1"), CreationDate: aws.String("2024-12-26T15:04:05.000Z")}},
		},
		{
			name:              "1 image to delete with x86 most populated",
			numImagesToDelete: 1,
			x86Images: []ec2Types.Image{
				{ImageId: aws.String("test1"), CreationDate: aws.String("2024-12-26T15:04:05.000Z")},
				{ImageId: aws.String("test2"), CreationDate: aws.String("2024-12-26T15:04:06.000Z")},
				{ImageId: aws.String("test3"), CreationDate: aws.String("2024-12-26T15:04:07.000Z")},
			},
			want: []ec2Types.Image{{ImageId: aws.String("test1"), CreationDate: aws.String("2024-12-26T15:04:05.000Z")}},
		},
		{
			name:              "multiple images to delete",
			numImagesToDelete: 3,
			arm64Images: []ec2Types.Image{
				{ImageId: aws.String("arm1"), CreationDate: aws.String("2024-12-26T15:04:05.000Z")},
				{ImageId: aws.String("arm2"), CreationDate: aws.String("2024-12-26T15:04:06.000Z")},
				{ImageId: aws.String("arm3"), CreationDate: aws.String("2024-12-26T15:04:07.000Z")},
				{ImageId: aws.String("arm4"), CreationDate: aws.String("2024-12-26T15:04:08.000Z")},
				{ImageId: aws.String("arm5"), CreationDate: aws.String("2024-12-26T15:04:09.000Z")},
				{ImageId: aws.String("arm6"), CreationDate: aws.String("2024-12-26T15:04:10.000Z")},
			},
			legacyx86Images: []ec2Types.Image{
				{ImageId: aws.String("legacy1"), CreationDate: aws.String("2024-12-26T15:04:05.000Z")},
				{ImageId: aws.String("legacy2"), CreationDate: aws.String("2024-12-26T15:04:06.000Z")},
				{ImageId: aws.String("legacy3"), CreationDate: aws.String("2024-12-26T15:04:07.000Z")},
			},
			x86Images: []ec2Types.Image{
				{ImageId: aws.String("x861"), CreationDate: aws.String("2024-12-26T15:04:05.000Z")},
				{ImageId: aws.String("x862"), CreationDate: aws.String("2024-12-26T15:04:06.000Z")},
				{ImageId: aws.String("x863"), CreationDate: aws.String("2024-12-26T15:04:07.000Z")},
			},
			want: []ec2Types.Image{
				{ImageId: aws.String("arm1"), CreationDate: aws.String("2024-12-26T15:04:05.000Z")},
				{ImageId: aws.String("arm2"), CreationDate: aws.String("2024-12-26T15:04:06.000Z")},
				{ImageId: aws.String("arm3"), CreationDate: aws.String("2024-12-26T15:04:07.000Z")},
			},
		},
		{
			name:              "fewer images than number to delete",
			numImagesToDelete: 2,
			arm64Images:       []ec2Types.Image{{ImageId: aws.String("arm1")}},
			wantErr:           true,
		},
		{
			name:              "handles error when failing to find oldest image",
			numImagesToDelete: 1,
			arm64Images:       []ec2Types.Image{{ImageId: aws.String("arm1"), CreationDate: aws.String("not a time")}},
			wantErr:           true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findImagesToDelete(tt.numImagesToDelete, tt.arm64Images, tt.legacyx86Images, tt.x86Images)
			if (err != nil) != tt.wantErr {
				t.Errorf("findImagesToDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findImagesToDelete() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getMostPopulatedImageType(t *testing.T) {
	tests := []struct {
		name            string
		arm64Images     []ec2Types.Image
		legacyx86Images []ec2Types.Image
		x86Images       []ec2Types.Image
		wantSlice       []ec2Types.Image
		wantSliceName   string
	}{
		{
			name: "arm64Images has the longest slice",
			arm64Images: []ec2Types.Image{
				{ImageId: aws.String("a")},
				{ImageId: aws.String("b")},
			},
			legacyx86Images: []ec2Types.Image{
				{ImageId: aws.String("c")},
			},
			x86Images: []ec2Types.Image{
				{ImageId: aws.String("d")},
				{ImageId: aws.String("e")},
			},
			wantSlice:     []ec2Types.Image{{ImageId: aws.String("a")}, {ImageId: aws.String("b")}},
			wantSliceName: "arm64Images",
		},
		{
			name: "legacyx86Images has the longest slice",
			arm64Images: []ec2Types.Image{
				{ImageId: aws.String("a")},
			},
			legacyx86Images: []ec2Types.Image{
				{ImageId: aws.String("b")},
				{ImageId: aws.String("c")},
			},
			x86Images: []ec2Types.Image{
				{ImageId: aws.String("d")},
			},
			wantSlice:     []ec2Types.Image{{ImageId: aws.String("b")}, {ImageId: aws.String("c")}},
			wantSliceName: "legacyx86Images",
		},
		{
			name: "x86Images has the longest slice",
			arm64Images: []ec2Types.Image{
				{ImageId: aws.String("a")},
			},
			legacyx86Images: []ec2Types.Image{
				{ImageId: aws.String("b")},
			},
			x86Images: []ec2Types.Image{
				{ImageId: aws.String("c")},
				{ImageId: aws.String("d")},
				{ImageId: aws.String("e")},
			},
			wantSlice:     []ec2Types.Image{{ImageId: aws.String("c")}, {ImageId: aws.String("d")}, {ImageId: aws.String("e")}},
			wantSliceName: "x86Images",
		},
		{
			name: "all slices have the same length, should return arm64Images",
			arm64Images: []ec2Types.Image{
				{ImageId: aws.String("a")},
				{ImageId: aws.String("b")},
			},
			legacyx86Images: []ec2Types.Image{
				{ImageId: aws.String("c")},
				{ImageId: aws.String("d")},
			},
			x86Images: []ec2Types.Image{
				{ImageId: aws.String("e")},
				{ImageId: aws.String("f")},
			},
			wantSlice:     []ec2Types.Image{{ImageId: aws.String("a")}, {ImageId: aws.String("b")}},
			wantSliceName: "arm64Images",
		},
		{
			name:          "all slices are empty",
			wantSlice:     nil,
			wantSliceName: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSlice, gotSliceName := getMostPopulatedImageType(tt.arm64Images, tt.legacyx86Images, tt.x86Images)
			if !reflect.DeepEqual(gotSlice, tt.wantSlice) {
				t.Errorf("getMostPopulatedImageType() gotSlice = %v, want %v", gotSlice, tt.wantSlice)
			}
			if gotSliceName != tt.wantSliceName {
				t.Errorf("getMostPopulatedImageType() gotSliceName = %v, want %v", gotSliceName, tt.wantSliceName)
			}
		})
	}
}

func Test_getOldestImage(t *testing.T) {
	tests := []struct {
		name      string
		images    []ec2Types.Image
		wantIndex int
		wantImage ec2Types.Image
		wantErr   bool
	}{
		{
			name: "error when parsing time",
			images: []ec2Types.Image{
				{
					CreationDate: aws.String("not a time"),
				},
				{
					CreationDate: aws.String("not a time"),
				},
			},
			wantErr:   true,
			wantIndex: -1,
		},
		{
			name:      "successfully handles empty slice",
			wantIndex: -1,
		},
		{
			name: "successfully handles slice with multiple images",
			images: []ec2Types.Image{
				{ImageId: aws.String("a"), CreationDate: aws.String("2024-12-26T15:04:05.000Z")},
				{ImageId: aws.String("b"), CreationDate: aws.String("2024-04-27T15:04:05.000Z")},
				{ImageId: aws.String("c"), CreationDate: aws.String("2024-03-26T15:04:06.000Z")},
				{ImageId: aws.String("d"), CreationDate: aws.String("2024-03-26T15:04:05.000Z")},
			},
			wantIndex: 3,
			wantImage: ec2Types.Image{ImageId: aws.String("d"), CreationDate: aws.String("2024-03-26T15:04:05.000Z")},
		},
		{
			name: "takes first-found if two images have identical date",
			images: []ec2Types.Image{
				{ImageId: aws.String("a"), CreationDate: aws.String("2024-12-26T15:04:05.000Z")},
				{ImageId: aws.String("b"), CreationDate: aws.String("2024-12-26T15:04:05.000Z")},
			},
			wantIndex: 0,
			wantImage: ec2Types.Image{ImageId: aws.String("a"), CreationDate: aws.String("2024-12-26T15:04:05.000Z")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIndex, gotImage, err := getOldestImage(tt.images)
			if (err != nil) != tt.wantErr {
				t.Errorf("getOldestImage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotIndex != tt.wantIndex {
				t.Errorf("getOldestImage() gotIndex = %v, wantIndex %v", gotIndex, tt.wantIndex)
			}
			if !reflect.DeepEqual(gotImage, tt.wantImage) {
				t.Errorf("getOldestImage() gotImage = %v, wantImage %v", gotImage, tt.wantImage)
			}
		})
	}
}

type DeregisterImageClientMock struct {
	err error
}

func (d DeregisterImageClientMock) DeregisterImage(context.Context, *ec2.DeregisterImageInput, ...func(*ec2.Options)) (*ec2.DeregisterImageOutput, error) {
	return nil, d.err
}

func Test_deregisterImage(t *testing.T) {
	tests := []struct {
		name      string
		ec2Client DeregisterImageClientMock
		wantErr   bool
	}{
		{
			name:      "error when deregistering image",
			ec2Client: DeregisterImageClientMock{err: errors.New("fail")},
			wantErr:   true,
		},
		{
			name:      "successfully deleted image",
			ec2Client: DeregisterImageClientMock{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := deregisterImage(tt.ec2Client, ec2Types.Image{}); (err != nil) != tt.wantErr {
				t.Errorf("deregisterImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getImageVersionTag(t *testing.T) {
	tests := []struct {
		name  string
		image ec2Types.Image
		want  string
		ok    bool
	}{
		{
			name:  "no tags on image",
			image: ec2Types.Image{},
			want:  "",
			ok:    false,
		},
		{
			name:  "no version tag on image",
			image: ec2Types.Image{Tags: []ec2Types.Tag{{Key: aws.String("test")}}},
			want:  "",
			ok:    false,
		},
		{
			name:  "version tag with no value",
			image: ec2Types.Image{Tags: []ec2Types.Tag{{Key: aws.String("version")}}},
			want:  "",
			ok:    true,
		},
		{
			name:  "version tag with value",
			image: ec2Types.Image{Tags: []ec2Types.Tag{{Key: aws.String("version"), Value: aws.String("testVersion")}}},
			want:  "testVersion",
			ok:    true,
		},
		{
			name: "multiple tags on image",
			image: ec2Types.Image{
				Tags: []ec2Types.Tag{
					{Key: aws.String("test"), Value: aws.String("testValue")},
					{Key: aws.String("test2"), Value: aws.String("testValue2")},
					{Key: aws.String("version"), Value: aws.String("testVersion")},
				},
			},
			want: "testVersion",
			ok:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := getImageVersionTag(tt.image)
			if ok != tt.ok {
				t.Errorf("getImageVersionTag() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("getImageVersionTag() got = %v, want %v", got, tt.want)
			}
		})
	}
}
