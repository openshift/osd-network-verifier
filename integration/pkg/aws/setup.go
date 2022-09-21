package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	nfwTypes "github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"
)

// TODO: remove
const slug = "-abcd"

// SetupAvailabilityZone chooses a random available AZ (not requiring opt-in) from the selected region
func (id *OnvIntegrationTestData) SetupAvailabilityZone(ctx context.Context) error {
	azs, err := id.ec2Api.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{
		AllAvailabilityZones: aws.Bool(false),
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("region-name"),
				Values: []string{id.region},
			},
		},
	})
	if err != nil {
		return err
	}

	if len(azs.AvailabilityZones) == 0 {
		return fmt.Errorf("no available AZs found for region: %s", id.region)
	}

	if azs.AvailabilityZones[0].ZoneName == nil {
		// Shouldn't happen
		return errors.New("unexpected error: nil AZ name")
	}

	id.availabilityZoneName = azs.AvailabilityZones[0].ZoneName
	log.Printf("using AZ: %s", *id.availabilityZoneName)

	return nil
}

// SetupVpc creates a VPC and enables DNS support and hostname resolution
func (id *OnvIntegrationTestData) SetupVpc(ctx context.Context) error {
	vpc, err := id.ec2Api.CreateVpc(ctx, &ec2.CreateVpcInput{CidrBlock: id.cidrBlock})
	if err != nil {
		return err
	}

	id.vpcId = vpc.Vpc.VpcId
	log.Printf("created VPC: %s", *id.vpcId)

	if _, err := id.ec2Api.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{*id.vpcId},
		Tags:      defaultEc2Tags("onv-integration-test-vpc"),
	}); err != nil {
		return err
	}

	// Can only modify one attribute at a time
	// EnableDnsSupport is a prerequisite for EnableDnsHostnames
	if _, err := id.ec2Api.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
		VpcId:            id.vpcId,
		EnableDnsSupport: &ec2Types.AttributeBooleanValue{Value: aws.Bool(true)},
	}); err != nil {
		return err
	}
	log.Printf("enableDnsSupport is true for VPC: %s", *id.vpcId)

	if _, err := id.ec2Api.ModifyVpcAttribute(ctx, &ec2.ModifyVpcAttributeInput{
		VpcId:              id.vpcId,
		EnableDnsHostnames: &ec2Types.AttributeBooleanValue{Value: aws.Bool(true)},
	}); err != nil {
		return err
	}
	log.Printf("enableDnsHostnames is true for VPC: %s", *id.vpcId)

	return nil
}

// SetupSubnets creates a public/firewall/private subnets
func (id *OnvIntegrationTestData) SetupSubnets(ctx context.Context) error {
	if id.vpcId == nil {
		return errors.New("vpc ids must not be nil when creating subnets")
	}

	if id.availabilityZoneName == nil {
		return errors.New("availability zone id must not be nil when creating subnets")
	}

	log.Printf("creating private subnet: %s", privateSubnetCidr)
	privateSubnetId, err := id.createAndWaitForSubnet(ctx, "onv-integration-test-private-subnet",
		&ec2.CreateSubnetInput{
			VpcId:            id.vpcId,
			AvailabilityZone: id.availabilityZoneName,
			CidrBlock:        aws.String(privateSubnetCidr),
		})
	if err != nil {
		return err
	}

	id.privateSubnetId = privateSubnetId
	log.Printf("created private subnet: %s", *id.privateSubnetId)

	log.Printf("creating firewall subnet: %s", firewallSubnetCidr)
	firewallSubnetId, err := id.createAndWaitForSubnet(ctx, "onv-integration-test-firewall-subnet",
		&ec2.CreateSubnetInput{
			VpcId:            id.vpcId,
			AvailabilityZone: id.availabilityZoneName,
			CidrBlock:        aws.String(firewallSubnetCidr),
		})
	if err != nil {
		return err
	}

	id.firewallSubnetId = firewallSubnetId
	log.Printf("created firewall subnet: %s", *id.firewallSubnetId)

	log.Printf("creating public subnet: %s", publicSubnetCidr)
	publicSubnetId, err := id.createAndWaitForSubnet(ctx, "onv-integration-test-public-subnet",
		&ec2.CreateSubnetInput{
			VpcId:            id.vpcId,
			AvailabilityZone: id.availabilityZoneName,
			CidrBlock:        aws.String(publicSubnetCidr),
		})
	if err != nil {
		return err
	}

	id.publicSubnetId = publicSubnetId
	log.Printf("created public subnet: %s", *id.publicSubnetId)

	return nil
}

