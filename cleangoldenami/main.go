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
	"sync"
	"time"
)

const (
	serviceCode = "ec2"
	quotaCode   = "L-0E3CBAB9"
	timeLayout  = "2006-01-02T15:04:05.000Z"
	// desiredImageCapacity is the number of free "image slots" desired in each region
	desiredImageCapacity = 3
	ArchitectureARM64    = ec2Types.ArchitectureValuesArm64
	ArchitectureX86      = ec2Types.ArchitectureValuesX8664
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

	var wg sync.WaitGroup
	wg.Add(len(enabledRegions))

	for _, enabledRegion := range enabledRegions {
		go func(regionName string) {
			defer wg.Done()
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

			if imageCount := len(images); (quota - imageCount) >= desiredImageCapacity {
				if *verbose {
					fmt.Printf("Region %v is under quota. (Images: %v, Quota: %v)\n", regionName, imageCount, quota)
				}
			} else {
				numOfImagesToDelete := desiredImageCapacity - (quota - imageCount)

				arm64Images, legacyx86Images, x86Images := filterImages(images)

				var imagesToDelete []ec2Types.Image

				if canDeleteImages(numOfImagesToDelete, arm64Images, legacyx86Images, x86Images) {
					for len(imagesToDelete) < numOfImagesToDelete {
						// Returns the most populated image type and the slice containing images of that type
						mostPopulatedImagesByType, mostPopulatedImageType := getMostPopulatedImageType(arm64Images, legacyx86Images, x86Images)

						imageToDeleteIndex, imageToDelete, err := getOldestImage(mostPopulatedImagesByType)
						if err != nil {
							log.Fatalf("error determining which image to delete in region %v: %v", regionName, err)
						}

						// Add the image to the list of images to delete
						imagesToDelete = append(imagesToDelete, imageToDelete)

						// Remove the deleted image from its respective slice based on its type
						switch mostPopulatedImageType {
						case "arm64Images":
							arm64Images = append(arm64Images[:imageToDeleteIndex], arm64Images[imageToDeleteIndex+1:]...)
						case "legacyx86Images":
							legacyx86Images = append(legacyx86Images[:imageToDeleteIndex], legacyx86Images[imageToDeleteIndex+1:]...)
						case "x86Images":
							x86Images = append(x86Images[:imageToDeleteIndex], x86Images[imageToDeleteIndex+1:]...)
						}
					}
					for _, image := range imagesToDelete {
						for i, t := range image.Tags {
							if *t.Key == "version" {
								if !*dryRun {
									err = deregisterImage(ec2Client, image)
									if err != nil {
										fmt.Printf("error deregistering image %v (%v) in region %v: %v\n", *image.ImageId, *image.Tags[i].Value, regionName, err)
									}
									fmt.Printf("successfully deregistered image %v (%v) in region %v\n", *image.ImageId, *image.Tags[i].Value, regionName)
								} else {
									fmt.Printf("Region %v is at quota (%v) - would delete %v (%v)\n", regionName, quota, *image.ImageId, *image.Tags[i].Value)
								}
								break
							}
						}
					}
				} else {
					fmt.Printf("ERROR: MANUAL ACTION REQUIRED - Unable to delete images in region %v (Total Images: %v, rhel-arm64: %v, legacy-x86_64: %v, rhel-x86_64: %v, Quota: %v, ImagesToDelete: %v)\n",
						regionName, imageCount, len(arm64Images), len(legacyx86Images), len(x86Images), quota, numOfImagesToDelete)
				}
			}
		}(*enabledRegion.RegionName)
	}

	wg.Wait()
	fmt.Println("Done!")
}

// getEnabledRegions returns all enabled regions
func getEnabledRegions(describeRegionsClient DescribeRegionsClient) ([]ec2Types.Region, error) {
	describeRegionsResponse, err := describeRegionsClient.DescribeRegions(context.TODO(), nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching regions: %w", err)
	}
	return describeRegionsResponse.Regions, nil
}

// getPublicAMIServiceQuota returns the quota limit for AWS public images
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

