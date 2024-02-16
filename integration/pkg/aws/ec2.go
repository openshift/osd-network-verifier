package aws

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type byovpcEc2Api interface {
	DescribeAvailabilityZones(ctx context.Context, params *ec2.DescribeAvailabilityZonesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAvailabilityZonesOutput, error)
	CreateVpc(ctx context.Context, params *ec2.CreateVpcInput, optFns ...func(*ec2.Options)) (*ec2.CreateVpcOutput, error)
	DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	CreateTags(ctx context.Context, params *ec2.CreateTagsInput, optFns ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
	ModifyVpcAttribute(ctx context.Context, params *ec2.ModifyVpcAttributeInput, optFns ...func(*ec2.Options)) (*ec2.ModifyVpcAttributeOutput, error)
	CreateSubnet(ctx context.Context, params *ec2.CreateSubnetInput, optFns ...func(*ec2.Options)) (*ec2.CreateSubnetOutput, error)
	DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	AssociateRouteTable(ctx context.Context, params *ec2.AssociateRouteTableInput, optFns ...func(*ec2.Options)) (*ec2.AssociateRouteTableOutput, error)
	CreateRouteTable(ctx context.Context, params *ec2.CreateRouteTableInput, optFns ...func(*ec2.Options)) (*ec2.CreateRouteTableOutput, error)
	DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
	CreateInternetGateway(ctx context.Context, params *ec2.CreateInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.CreateInternetGatewayOutput, error)
	AttachInternetGateway(ctx context.Context, params *ec2.AttachInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.AttachInternetGatewayOutput, error)
	DescribeInternetGateways(ctx context.Context, params *ec2.DescribeInternetGatewaysInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInternetGatewaysOutput, error)
	CreateRoute(ctx context.Context, params *ec2.CreateRouteInput, optFns ...func(*ec2.Options)) (*ec2.CreateRouteOutput, error)
	AllocateAddress(ctx context.Context, params *ec2.AllocateAddressInput, optFns ...func(*ec2.Options)) (*ec2.AllocateAddressOutput, error)
	DescribeAddresses(ctx context.Context, params *ec2.DescribeAddressesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error)
	CreateNatGateway(ctx context.Context, params *ec2.CreateNatGatewayInput, optFns ...func(*ec2.Options)) (*ec2.CreateNatGatewayOutput, error)
	DescribeNatGateways(context.Context, *ec2.DescribeNatGatewaysInput, ...func(*ec2.Options)) (*ec2.DescribeNatGatewaysOutput, error)

	DeleteRoute(ctx context.Context, params *ec2.DeleteRouteInput, optFns ...func(*ec2.Options)) (*ec2.DeleteRouteOutput, error)
	DeleteNatGateway(ctx context.Context, params *ec2.DeleteNatGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteNatGatewayOutput, error)
	ReleaseAddress(ctx context.Context, params *ec2.ReleaseAddressInput, optFns ...func(*ec2.Options)) (*ec2.ReleaseAddressOutput, error)
	DetachInternetGateway(ctx context.Context, params *ec2.DetachInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DetachInternetGatewayOutput, error)
	DeleteInternetGateway(ctx context.Context, params *ec2.DeleteInternetGatewayInput, optFns ...func(*ec2.Options)) (*ec2.DeleteInternetGatewayOutput, error)
	DisassociateRouteTable(ctx context.Context, params *ec2.DisassociateRouteTableInput, optFns ...func(*ec2.Options)) (*ec2.DisassociateRouteTableOutput, error)
	DeleteRouteTable(ctx context.Context, params *ec2.DeleteRouteTableInput, optFns ...func(*ec2.Options)) (*ec2.DeleteRouteTableOutput, error)
	DeleteSubnet(ctx context.Context, params *ec2.DeleteSubnetInput, optFns ...func(*ec2.Options)) (*ec2.DeleteSubnetOutput, error)
	DeleteVpc(ctx context.Context, params *ec2.DeleteVpcInput, optFns ...func(*ec2.Options)) (*ec2.DeleteVpcOutput, error)

	DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(options *ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
	DeleteSecurityGroup(ctx context.Context, params *ec2.DeleteSecurityGroupInput, optFns ...func(options *ec2.Options)) (*ec2.DeleteSecurityGroupOutput, error)
}

func (id *OnvIntegrationTestData) createAndWaitForSubnet(ctx context.Context, name string, input *ec2.CreateSubnetInput) (*string, error) {
	subnet, err := id.ec2Api.CreateSubnet(ctx, input)
	if err != nil {
		return nil, err
	}

	if subnet.Subnet == nil {
		// Shouldn't happen
		return nil, errors.New("unexpected error, empty subnet response")
	}

	log.Printf("waiting up to %s for subnet to become available", 60*time.Second)
	subnetWaiter := ec2.NewSubnetAvailableWaiter(id.ec2Api)
	if err := subnetWaiter.Wait(ctx, &ec2.DescribeSubnetsInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("subnet-id"),
				Values: []string{*subnet.Subnet.SubnetId},
			},
		},
	}, 60*time.Second); err != nil {
		return nil, err
	}
	log.Printf("subnet available: %s", *subnet.Subnet.SubnetId)

	return subnet.Subnet.SubnetId, nil
}

func (id *OnvIntegrationTestData) createAndAssociateRouteTable(ctx context.Context, subnetId *string, name string) (*string, *string, error) {
	if id.vpcId == nil {
		return nil, nil, errors.New("vpc id must not be nil when creating route tables")
	}

	rt, err := id.ec2Api.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
		VpcId: id.vpcId,
		TagSpecifications: []ec2Types.TagSpecification{
			{
				ResourceType: ec2Types.ResourceTypeRouteTable,
				Tags:         defaultEc2Tags(),
			},
		},
	})
	if err != nil {
		return nil, nil, err
	}
	log.Printf("created %s subnet route table: %s", name, *rt.RouteTable.RouteTableId)

	rtAssoc, err := id.ec2Api.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
		RouteTableId: rt.RouteTable.RouteTableId,
		SubnetId:     subnetId,
	})
	if err != nil {
		return nil, nil, err
	}
	log.Printf("associated %s subnet route table: %s", name, *rtAssoc.AssociationId)

	return rt.RouteTable.RouteTableId, rtAssoc.AssociationId, nil
}

func (id *OnvIntegrationTestData) setupPrivateRouteTable(ctx context.Context) error {
	if id.privateSubnetId == nil {
		return errors.New("private subnet id must not be nil when creating private subnet route table")
	}

	var err error
	id.privateSubnetRouteTableId, id.privateSubnetRouteTableAssociationId, err = id.createAndAssociateRouteTable(ctx, id.privateSubnetId, "private")
	return err
}

func (id *OnvIntegrationTestData) setupPublicRouteTable(ctx context.Context) error {
	if id.publicSubnetId == nil {
		return errors.New("public subnet id must not be nil when creating public subnet route table")
	}

	var err error
	id.publicSubnetRouteTableId, id.publicSubnetRouteTableAssociationId, err = id.createAndAssociateRouteTable(ctx, id.publicSubnetId, "public")
	return err
}

func (id *OnvIntegrationTestData) setupFirewallRouteTable(ctx context.Context) error {
	if id.firewallSubnetId == nil {
		return errors.New("firewall subnet id must not be nil when creating firewall subnet route table")
	}

	var err error
	id.firewallSubnetRouteTableId, id.firewallSubnetRouteTableAssociationId, err = id.createAndAssociateRouteTable(ctx, id.firewallSubnetId, "firewall")
	return err
}
