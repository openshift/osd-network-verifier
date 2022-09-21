package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
)

// CleanupVpc deletes a VPC
// Requires CleanupNatGateway, CleanupInternetGateway, CleanupRouteTables, and Cleanup Subnets to be run first
func (id *OnvIntegrationTestData) CleanupVpc(ctx context.Context) error {
	if id.vpcId == nil {
		log.Println("skipping VPC cleanup due to missing vpc id")
		return nil
	}

	if _, err := id.ec2Api.DeleteVpc(ctx, &ec2.DeleteVpcInput{VpcId: id.vpcId}); err != nil {
		return err
	}

	log.Printf("deleted VPC: %s", *id.vpcId)
	id.vpcId = nil

	return nil
}

// CleanupSubnets deletes the public/private subnets
func (id *OnvIntegrationTestData) CleanupSubnets(ctx context.Context) error {
	if id.publicSubnetId == nil || id.privateSubnetId == nil || id.firewallSubnetId == nil {
		log.Println("skipping subnet cleanup due to missing subnet ids")
		return nil
	}

	if _, err := id.ec2Api.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{SubnetId: id.privateSubnetId}); err != nil {
		return err
	}

	log.Printf("deleted private subnet: %s", *id.privateSubnetId)
	id.privateSubnetId = nil

	if _, err := id.ec2Api.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{SubnetId: id.firewallSubnetId}); err != nil {
		return err
	}

	log.Printf("deleted firewall subnet: %s", *id.firewallSubnetId)
	id.firewallRuleGroupArn = nil

	if _, err := id.ec2Api.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{SubnetId: id.publicSubnetId}); err != nil {
		return err
	}

	log.Printf("deleted public subnet: %s", *id.publicSubnetId)
	id.publicSubnetId = nil

	return nil
}

// CleanupRouteTables disassociates and deletes the subnet route tables
func (id *OnvIntegrationTestData) CleanupRouteTables(ctx context.Context) error {
	if id.publicSubnetRouteTableAssociationId == nil || id.privateSubnetRouteTableAssociationId == nil || id.firewallSubnetRouteTableAssociationId == nil {
		log.Println("skipping route table cleanup due to missing association ids")
		return nil
	}

	if id.privateSubnetRouteTableId == nil || id.publicSubnetRouteTableId == nil || id.firewallSubnetRouteTableId == nil {
		log.Println("skipping route table cleanup due to missing route table ids")
		return nil
	}

	if _, err := id.ec2Api.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{
		AssociationId: id.privateSubnetRouteTableAssociationId,
	}); err != nil {
		return err
	}

	log.Printf("disassociated private subnet route table: %s", *id.privateSubnetRouteTableAssociationId)
	id.privateSubnetRouteTableAssociationId = nil

	if _, err := id.ec2Api.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{
		AssociationId: id.firewallSubnetRouteTableAssociationId,
	}); err != nil {
		return err
	}

	log.Printf("disassociated firewall subnet route table: %s", *id.firewallSubnetRouteTableAssociationId)
	id.firewallSubnetRouteTableAssociationId = nil

	if _, err := id.ec2Api.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{
		AssociationId: id.publicSubnetRouteTableAssociationId,
	}); err != nil {
		return err
	}

	log.Printf("disassociated public subnet route table: %s", *id.publicSubnetRouteTableAssociationId)
	id.publicSubnetRouteTableAssociationId = nil

	if _, err := id.ec2Api.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{
		AssociationId: id.internetGatewayRouteTableAssociationId,
	}); err != nil {
		return err
	}

	log.Printf("disassociated internet gateway route table: %s", *id.internetGatewayRouteTableAssociationId)
	id.privateSubnetRouteTableAssociationId = nil

	if _, err := id.ec2Api.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
		RouteTableId: id.privateSubnetRouteTableId,
	}); err != nil {
		return err
	}
	log.Printf("deleted private subnet route table: %s", *id.privateSubnetRouteTableId)
	id.privateSubnetRouteTableId = nil

	if _, err := id.ec2Api.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
		RouteTableId: id.firewallSubnetRouteTableId,
	}); err != nil {
		return err
	}
	log.Printf("deleted firewall subnet route table: %s", *id.firewallSubnetRouteTableId)
	id.firewallSubnetRouteTableId = nil

	if _, err := id.ec2Api.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
		RouteTableId: id.publicSubnetRouteTableId,
	}); err != nil {
		return err
	}
	log.Printf("deleted public subnet route table: %s", *id.publicSubnetRouteTableId)
	id.publicSubnetRouteTableId = nil

	if _, err := id.ec2Api.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
		RouteTableId: id.internetGatewayRouteTableId,
	}); err != nil {
		return err
	}
	log.Printf("deleted internet gateway route table: %s", *id.internetGatewayRouteTableId)
	id.publicSubnetRouteTableId = nil

	return nil
}