// getMostPopulatedImageType returns the AWS AMIs with the highest availability of images
func getMostPopulatedImageType(arm64Images []ec2Types.Image, legacyx86Images []ec2Types.Image, x86Images []ec2Types.Image) ([]ec2Types.Image, string) {
	slices := []struct {
		name   string
		images []ec2Types.Image
	}{
		{name: "arm64Images", images: arm64Images},
		{name: "legacyx86Images", images: legacyx86Images},
		{name: "x86Images", images: x86Images},
	}

	var mostPopulatedImagesByType []ec2Types.Image
	var mostPopulatedImageType string
	longestSliceLength := 0

	for _, slice := range slices {
		if len(slice.images) > longestSliceLength {
			mostPopulatedImageType = slice.name
			mostPopulatedImagesByType = slice.images
			longestSliceLength = len(slice.images)
		}
	}

	return mostPopulatedImagesByType, mostPopulatedImageType
}

// getPublicImages retrieves all public images belonging to the AWS account
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

func hasVersionTag(image ec2Types.Image, tag string) bool {
	for _, t := range image.Tags {
		if *t.Key == "version" && *t.Value == tag {
			return true
		}
	}
	return false
}

func hasMatchingArchitecture(image ec2Types.Image, architecture ec2Types.ArchitectureValues) bool {
	return image.Architecture == architecture
}

// filterImages filters AWS AMIs based on their version tag and architecture.
func filterImages(images []ec2Types.Image) ([]ec2Types.Image, []ec2Types.Image, []ec2Types.Image) {
	var arm64Images, legacyx86Images, x86Images []ec2Types.Image
	for _, image := range images {

		if hasVersionTag(image, "rhel-arm64") && hasMatchingArchitecture(image, ArchitectureARM64) {
			arm64Images = append(arm64Images, image)
		} else if hasVersionTag(image, "legacy-x86_64") && hasMatchingArchitecture(image, ArchitectureX86) {
			legacyx86Images = append(legacyx86Images, image)
		} else if hasVersionTag(image, "rhel-x86_64") && hasMatchingArchitecture(image, ArchitectureX86) {
			x86Images = append(x86Images, image)
		}
	}
	return arm64Images, legacyx86Images, x86Images
}

// getOldestImage retrieves the oldest AWS AMI based on creation date
func getOldestImage(images []ec2Types.Image) (int, ec2Types.Image, error) {
	var oldestImage ec2Types.Image
	var oldestImageTimestamp int64
	var oldestImageIndex = -1

	for i, image := range images {
		creationTime, err := time.Parse(timeLayout, *image.CreationDate)
		if err != nil {
			return -1, ec2Types.Image{}, fmt.Errorf("error parsing timestamp %v for image %v: %w", *image.CreationDate, image.ImageId, err)
		}
		if creationTimeUnix := creationTime.Unix(); oldestImage.ImageId == nil || creationTimeUnix < oldestImageTimestamp {
			oldestImage = image
			oldestImageTimestamp = creationTimeUnix
			oldestImageIndex = i
		}
	}
	return oldestImageIndex, oldestImage, nil
}

// deregisterImage deregisters an AWS AMI
func deregisterImage(ec2Client DeregisterImageClient, image ec2Types.Image) error {
	_, err := ec2Client.DeregisterImage(context.TODO(), &ec2.DeregisterImageInput{ImageId: image.ImageId})
	if err != nil {
		return fmt.Errorf("error deregistering image: %w", err)
	}
	return nil
}

// canDeleteImages verifies it's safe to delete the required number of images from each of the provided image slices while ensuring that deleting images won't reduce any slice length below 1
func canDeleteImages(numOfImagesToDelete int, arm64Images []ec2Types.Image, legacyx86Images []ec2Types.Image, x86Images []ec2Types.Image) bool {
	// Calculate the maximum number of images that can be safely deleted from each slice
	canDeleteArm64 := max(0, len(arm64Images)-1)
	canDeleteLegacyX86 := max(0, len(legacyx86Images)-1)
	canDeleteX86 := max(0, len(x86Images)-1)

	// Calculate the total number of images that can be safely deleted across all slices
	totalCanDelete := canDeleteArm64 + canDeleteLegacyX86 + canDeleteX86

	return totalCanDelete >= numOfImagesToDelete
}

// Helper function to get the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
