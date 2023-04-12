package aws

import (
	"context"
	"errors"
	"github.com/aws/smithy-go"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	nfwTypes "github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"
)

// CleanupVpc deletes a VPC
// Requires CleanupNatGateway, CleanupInternetGateway, CleanupRouteTables, and Cleanup Subnets to be run first
func (id *OnvIntegrationTestData) CleanupVpc(ctx context.Context) error {
	resp, err := id.ec2Api.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: defaultEc2TagFilters(),
	})
	if err != nil {
		return err
	}

	if len(resp.Vpcs) == 0 {
		log.Println("No VPCs found - skipping cleanup")
		return nil
	}

	for _, vpc := range resp.Vpcs {
		if _, err := id.ec2Api.DeleteVpc(ctx, &ec2.DeleteVpcInput{VpcId: vpc.VpcId}); err != nil {
			return err
		}
		log.Printf("deleted VPC: %s", *vpc.VpcId)
	}

	return nil
}

// CleanupSubnets deletes the firewall/public/private subnets
func (id *OnvIntegrationTestData) CleanupSubnets(ctx context.Context) error {
	resp, err := id.ec2Api.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: defaultEc2TagFilters(),
	})
	if err != nil {
		return err
	}

	if len(resp.Subnets) == 0 {
		log.Println("No Subnets found - skipping cleanup")
		return nil
	}

	for _, subnet := range resp.Subnets {
		if _, err := id.ec2Api.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{SubnetId: subnet.SubnetId}); err != nil {
			return err
		}
		log.Printf("deleted subnet: %s", *subnet.SubnetId)
	}

	return nil
}

// CleanupRouteTables disassociates and deletes the subnet route tables
func (id *OnvIntegrationTestData) CleanupRouteTables(ctx context.Context) error {
	resp, err := id.ec2Api.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: defaultEc2TagFilters(),
	})
	if err != nil {
		return err
	}

	for _, rt := range resp.RouteTables {
		log.Printf("cleaning up route table: %s", *rt.RouteTableId)
		for _, assoc := range rt.Associations {
			if _, err := id.ec2Api.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{
				AssociationId: assoc.RouteTableAssociationId,
			}); err != nil {
				return err
			}
			log.Printf("disassociated association: %s", *assoc.RouteTableAssociationId)
		}

		if _, err := id.ec2Api.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
			RouteTableId: rt.RouteTableId,
		}); err != nil {
			return err
		}
		log.Printf("deleted route table: %s", *rt.RouteTableId)
	}

	return nil
}

// CleanupInternetGateway detaches and deletes the IGW
func (id *OnvIntegrationTestData) CleanupInternetGateway(ctx context.Context) error {
	resp, err := id.ec2Api.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
		Filters: defaultEc2TagFilters(),
	})
	if err != nil {
		return err
	}

	if len(resp.InternetGateways) == 0 {
		log.Println("No internet gateways found - skipping cleanup")
		return nil
	}

	for _, igw := range resp.InternetGateways {
		for _, attach := range igw.Attachments {
			if _, err := id.ec2Api.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
				InternetGatewayId: igw.InternetGatewayId,
				VpcId:             attach.VpcId,
			}); err != nil {
				return err
			}
			log.Printf("detached internet gateway: %s from vpc: %s", *igw.InternetGatewayId, *attach.VpcId)
		}

		if _, err := id.ec2Api.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: igw.InternetGatewayId,
		}); err != nil {
			return err
		}
		log.Printf("internet gateway deleted: %s", *igw.InternetGatewayId)
	}

	return nil
}

// CleanupNatGateway deletes NAT Gateways
func (id *OnvIntegrationTestData) CleanupNatGateway(ctx context.Context) error {
	resp, err := id.ec2Api.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
		Filter: defaultEc2TagFilters(),
	})
	if err != nil {
		return err
	}

	if len(resp.NatGateways) == 0 {
		log.Println("No NAT gateways found - skipping cleanup")
		return nil
	}

	for _, nat := range resp.NatGateways {
		log.Printf("deleting NAT Gateway: %s", *nat.NatGatewayId)
		if _, err := id.ec2Api.DeleteNatGateway(ctx, &ec2.DeleteNatGatewayInput{NatGatewayId: nat.NatGatewayId}); err != nil {
			return err
		}

		log.Printf("waiting up to %s for the NAT Gateway to be deleted", 5*time.Minute)
		natGwWaiter := ec2.NewNatGatewayDeletedWaiter(id.ec2Api)

		if err := natGwWaiter.Wait(ctx, &ec2.DescribeNatGatewaysInput{
			NatGatewayIds: []string{*nat.NatGatewayId},
		}, 5*time.Minute); err != nil {
			return err
		}
		log.Printf("NAT gateway deleted: %s", *nat.NatGatewayId)
	}

	return nil
}

