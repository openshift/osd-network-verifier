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
				return
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
				return
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
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getPublicImages() got = %v, want %v", got, tt.want)
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
			name: "successfully handles slice with one image",
			images: []ec2Types.Image{
				{ImageId: aws.String("a"), CreationDate: aws.String("2024-04-26T15:04:05.000Z")},
			},
			wantIndex: -1,
			wantImage: ec2Types.Image{},
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIndex, gotImage, err := getOldestImage(tt.images)
			if (err != nil) != tt.wantErr {
				t.Errorf("getOldestImage() error = %v, wantErr %v", err, tt.wantErr)
				return
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

func Test_imageDeletionCheck(t *testing.T) {
	tests := []struct {
		name                string
		imagesToDelete      []ec2Types.Image
		numOfImagesToDelete int
		arm64Images         []ec2Types.Image
		legacyx86Images     []ec2Types.Image
		x86Images           []ec2Types.Image
		want                bool
	}{
		{
			name: "imagesToDelete length matches numOfImagesToDelete",
			imagesToDelete: []ec2Types.Image{
				{ImageId: aws.String("a")},
				{ImageId: aws.String("b")},
				{ImageId: aws.String("c")},
			},
			numOfImagesToDelete: 3,
			want:                true,
		},
		{
			name: "one of the architecture slices has <= 1 image",
			arm64Images: []ec2Types.Image{
				{ImageId: aws.String("a")},
			},
			legacyx86Images: []ec2Types.Image{
				{ImageId: aws.String("b")},
			},
			x86Images: []ec2Types.Image{
				{ImageId: aws.String("c")},
			},
			want: true,
		},
		{
			name: "none of the conditions are met",
			imagesToDelete: []ec2Types.Image{
				{ImageId: aws.String("a")},
				{ImageId: aws.String("b")},
			},
			numOfImagesToDelete: 3,
			arm64Images: []ec2Types.Image{
				{ImageId: aws.String("c")},
				{ImageId: aws.String("d")},
			},
			legacyx86Images: []ec2Types.Image{
				{ImageId: aws.String("d")},
				{ImageId: aws.String("e")},
			},
			x86Images: []ec2Types.Image{
				{ImageId: aws.String("f")},
				{ImageId: aws.String("g")},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := imageDeletionCheck(tt.imagesToDelete, tt.numOfImagesToDelete, tt.arm64Images, tt.legacyx86Images, tt.x86Images)
			if got != tt.want {
				t.Errorf("imageDeletionCheck() got = %v, want %v", got, tt.want)
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