// SetupRouteTables creates two route tables and associates them with the public/private subnets
// https://docs.aws.amazon.com/network-firewall/latest/developerguide/arch-igw-ngw.html
func (id *OnvIntegrationTestData) SetupRouteTables(ctx context.Context) error {
	if err := id.setupPrivateRouteTable(ctx); err != nil {
		return err
	}

	if err := id.setupPublicRouteTable(ctx); err != nil {
		return err
	}

	if err := id.setupFirewallRouteTable(ctx); err != nil {
		return err
	}

	// Ensure the IGW is attached before setting up the firewall subnet route table
	log.Printf("waiting up to %s for internet gateway attachment to become available", 30*time.Second)
	igwWaiter := ec2.NewInternetGatewayExistsWaiter(id.ec2Api)
	if err := igwWaiter.Wait(ctx, &ec2.DescribeInternetGatewaysInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("internet-gateway-id"),
				Values: []string{*id.internetGatewayId},
			},
			{
				Name:   aws.String("attachment.vpc-id"),
				Values: []string{*id.vpcId},
			},
			{
				Name:   aws.String("attachment.state"),
				Values: []string{"available"},
			},
		},
	}, 30*time.Second); err != nil {
		return err
	}
	log.Printf("internet gateway attached: %s", *id.internetGatewayId)

	if _, err := id.ec2Api.CreateRoute(ctx, &ec2.CreateRouteInput{
		RouteTableId:         id.firewallSubnetRouteTableId,
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            id.internetGatewayId,
	}); err != nil {
		return err
	}
	log.Println("route created for firewall subnet 0.0.0.0/0 --> IGW")

	igwRouteTable, err := id.ec2Api.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
		VpcId: id.vpcId,
	})
	if err != nil {
		return err
	}
	id.internetGatewayRouteTableId = igwRouteTable.RouteTable.RouteTableId
	log.Printf("created internet gateway route table: %s", *id.firewallSubnetRouteTableId)

	if _, err := id.ec2Api.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{*id.internetGatewayRouteTableId},
		Tags:      defaultEc2Tags("onv-integration-test-igw"),
	}); err != nil {
		return err
	}

	igwAssoc, err := id.ec2Api.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
		RouteTableId: id.internetGatewayRouteTableId,
		GatewayId:    id.internetGatewayId,
	})
	if err != nil {
		return err
	}
	id.internetGatewayRouteTableAssociationId = igwAssoc.AssociationId
	log.Printf("associated internet gateway route table: %s", *id.internetGatewayRouteTableAssociationId)

	// Ensure the NAT Gateway is available before setting up the private subnet route table
	log.Printf("waiting up to %s for the NAT Gateway to become available", 5*time.Minute)
	natGwWaiter := ec2.NewNatGatewayAvailableWaiter(id.ec2Api)
	if err := natGwWaiter.Wait(ctx, &ec2.DescribeNatGatewaysInput{
		NatGatewayIds: []string{*id.natGatewayId},
	}, 5*time.Minute); err != nil {
		return err
	}
	log.Printf("NAT gateway available: %s", *id.natGatewayId)

	if _, err := id.ec2Api.CreateRoute(ctx, &ec2.CreateRouteInput{
		RouteTableId:         id.privateSubnetRouteTableId,
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		NatGatewayId:         id.natGatewayId,
	}); err != nil {
		return err
	}
	log.Println("route created for private subnet 0.0.0.0/0 --> NAT")

	// Ensure the Firewall is available before setting up the public subnet route table
	log.Printf("waiting up to %s for the firewall to become READY", 10*time.Minute)
	firewallWaiter := NewFirewallReadyWaiter(id.networkFirewallApi)
	if err := firewallWaiter.Wait(ctx, &networkfirewall.DescribeFirewallInput{
		FirewallArn: id.firewallArn,
	}, 10*time.Minute); err != nil {
		return err
	}
	log.Printf("firewall ready: %s", *id.firewallArn)

	firewall, err := id.networkFirewallApi.DescribeFirewall(ctx, &networkfirewall.DescribeFirewallInput{FirewallArn: id.firewallArn})
	if err != nil {
		return err
	}

	if _, ok := firewall.FirewallStatus.SyncStates[*id.availabilityZoneName]; !ok {
		log.Println(firewall.FirewallStatus.SyncStates)
		return fmt.Errorf("unexpected error: firewall ready, but has no VPC Endpoint in %s", *id.availabilityZoneName)
	}
	id.firewallVpcEndpointId = firewall.FirewallStatus.SyncStates[*id.availabilityZoneName].Attachment.EndpointId

	if _, err := id.ec2Api.CreateRoute(ctx, &ec2.CreateRouteInput{
		RouteTableId:         id.publicSubnetRouteTableId,
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		VpcEndpointId:        id.firewallVpcEndpointId,
	}); err != nil {
		return err
	}
	log.Println("route created for public subnet 0.0.0.0/0 --> firewall")

	if _, err := id.ec2Api.CreateRoute(ctx, &ec2.CreateRouteInput{
		RouteTableId:         id.internetGatewayRouteTableId,
		DestinationCidrBlock: aws.String(publicSubnetCidr),
		VpcEndpointId:        id.firewallVpcEndpointId,
	}); err != nil {
		return err
	}

	return nil
}

