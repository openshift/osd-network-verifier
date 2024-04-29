package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"log"
	"os"
	"time"
)

const (
	serviceCode = "ec2"
	quotaCode   = "L-0E3CBAB9"
	timeLayout  = "2006-01-02T15:04:05.000Z"
)

func main() {
	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	dryRun := f.Bool("dry-run", false, "When specified, show which AMIs would be deregistered without actually deregistering them.")
	verbose := f.Bool("verbose", false, "When specified, explicitly states which regions are not at their quota.")
	if err := f.Parse(os.Args[1:]); err != nil {
		panic(err)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	ec2RegionClient := ec2.NewFromConfig(cfg)
	enabledRegions, err := getEnabledRegions(ec2RegionClient)
	if err != nil {
		log.Fatalf("error fetching enabled regions for AWS account: %v", err)
	}

	for _, enabledRegion := range enabledRegions {
		regionName := *enabledRegion.RegionName
		servicequotaClient := servicequotas.NewFromConfig(cfg, func(o *servicequotas.Options) { o.Region = regionName })
		quota, err := getPublicAMIServiceQuota(servicequotaClient)
		if err != nil {
			log.Fatalf("error fetching image quota for region %v: %v", regionName, err)
		}

		ec2Client := ec2.NewFromConfig(cfg, func(o *ec2.Options) { o.Region = regionName })
		images, err := getPublicImages(ec2Client)
		if err != nil {
			log.Fatalf("error fetching images for region %v: %v", regionName, err)
		}

		if imageCount := len(images); imageCount < quota {
			if *verbose {
				fmt.Printf("Region %v is under quota. (Images: %v, Quota: %v)\n", regionName, imageCount, quota)
			}
		} else {
			imageToDelete, err := getOldestImage(images)
			if err != nil {
				log.Fatalf("error determining which image to delete in region %v: %v", regionName, err)
			}
			if *dryRun {
				fmt.Printf("Region %v is at quota (%v) - would delete %v\n", regionName, quota, *imageToDelete.ImageId)
			} else {
				err := deregisterImage(ec2Client, imageToDelete)
				if err != nil {
					log.Fatalf("error deregistering image %v in region %v: %v", *imageToDelete.ImageId, regionName, err)
				}
				fmt.Printf("successfully deregistered image %v in region %v", *imageToDelete.ImageId, regionName)
			}
		}
	}

	fmt.Println("Done!")
}

func getEnabledRegions(describeRegionsClient DescribeRegionsClient) ([]ec2Types.Region, error) {
	describeRegionsResponse, err := describeRegionsClient.DescribeRegions(context.TODO(), nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching regions: %w", err)
	}
	return describeRegionsResponse.Regions, nil
}

func getPublicAMIServiceQuota(servicequotasClient GetServiceQuotaClient) (int, error) {
	getServiceQuotaResponse, err := servicequotasClient.GetServiceQuota(context.TODO(), &servicequotas.GetServiceQuotaInput{
		ServiceCode: aws.String(serviceCode),
		QuotaCode:   aws.String(quotaCode),
	})
	if err != nil {
		return 0, fmt.Errorf("error fetching image quota: %w", err)
	}
	serviceQuotaValue := int(*getServiceQuotaResponse.Quota.Value)
	return serviceQuotaValue, nil
}

func getPublicImages(ec2Client DescribeImagesClient) ([]ec2Types.Image, error) {
	describeImagesResponse, err := ec2Client.DescribeImages(context.TODO(), &ec2.DescribeImagesInput{
		ExecutableUsers: []string{"all"},
		Owners:          []string{"self"},
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching images: %w", err)
	}
	return describeImagesResponse.Images, nil
}

func getOldestImage(images []ec2Types.Image) (ec2Types.Image, error) {
	var oldestImage ec2Types.Image
	var oldestImageTimestamp int64
	for _, image := range images {
		creationTime, err := time.Parse(timeLayout, *image.CreationDate)
		if err != nil {
			return ec2Types.Image{}, fmt.Errorf("error parsing timestamp %v for image %v: %w", *image.CreationDate, image.ImageId, err)
		}
		if creationTimeUnix := creationTime.Unix(); oldestImage.ImageId == nil || creationTimeUnix < oldestImageTimestamp {
			oldestImage = image
			oldestImageTimestamp = creationTimeUnix
		}
	}
	return oldestImage, nil
}

func deregisterImage(ec2Client DeregisterImageClient, image ec2Types.Image) error {
	_, err := ec2Client.DeregisterImage(context.TODO(), &ec2.DeregisterImageInput{ImageId: image.ImageId})
	if err != nil {
		return fmt.Errorf("error deregistering image: %w", err)
	}
	return nil
}
