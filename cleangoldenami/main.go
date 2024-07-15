package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"log"
	"os"
)

// desiredImageCapacity is the number of free "image slots" desired in each region
const desiredImageCapacity = 3

type deregisterImageResult struct {
	region        string
	currentImages int
	quota         int
	imageId       string
	version       string
	dryRun        bool
	err           error
}

type regionResult struct {
	result       string
	err          error
	requestCount int
}

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

	deregisterImageResultChan := make(chan deregisterImageResult)
	regionResultChan := make(chan regionResult)
	for _, enabledRegion := range enabledRegions {
		go func(regionName string) {
			servicequotaClient := servicequotas.NewFromConfig(cfg, func(o *servicequotas.Options) { o.Region = regionName })
			quota, err := getPublicAMIServiceQuota(servicequotaClient)
			if err != nil {
				regionResultChan <- regionResult{err: fmt.Errorf("error fetching image quota for region %v: %w", regionName, err)}
				return
			}

			ec2Client := ec2.NewFromConfig(cfg, func(o *ec2.Options) { o.Region = regionName })
			images, err := getPublicImages(ec2Client)
			if err != nil {
				regionResultChan <- regionResult{err: fmt.Errorf("error fetching images for region %v: %w", regionName, err)}
				return
			}

			imageCount := len(images)
			if (quota - imageCount) >= desiredImageCapacity {
				regionResultChan <- regionResult{result: fmt.Sprintf("Region %s is under quota. (Images: %d, Quota: %d)", regionName, imageCount, quota)}
				return
			}

			numOfImagesToDelete := desiredImageCapacity - (quota - imageCount)
			arm64Images, legacyx86Images, x86Images, untaggedImages := sortImages(images)
			// TODO Ideally, untagged images should always be empty.
			// TODO In the future we should either:
			// TODO - delete untagged images first
			// TODO - log the untagged images somewhere
			if !canDeleteImages(numOfImagesToDelete, arm64Images, legacyx86Images, x86Images) {
				regionResultChan <- regionResult{err: fmt.Errorf("ERROR: MANUAL ACTION REQUIRED - Unable to delete images in region %s (Total Images: %d, rhel-arm64: %d, legacy-x86_64: %d, rhel-x86_64: %d, untagged: %d, Quota: %d, NumImagesToDelete: %d)",
					regionName, imageCount, len(arm64Images), len(legacyx86Images), len(x86Images), len(untaggedImages), quota, numOfImagesToDelete)}
				return
			}

			imagesToDelete, err := findImagesToDelete(numOfImagesToDelete, arm64Images, legacyx86Images, x86Images)
			if err != nil {
				regionResultChan <- regionResult{err: fmt.Errorf("error determining images to delete in region %s: %w", regionName, err)}
			}

			for index, image := range imagesToDelete {
				go func(img ec2Types.Image, imageIndex int) {
					var deregisterImageErr error
					if !*dryRun {
						deregisterImageErr = deregisterImage(ec2Client, img)
					}
					version, ok := getImageVersionTag(image)
					deregisterResult := deregisterImageResult{
						region:        regionName,
						currentImages: len(images),
						quota:         quota,
						imageId:       *img.ImageId,
						dryRun:        *dryRun,
						err:           deregisterImageErr,
					}
					if ok {
						deregisterResult.version = version
					} else {
						deregisterResult.version = "unknown version"
					}
					deregisterImageResultChan <- deregisterResult
				}(image, index)
			}

			regionResultChan <- regionResult{result: fmt.Sprintf("successfully processed region %v", regionName), requestCount: len(imagesToDelete)}
		}(*enabledRegion.RegionName)
	}

	processResults(len(enabledRegions), regionResultChan, deregisterImageResultChan, *verbose)

	fmt.Println("Done!")
}