// SetupInternetGateway creates an internet gateway, and associates it with the VPC without waiting for association
// to complete
func (id *OnvIntegrationTestData) SetupInternetGateway(ctx context.Context) error {
	if id.vpcId == nil {
		return errors.New("vpc id must not be nil when creating internet gateway")
	}

	igw, err := id.ec2Api.CreateInternetGateway(ctx, &ec2.CreateInternetGatewayInput{})
	if err != nil {
		return err
	}
	id.internetGatewayId = igw.InternetGateway.InternetGatewayId
	log.Printf("created internet gateway: %s", *id.internetGatewayId)

	if _, err := id.ec2Api.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{*id.internetGatewayId},
		Tags:      defaultEc2Tags("osd-network-verifier-igw"),
	}); err != nil {
		return err
	}

	if _, err := id.ec2Api.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
		InternetGatewayId: id.internetGatewayId,
		VpcId:             id.vpcId,
	}); err != nil {
		return err
	}

	return nil
}

// SetupNatGateway creates a public NAT gateway and creates a route in the private subnet route table
func (id *OnvIntegrationTestData) SetupNatGateway(ctx context.Context) error {
	if id.vpcId == nil {
		return errors.New("vpc id must not be nil when creating NAT gateway")
	}

	if id.publicSubnetId == nil {
		return errors.New("public subnet id must not be nil when creating NAT gateway")
	}

	eip, err := id.ec2Api.AllocateAddress(ctx, &ec2.AllocateAddressInput{})
	if err != nil {
		return err
	}
	id.eipAllocationId = eip.AllocationId
	log.Printf("allocated EIP address: %s", *id.eipAllocationId)

	if _, err := id.ec2Api.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{*id.eipAllocationId},
		Tags:      defaultEc2Tags("osd-network-verifier-nat"),
	}); err != nil {
		return err
	}

	nat, err := id.ec2Api.CreateNatGateway(ctx, &ec2.CreateNatGatewayInput{
		SubnetId:         id.publicSubnetId,
		AllocationId:     id.eipAllocationId,
		ConnectivityType: "public",
	})
	if err != nil {
		return err
	}

	id.natGatewayId = nat.NatGateway.NatGatewayId
	log.Printf("created NAT gateway: %s", *id.natGatewayId)

	if _, err := id.ec2Api.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{*id.natGatewayId},
		Tags:      defaultEc2Tags("osd-network-verifier-nat"),
	}); err != nil {
		return err
	}

	return nil
}

