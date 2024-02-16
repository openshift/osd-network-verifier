package aws

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	nfwTypes "github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"
	tags "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
)

const (
	// These are temporary because CIDR math is hard
	// A /25 is the minimum CIDR for a single-AZ OpenShift cluster
	vpcCidr               = "10.0.0.0/23"
	privateSubnetCidr     = "10.0.0.0/25"
	firewallSubnetCidr    = "10.0.0.128/25"
	publicSubnetCidr      = "10.0.1.0/25"
	firewallName          = "osd-network-verifier-firewall"
	firewallPolicyName    = "osd-network-verifier-firewall-policy"
	firewallRuleGroupName = "osd-network-verifier-rule-group"
)

type OnvIntegrationTestData struct {
	ec2Api             byovpcEc2Api
	networkFirewallApi byovpcNetworkFirewallApi
	tagsApi            byovpcTagsApi
	region             string
	cidrBlock          *string

	availabilityZoneName                   *string
	vpcId                                  *string
	privateSubnetId                        *string
	privateSubnetRouteTableId              *string
	privateSubnetRouteTableAssociationId   *string
	publicSubnetId                         *string
	publicSubnetRouteTableId               *string
	publicSubnetRouteTableAssociationId    *string
	internetGatewayId                      *string
	internetGatewayRouteTableId            *string
	internetGatewayRouteTableAssociationId *string
	eipAllocationId                        *string
	natGatewayId                           *string
	firewallSubnetId                       *string
	firewallSubnetRouteTableId             *string
	firewallSubnetRouteTableAssociationId  *string
	firewallVpcEndpointId                  *string
	firewallRuleGroupArn                   *string
	firewallPolicyArn                      *string
	firewallArn                            *string
}

func NewIntegrationTestData(cfg aws.Config) *OnvIntegrationTestData {
	log.Printf("using region %s", cfg.Region)

	return &OnvIntegrationTestData{
		ec2Api:             ec2.NewFromConfig(cfg),
		networkFirewallApi: networkfirewall.NewFromConfig(cfg),
		tagsApi:            tags.NewFromConfig(cfg),
		region:             cfg.Region,
		cidrBlock:          aws.String(vpcCidr),
	}
}

type processAwsResourceFunc func(ctx context.Context) error

func (id *OnvIntegrationTestData) processAwsResources(ctx context.Context, processFuncs []processAwsResourceFunc) error {
	for _, processFunc := range processFuncs {
		if err := processFunc(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (id *OnvIntegrationTestData) Setup(ctx context.Context) error {
	if err := id.processAwsResources(ctx, []processAwsResourceFunc{
		id.SetupAvailabilityZone,
		id.SetupVpc,
		id.SetupSubnets,
		id.SetupFirewall,
		id.SetupNatGateway,
		id.SetupInternetGateway,
		id.SetupRouteTables,
	}); err != nil {
		return err
	}

	return nil
}

func (id *OnvIntegrationTestData) Cleanup(ctx context.Context) error {
	if err := id.processAwsResources(ctx, []processAwsResourceFunc{
		id.CleanupRouteTables,
		id.CleanupFirewall,
		id.CleanupFirewallPolicy,
		id.CleanupRuleGroup,
		id.CleanupNatGateway,
		id.CleanupElasticIp,
		id.CleanupInternetGateway,
		id.CleanupSubnets,
		id.CleanupSecurityGroup,
		id.CleanupVpc,
	}); err != nil {
		return err
	}

	return nil
}

// GetPrivateSubnetId returns the value of privateSubnetId stored in the struct
func (id *OnvIntegrationTestData) GetPrivateSubnetId() *string {
	return id.privateSubnetId
}

// defaultEc2Tags returns the list of all default tags for created EC2 resources
func defaultEc2Tags() []ec2Types.Tag {
	return []ec2Types.Tag{
		{
			Key:   aws.String("owned"),
			Value: aws.String("red-hat-managed"),
		},
		{
			Key:   aws.String("purpose"),
			Value: aws.String("osd-network-verifier-integration-test"),
		},
	}
}

func defaultEc2TagFilters() []ec2Types.Filter {
	defaultTags := defaultEc2Tags()
	filters := make([]ec2Types.Filter, len(defaultTags))

	for i, tag := range defaultEc2Tags() {
		filters[i] = ec2Types.Filter{
			Name:   aws.String(fmt.Sprintf("tag:%s", *tag.Key)),
			Values: []string{*tag.Value},
		}
	}

	return filters
}

// defaultNetworkFirewallTags returns the list of all default tags for created network firewall resources
func defaultNetworkFirewallTags() []nfwTypes.Tag {
	return []nfwTypes.Tag{
		{
			Key:   aws.String("owned"),
			Value: aws.String("red-hat-managed"),
		},
		{
			Key:   aws.String("purpose"),
			Value: aws.String("osd-network-verifier-integration-test"),
		},
	}
}