// CleanupElasticIp deletes EIPs that should have previously been associated with NAT Gateways
func (id *OnvIntegrationTestData) CleanupElasticIp(ctx context.Context) error {
	eipResp, err := id.ec2Api.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		Filters: defaultEc2TagFilters(),
	})
	if err != nil {
		return err
	}

	if len(eipResp.Addresses) == 0 {
		log.Println("No Elastic IPs found - skipping cleanup")

	}

	for _, eip := range eipResp.Addresses {
		if _, err := id.ec2Api.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{AllocationId: eip.AllocationId}); err != nil {
			return err
		}
		log.Printf("EIP released: %s", *eip.AllocationId)
	}

	return nil
}

// CleanupFirewall deletes all AWS NetworkFirewall Firewalls
func (id *OnvIntegrationTestData) CleanupFirewall(ctx context.Context) error {
	if _, err := id.networkFirewallApi.DescribeFirewall(ctx, &networkfirewall.DescribeFirewallInput{
		FirewallName: aws.String(firewallName),
	}); err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) {
			if ae.ErrorCode() == "ResourceNotFoundException" {
				log.Println("no firewall found - skipping cleanup")
				return nil
			}
		}
		return err
	}

	log.Printf("deleting firewall %s", firewallName)
	if _, err := id.networkFirewallApi.DeleteFirewall(ctx, &networkfirewall.DeleteFirewallInput{
		FirewallName: aws.String(firewallName),
	}); err != nil {
		return err
	}

	log.Printf("waiting up to %s for the firewall to be deleted", 20*time.Minute)
	firewallWaiter := NewFirewallDeletedWaiter(id.networkFirewallApi)
	if err := firewallWaiter.Wait(ctx, &networkfirewall.DescribeFirewallInput{
		FirewallName: aws.String(firewallName),
	}, 20*time.Minute); err != nil {
		return err
	}
	log.Printf("firewall deleted: %s", firewallName)

	return nil
}

func (id *OnvIntegrationTestData) CleanupFirewallPolicy(ctx context.Context) error {
	if _, err := id.networkFirewallApi.DescribeFirewallPolicy(ctx, &networkfirewall.DescribeFirewallPolicyInput{
		FirewallPolicyName: aws.String(firewallPolicyName),
	}); err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) {
			if ae.ErrorCode() == "ResourceNotFoundException" {
				log.Println("no firewall policy found - skipping cleanup")
				return nil
			}
		}
		return err
	}

	log.Printf("deleting firewall policy %s", firewallPolicyName)
	if _, err := id.networkFirewallApi.DeleteFirewallPolicy(ctx, &networkfirewall.DeleteFirewallPolicyInput{
		FirewallPolicyName: aws.String(firewallPolicyName),
	}); err != nil {
		return err
	}

	log.Printf("waiting up to %s for the firewall policy to be deleted", 10*time.Minute)
	firewallPolicyWaiter := NewFirewallPolicyDeletedWaiter(id.networkFirewallApi)
	if err := firewallPolicyWaiter.Wait(ctx, &networkfirewall.DescribeFirewallPolicyInput{
		FirewallPolicyName: aws.String(firewallPolicyName),
	}, 10*time.Minute); err != nil {
		return err
	}
	log.Printf("firewall policy deleted: %s", firewallPolicyName)

	return nil
}

func (id *OnvIntegrationTestData) CleanupRuleGroup(ctx context.Context) error {
	if _, err := id.networkFirewallApi.DescribeRuleGroup(ctx, &networkfirewall.DescribeRuleGroupInput{
		RuleGroupName: aws.String(firewallRuleGroupName),
		Type:          nfwTypes.RuleGroupTypeStateful,
	}); err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) {
			if ae.ErrorCode() == "ResourceNotFoundException" {
				log.Println("no rule group found - skipping cleanup")
				return nil
			}
		}
		return err
	}

	log.Printf("deleting firewall rule group %s", firewallRuleGroupName)
	if _, err := id.networkFirewallApi.DeleteRuleGroup(ctx, &networkfirewall.DeleteRuleGroupInput{
		RuleGroupName: aws.String(firewallRuleGroupName),
		Type:          nfwTypes.RuleGroupTypeStateful,
	}); err != nil {
		return err
	}
	log.Printf("rule group deleted: %s", firewallRuleGroupName)

	return nil
}