// SetupFirewall creates a firewall, firewall policy, and firewall rule group to block quay.io
func (id *OnvIntegrationTestData) SetupFirewall(ctx context.Context) error {
	if id.vpcId == nil {
		return errors.New("vpc id must not be nil when creating firewall")
	}

	if id.firewallSubnetId == nil {
		return errors.New("firewall subnet id must not be nil when creating firewall")
	}

	rulegroup, err := id.networkFirewallApi.CreateRuleGroup(ctx, &networkfirewall.CreateRuleGroupInput{
		Capacity:      aws.Int32(3),
		RuleGroupName: aws.String("osd-network-verifier-rule-group" + slug),
		Type:          nfwTypes.RuleGroupTypeStateful,
		Description:   aws.String("Block quay.io"),
		RuleGroup: &nfwTypes.RuleGroup{
			RulesSource: &nfwTypes.RulesSource{
				RulesSourceList: &nfwTypes.RulesSourceList{
					GeneratedRulesType: nfwTypes.GeneratedRulesTypeDenylist,
					TargetTypes:        []nfwTypes.TargetType{nfwTypes.TargetTypeHttpHost, nfwTypes.TargetTypeTlsSni},
					Targets:            []string{".quay.io"},
				},
			},
		},
		Tags: defaultNetworkFirewallTags("osd-network-verifier-rule-group"),
	})
	if err != nil {
		return err
	}

	id.firewallRuleGroupArn = rulegroup.RuleGroupResponse.RuleGroupArn
	log.Printf("created firewall rule group: %s", *id.firewallRuleGroupArn)

	policy, err := id.networkFirewallApi.CreateFirewallPolicy(ctx, &networkfirewall.CreateFirewallPolicyInput{
		FirewallPolicy: &nfwTypes.FirewallPolicy{
			StatelessDefaultActions:         []string{"aws:forward_to_sfe"},
			StatelessFragmentDefaultActions: []string{"aws:forward_to_sfe"},
			StatefulRuleGroupReferences: []nfwTypes.StatefulRuleGroupReference{
				{
					ResourceArn: id.firewallRuleGroupArn,
				},
			},
		},
		FirewallPolicyName: aws.String("osd-network-verifier-firewall-policy" + slug),
		Description:        aws.String("Block quay.io"),
		Tags:               defaultNetworkFirewallTags("osd-network-verifier-firewall-policy"),
	})
	if err != nil {
		return err
	}

	id.firewallPolicyArn = policy.FirewallPolicyResponse.FirewallPolicyArn
	log.Printf("created firewall policy: %s", *id.firewallPolicyArn)

	firewall, err := id.networkFirewallApi.CreateFirewall(ctx, &networkfirewall.CreateFirewallInput{
		FirewallName:      aws.String("osd-network-verifier-firewall" + slug),
		FirewallPolicyArn: id.firewallPolicyArn,
		SubnetMappings: []nfwTypes.SubnetMapping{
			{
				SubnetId: id.firewallSubnetId,
			},
		},
		VpcId:                          id.vpcId,
		DeleteProtection:               false,
		Description:                    aws.String("osd-network-verifier-firewall"),
		EncryptionConfiguration:        nil,
		FirewallPolicyChangeProtection: false,
		SubnetChangeProtection:         false,
		Tags:                           defaultNetworkFirewallTags("osd-network-verifier-firewall"),
	})
	if err != nil {
		return err
	}

	id.firewallArn = firewall.Firewall.FirewallArn
	log.Printf("created firewall: %s", *id.firewallArn)

	return nil
}