// CleanupInternetGateway detaches and deletes the IGW
func (id *OnvIntegrationTestData) CleanupInternetGateway(ctx context.Context) error {
	if id.vpcId == nil {
		log.Println("skipping internet gateway cleanup due to missing VPC id")
		return nil
	}

	if id.internetGatewayId == nil {
		log.Println("skipping internet gateway cleanup due to missing IGW id")
		return nil
	}

	if _, err := id.ec2Api.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
		InternetGatewayId: id.internetGatewayId,
		VpcId:             id.vpcId,
	}); err != nil {
		return err
	}
	log.Printf("detached internet gateway: %s", *id.internetGatewayId)

	if _, err := id.ec2Api.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: id.internetGatewayId,
	}); err != nil {
		return err
	}
	log.Printf("internet gateway deleted: %s", *id.internetGatewayId)
	id.internetGatewayId = nil

	return nil
}

// CleanupNatGateway deletes the NAT Gateway and associated EIP
func (id *OnvIntegrationTestData) CleanupNatGateway(ctx context.Context) error {
	if id.natGatewayId == nil {
		log.Println("skipping NAT gateway cleanup due to missing nat gateway id")
		return nil
	}

	log.Printf("deleting NAT Gateway: %s", *id.natGatewayId)
	if _, err := id.ec2Api.DeleteNatGateway(ctx, &ec2.DeleteNatGatewayInput{NatGatewayId: id.natGatewayId}); err != nil {
		return err
	}

	log.Printf("waiting up to %s for the NAT Gateway to be deleted", 5*time.Minute)
	natGwWaiter := ec2.NewNatGatewayDeletedWaiter(id.ec2Api)
	if err := natGwWaiter.Wait(ctx, &ec2.DescribeNatGatewaysInput{
		NatGatewayIds: []string{*id.natGatewayId},
	}, 5*time.Minute); err != nil {
		return err
	}
	log.Printf("NAT gateway deleted: %s", *id.natGatewayId)
	id.natGatewayId = nil

	if id.eipAllocationId == nil {
		log.Println("skipping eip cleanup due to missing eip allocation id")
		return nil
	}

	if _, err := id.ec2Api.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{AllocationId: id.eipAllocationId}); err != nil {
		return err
	}

	log.Printf("EIP released: %s", *id.eipAllocationId)
	id.eipAllocationId = nil

	return nil
}

func (id *OnvIntegrationTestData) CleanupFirewall(ctx context.Context) error {
	if id.firewallVpcEndpointId != nil && id.internetGatewayRouteTableId != nil && id.publicSubnetRouteTableId != nil {
		if _, err := id.ec2Api.DeleteRoute(ctx, &ec2.DeleteRouteInput{
			RouteTableId:         id.internetGatewayRouteTableId,
			DestinationCidrBlock: aws.String(publicSubnetCidr),
		}); err != nil {
			return err
		}

		if _, err := id.ec2Api.DeleteRoute(ctx, &ec2.DeleteRouteInput{
			RouteTableId:         id.publicSubnetRouteTableId,
			DestinationCidrBlock: aws.String("0.0.0.0/0"),
		}); err != nil {
			return err
		}
	}
	if id.firewallArn == nil {
		log.Println("skipping firewall cleanup due to missing firewall arn")
		return nil
	}

	log.Printf("deleting firewall %s", *id.firewallArn)
	if _, err := id.networkFirewallApi.DeleteFirewall(ctx, &networkfirewall.DeleteFirewallInput{FirewallArn: id.firewallArn}); err != nil {
		return err
	}

	log.Printf("waiting up to %s for the firewall to be deleted", 10*time.Minute)
	firewallWaiter := NewFirewallDeletedWaiter(id.networkFirewallApi)
	if err := firewallWaiter.Wait(ctx, &networkfirewall.DescribeFirewallInput{
		FirewallArn: id.firewallArn,
	}, 10*time.Minute); err != nil {
		return err
	}
	log.Printf("firewall deleted: %s", *id.firewallArn)
	id.firewallArn = nil

	log.Printf("deleting firewall policy %s", *id.firewallPolicyArn)
	if _, err := id.networkFirewallApi.DeleteFirewallPolicy(ctx, &networkfirewall.DeleteFirewallPolicyInput{
		FirewallPolicyArn: id.firewallPolicyArn,
	}); err != nil {
		return err
	}

	log.Printf("waiting up to %s for the firewall policy to be deleted", 10*time.Minute)
	firewallPolicyWaiter := NewFirewallPolicyDeletedWaiter(id.networkFirewallApi)
	if err := firewallPolicyWaiter.Wait(ctx, &networkfirewall.DescribeFirewallPolicyInput{
		FirewallPolicyArn: id.firewallPolicyArn,
	}, 10*time.Minute); err != nil {
		return err
	}
	log.Printf("firewall policy deleted: %s", *id.firewallPolicyArn)
	id.firewallPolicyArn = nil

	log.Printf("deleting firewall rule group %s", *id.firewallRuleGroupArn)
	if _, err := id.networkFirewallApi.DeleteRuleGroup(ctx, &networkfirewall.DeleteRuleGroupInput{
		RuleGroupArn: id.firewallRuleGroupArn,
	}); err != nil {
		return err
	}
	log.Printf("rule group deleted: %s", *id.firewallRuleGroupArn)

	return nil
}
