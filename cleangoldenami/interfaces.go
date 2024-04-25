package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
)

type DescribeRegionsClient interface {
	DescribeRegions(ctx context.Context, params *ec2.DescribeRegionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRegionsOutput, error)
}

type GetServiceQuotaClient interface {
	GetServiceQuota(ctx context.Context, params *servicequotas.GetServiceQuotaInput, optFns ...func(*servicequotas.Options)) (*servicequotas.GetServiceQuotaOutput, error)
}

type DescribeImagesClient interface {
	DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
}

type DeregisterImageClient interface {
	DeregisterImage(ctx context.Context, params *ec2.DeregisterImageInput, optFns ...func(*ec2.Options)) (*ec2.DeregisterImageOutput, error)
}
