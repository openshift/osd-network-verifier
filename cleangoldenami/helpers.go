package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"time"
)

const (
	serviceCode = "ec2"
	quotaCode   = "L-0E3CBAB9"

	timeLayout = "2006-01-02T15:04:05.000Z"

	arm64ImagesLabel     = "arm64Images"
	legacyX86ImagesLabel = "legacyx86Images"
	x86ImagesLabel       = "x86Images"
)

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

// sortImages sorts AWS AMIs based on their version tag and architecture.
func sortImages(images []ec2Types.Image) (arm64Images []ec2Types.Image, legacyx86Images []ec2Types.Image, x86Images []ec2Types.Image, untaggedImages []ec2Types.Image) {
	for _, image := range images {
		versionTag, ok := getImageVersionTag(image)
		if ok {
			if versionTag == "rhel-arm64" && image.Architecture == ec2Types.ArchitectureValuesArm64 {
				arm64Images = append(arm64Images, image)
			} else if versionTag == "legacy-x86_64" && image.Architecture == ec2Types.ArchitectureValuesX8664 {
				legacyx86Images = append(legacyx86Images, image)
			} else if versionTag == "rhel-x86_64" && image.Architecture == ec2Types.ArchitectureValuesX8664 {
				x86Images = append(x86Images, image)
			} else {
				untaggedImages = append(untaggedImages, image)
			}
		} else {
			untaggedImages = append(untaggedImages, image)
		}
	}
	return
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

// findImagesToDelete determines which images from the passed in lists should be deleted.
// This function determines how many total images need to be deleted
// and prefers deleting from the most populated list first.
func findImagesToDelete(numImagesToDelete int, arm64Images, legacyx86Images, x86Images []ec2Types.Image) ([]ec2Types.Image, error) {
	var imagesToDelete []ec2Types.Image
	if totalImages := len(arm64Images) + len(legacyx86Images) + len(x86Images); numImagesToDelete > totalImages {
		return nil, fmt.Errorf("requested to delete %d images but there are only %d images present", numImagesToDelete, totalImages)
	}
	for len(imagesToDelete) < numImagesToDelete {
		mostPopulatedImagesByType, mostPopulatedImageType := getMostPopulatedImageType(arm64Images, legacyx86Images, x86Images)

		imageToDeleteIndex, imageToDelete, err := getOldestImage(mostPopulatedImagesByType)
		if err != nil {
			return nil, fmt.Errorf("error determining oldest image: %w", err)
		}

		imagesToDelete = append(imagesToDelete, imageToDelete)

		switch mostPopulatedImageType {
		case arm64ImagesLabel:
			arm64Images = append(arm64Images[:imageToDeleteIndex], arm64Images[imageToDeleteIndex+1:]...)
		case legacyX86ImagesLabel:
			legacyx86Images = append(legacyx86Images[:imageToDeleteIndex], legacyx86Images[imageToDeleteIndex+1:]...)
		case x86ImagesLabel:
			x86Images = append(x86Images[:imageToDeleteIndex], x86Images[imageToDeleteIndex+1:]...)
		}
	}
	return imagesToDelete, nil
}

// getMostPopulatedImageType returns the AWS AMIs with the highest availability of images
func getMostPopulatedImageType(arm64Images []ec2Types.Image, legacyx86Images []ec2Types.Image, x86Images []ec2Types.Image) ([]ec2Types.Image, string) {
	slices := []struct {
		name   string
		images []ec2Types.Image
	}{
		{name: arm64ImagesLabel, images: arm64Images},
		{name: legacyX86ImagesLabel, images: legacyx86Images},
		{name: x86ImagesLabel, images: x86Images},
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

// deregisterImage de-registers an AWS AMI
func deregisterImage(ec2Client DeregisterImageClient, image ec2Types.Image) error {
	_, err := ec2Client.DeregisterImage(context.TODO(), &ec2.DeregisterImageInput{ImageId: image.ImageId})
	if err != nil {
		return fmt.Errorf("error deregistering image: %w", err)
	}
	return nil
}

// getImageVersionTag returns the value of the `version` tag for a given image
func getImageVersionTag(image ec2Types.Image) (string, bool) {
	for _, t := range image.Tags {
		if *t.Key == "version" {
			if t.Value == nil {
				return "", true
			}
			return *t.Value, true
		}
	}
	return "", false
}

// processResults reads from the results channels and presents them as output
func processResults(expectedRegionResultCount int, regionResultChan <-chan regionResult, deregisterImageResultChan <-chan deregisterImageResult, verbose bool) {
	amiResultsRemaining := 0
	for expectedRegionResultCount > 0 || amiResultsRemaining > 0 {
		select {
		case result := <-regionResultChan:
			expectedRegionResultCount--
			amiResultsRemaining += result.requestCount
			if result.err != nil {
				fmt.Println(result.err)
			} else if verbose {
				fmt.Println(result.result)
			}
		case result := <-deregisterImageResultChan:
			amiResultsRemaining--
			printResult(result)
		}
	}
}

func printResult(result deregisterImageResult) {
	var resultOutput string
	if result.dryRun {
		resultOutput = fmt.Sprintf("would deregister image %v (%v)", result.imageId, result.version)
	} else {
		if result.err != nil {
			resultOutput = fmt.Sprintf("error deregistering image %v (%v): %v", result.imageId, result.version, result.err)
		} else {
			resultOutput = fmt.Sprintf("successfully deregistered image %v (%v)", result.imageId, result.version)
		}
	}

	fmt.Printf("Region %v is at %d out of %d images: %s\n", result.region, result.currentImages, result.quota, resultOutput)
}
