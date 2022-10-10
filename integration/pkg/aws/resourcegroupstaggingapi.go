package aws

import (
	"context"

	tags "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
)

type byovpcTagsApi interface {
	TagResources(ctx context.Context, params *tags.TagResourcesInput, optFns ...func(*tags.Options)) (*tags.TagResourcesOutput, error)
}
