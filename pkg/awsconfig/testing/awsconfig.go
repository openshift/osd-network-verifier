package main

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/client"
	aws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	credentials "github.com/aws/aws-sdk-go/aws/credentials"
	session "github.com/aws/aws-sdk-go/aws/session"
	ec2 "github.com/aws/aws-sdk-go/service/ec2"
	networkfirewall "github.com/aws/aws-sdk-go/service/networkfirewall"
)

type EC2 struct {
	*client.Client
}
func main() {

	wait := 120 * time.Second
	wait2 := 360 * time.Second
	creds := credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), os.Getenv("AWS_SESSION_TOKEN"))
	region := "us-east-1"


	svc := ec2.New(session.New(&aws.Config{
		Region:      &region,
		Credentials: creds,
	}))
	//create firewall client
	firewall_client := networkfirewall.New(session.Must(session.NewSession()), &aws.Config{
		Region:      &region,
		Credentials: creds,
	})
	
	fmt.Println("Creating VPC")
	//Create a VPC with CIRD block 10.0.0.0/16
	VPCinput := &ec2.CreateVpcInput{
		CidrBlock: aws.String("10.0.0.0/16"),
	}

	VPCresult, err := svc.CreateVpc(VPCinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully created a vpc")
	//Enable DNSHostname
	DNSHostnameinput := &ec2.ModifyVpcAttributeInput{
		EnableDnsHostnames: &ec2.AttributeBooleanValue{
			Value: aws.Bool(true),
		},
		VpcId: aws.String(*VPCresult.Vpc.VpcId),
	}

	_, err = svc.ModifyVpcAttribute(DNSHostnameinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}

	//Create InternetGateway
	IGinput := &ec2.CreateInternetGatewayInput{}
	IGresult, err := svc.CreateInternetGateway(IGinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully created IG")
	//Attach the InternetGateway to the VPC
	IGAttachinput := &ec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(*IGresult.InternetGateway.InternetGatewayId),
		VpcId:             aws.String(*VPCresult.Vpc.VpcId),
	}
	 _ , err = svc.AttachInternetGateway(IGAttachinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully attached IG to VPC")

	//Create a subnet named Public with range 10.0.0.0/24 in some AZ in the VPC
	Subnetinput := &ec2.CreateSubnetInput{
		CidrBlock: aws.String("10.0.0.0/24"),
		VpcId:     aws.String(*VPCresult.Vpc.VpcId),
	}

	PublicSubnet, err := svc.CreateSubnet(Subnetinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully created a public subnet")
	//create a subnet named Private with range 10.0.1.0/24 in the same AZ as step 3 in the VPC
	Subnet2input := &ec2.CreateSubnetInput{
		CidrBlock: aws.String("10.0.1.0/24"),
		VpcId:     aws.String(*VPCresult.Vpc.VpcId),
	}

	PrivateSubnet, err := svc.CreateSubnet(Subnet2input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully created a private subnet")

	//create a FirewallSubnet
	FWSubnetinput := &ec2.CreateSubnetInput{
		CidrBlock: aws.String("10.0.2.0/24"),
		VpcId:     aws.String(*VPCresult.Vpc.VpcId),
	}

	FirewallSubnet, err := svc.CreateSubnet(FWSubnetinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully created a firewall subnet")


	//Create a route table
	RouteTable1input := &ec2.CreateRouteTableInput{
		VpcId: aws.String(*VPCresult.Vpc.VpcId),
	}

	PublicRT, err := svc.CreateRouteTable(RouteTable1input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully created the first route table")

	//Associate the RouteTable to the public subnet
	AssociateRT1input := &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(*PublicRT.RouteTable.RouteTableId),
		SubnetId:     aws.String(*PublicSubnet.Subnet.SubnetId),
	}
	//RT1PublicAssociation will be declared when delete resources
	_ , err = svc.AssociateRouteTable(AssociateRT1input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully associate the first route table to the public subnet")
	//Get the EIP for the NAT Gateway: Allocate an Elastic IP address
	EIPinput := &ec2.AllocateAddressInput{
		Domain: aws.String("vpc"),
	}

	EIPresult, err := svc.AllocateAddress(EIPinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}

	//Create NAT Gateway in the public subnet(PublicSubnet) with an EIP
	NGinput := &ec2.CreateNatGatewayInput{
		AllocationId: aws.String(*EIPresult.AllocationId),
		SubnetId:     aws.String(*PublicSubnet.Subnet.SubnetId),
	}

	NGresult, err := svc.CreateNatGateway(NGinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	time.Sleep(wait)
	fmt.Println("Successfully create a NAT Gateway")
	//Create RouteTable2
	RouteTable2input := &ec2.CreateRouteTableInput{
		VpcId: aws.String(*VPCresult.Vpc.VpcId),
	}

	PrivateRT, err := svc.CreateRouteTable(RouteTable2input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully created the second route table")

	//Associate RouteTable2 to the Private Subnet
	AssociateRT2input := &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(*PrivateRT.RouteTable.RouteTableId),
		SubnetId:     aws.String(*PrivateSubnet.Subnet.SubnetId),
	}

	// AssociateRT2 will be declared when delete resources
	_ , err = svc.AssociateRouteTable(AssociateRT2input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully Associated the second Route table to the private subnet")
	//In the PrivateRT, create rule for 0.0.0.0/0 to the NAT Gateway
	Rule2input := &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(*NGresult.NatGateway.NatGatewayId),
		RouteTableId:         aws.String(*PrivateRT.RouteTable.RouteTableId),
	}

	_, err = svc.CreateRoute(Rule2input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully routed 0.0.0.0/0 to NAT Gateway in PrivateRT")
	
	//Create a third Route Table for Firewall Subnet
	RouteTable3input := &ec2.CreateRouteTableInput{
		VpcId: aws.String(*VPCresult.Vpc.VpcId),
	}

	FirewallRT, err := svc.CreateRouteTable(RouteTable3input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully created the third route table")

	//Associate RouteTable3 to the Firewall Subnet
	AssociateRT3input := &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(*FirewallRT.RouteTable.RouteTableId),
		SubnetId:     aws.String(*FirewallSubnet.Subnet.SubnetId),
	}

	// AssociateRT3 will be declared when delete resources
	_ , err = svc.AssociateRouteTable(AssociateRT3input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully Associated the third Route table to the firewall subnet")

	//In the  Firewall RouteTable, create rule for 0.0.0.0/0 to the Internet Gateway
	Rule3input := &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(*IGresult.InternetGateway.InternetGatewayId),
		RouteTableId:         aws.String(*FirewallRT.RouteTable.RouteTableId),
	}

	_, err = svc.CreateRoute(Rule3input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully routed 0.0.0.0/0 to Internet Gateway in FirewallRT")

	//Create Firewall Rule Group
	RuleGroupinput := &networkfirewall.CreateRuleGroupInput{
		Capacity:      aws.Int64(100),
		RuleGroupName: aws.String("test-firewall"),
		Type:          aws.String("STATEFUL"),
		RuleGroup: &networkfirewall.RuleGroup{
			RulesSource: &networkfirewall.RulesSource{
				RulesSourceList: &networkfirewall.RulesSourceList{
					GeneratedRulesType: aws.String("DENYLIST"),
					TargetTypes:        []*string{aws.String("TLS_SNI")},
					Targets:            []*string{aws.String(".quay.io"), aws.String(".amazonaws.com"), aws.String("api.openshift.com"), aws.String(".redhat.io")},
				},
			},
		},
	}

	statefulRuleGroup, err := firewall_client.CreateRuleGroup(RuleGroupinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully creates a Stateful Rule Group")
	time.Sleep(wait)
	//Create a firewall Policy
	FirewallPolicyInput := &networkfirewall.CreateFirewallPolicyInput{
		Description: aws.String("test"),
		FirewallPolicyName:  aws.String("testPolicy"),
		FirewallPolicy: &networkfirewall.FirewallPolicy{
			StatefulRuleGroupReferences: []*networkfirewall.StatefulRuleGroupReference{&networkfirewall.StatefulRuleGroupReference{
				ResourceArn: statefulRuleGroup.RuleGroupResponse.RuleGroupArn},
			},
			StatelessDefaultActions: []*string{aws.String("aws:forward_to_sfe")},
			StatelessFragmentDefaultActions: []*string{aws.String("aws:forward_to_sfe")},
		},
	}
	testFirewallPolicy, err := firewall_client.CreateFirewallPolicy(FirewallPolicyInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully created a Firewall Policy")

	//Create Firewall
	testFirewallInput := &networkfirewall.CreateFirewallInput{
		FirewallName:      aws.String("testFirewall"),
		FirewallPolicyArn: testFirewallPolicy.FirewallPolicyResponse.FirewallPolicyArn,
		SubnetMappings: []*networkfirewall.SubnetMapping{&networkfirewall.SubnetMapping{
			SubnetId: FirewallSubnet.Subnet.SubnetId},
		},
		VpcId: aws.String(*VPCresult.Vpc.VpcId),
	}

	Firewall, err := firewall_client.CreateFirewall(testFirewallInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully created a Firewall. Having to wait a long time for the endpoint to be ready!")
	time.Sleep(wait2)

	DescribeFirewallInput := &networkfirewall.DescribeFirewallInput{
		FirewallName	:aws.String(*Firewall.Firewall.FirewallName),
	}
	DescribeFirewall, err := firewall_client.DescribeFirewall(DescribeFirewallInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully Get the Information of the Firewall")
	//Need to get the subnet availability zone to access SyncStatus of the describefirewall output
	Rule1input := &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		VpcEndpointId:            aws.String(*DescribeFirewall.FirewallStatus.SyncStates[*FirewallSubnet.Subnet.AvailabilityZone].Attachment.EndpointId),
		RouteTableId:         aws.String(*PublicRT.RouteTable.RouteTableId),
	}

	_, err = svc.CreateRoute(Rule1input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully route 0.0.0.0/0 to Firewall Endpoint in the PublicRT")

	//Create a fourth Route Table for IG
	RouteTable4input := &ec2.CreateRouteTableInput{
		VpcId: aws.String(*VPCresult.Vpc.VpcId),
	}

	IgRT, err := svc.CreateRouteTable(RouteTable4input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully created the fourth route table")

	//Associate IgRT(Internet Gateway Route Table) to the IG
	AssociateRT4input := &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(*IgRT.RouteTable.RouteTableId),
		GatewayId:     aws.String(*IGresult.InternetGateway.InternetGatewayId),
	}

	// AssociateRT4 will be declared when delete resources
	_ , err = svc.AssociateRouteTable(AssociateRT4input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully Associated the fourth Route table to the IG")

	//In the RouteTable, create rule for 10.0.0.0/24 to the Firewall Endpoint
	Rule4input := &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String("10.0.0.0/24"),
		VpcEndpointId:            aws.String(*DescribeFirewall.FirewallStatus.SyncStates[*FirewallSubnet.Subnet.AvailabilityZone].Attachment.EndpointId),
		RouteTableId:         aws.String(*IgRT.RouteTable.RouteTableId),
	}

	_, err = svc.CreateRoute(Rule4input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	fmt.Println("Successfully routed 10.0.0.0/24 to Firewall Endpoint")

/*

	//Resource Deletion***********************

	//Detach InternetGateway
	DetachIGinput := &ec2.DetachInternetGatewayInput{
		InternetGatewayId: aws.String(*IGresult.InternetGateway.InternetGatewayId),
		VpcId:             aws.String(*VPCresult.Vpc.VpcId),
	}

	_ , err = svc.DetachInternetGateway(DetachIGinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	
	//Delete InternetGateway
	DeleteIGinput := &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: aws.String(*IGresult.InternetGateway.InternetGatewayId),
	}

	_ , err = svc.DeleteInternetGateway(DeleteIGinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}

	//Delete NAT Gateway
	DeleteNATGatewayinput := &ec2.DeleteNatGatewayInput{
		NatGatewayId: aws.String(*NGresult.NatGateway.NatGatewayId),
	}

	_ , err = svc.DeleteNatGateway(DeleteNATGatewayinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}

	//Delete Subnets
	DeleteSubnet1input := &ec2.DeleteSubnetInput{
		SubnetId: aws.String(*PublicSubnet.Subnet.SubnetId),
	}
	DeleteSubnet2input := &ec2.DeleteSubnetInput{
		SubnetId: aws.String(*PrivateSubnet.Subnet.SubnetId),
	}


	_ , err = svc.DeleteSubnet(DeleteSubnet1input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}

	_ , err = svc.DeleteSubnet(DeleteSubnet2input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	
	//Dissasociate RouteTable1 to Public Subnet
	DisassociationRT1Subnetinput := &ec2.DisassociateRouteTableInput{
		AssociationId: aws.String(*RT1PublicAssociation.AssociationId),
	}

	_ , err = svc.DisassociateRouteTable(DisassociationRT1Subnetinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	//Delete route 0.0.0.0/0 to IG in the route table
	DeleteRT1Routeinput := &ec2.DeleteRouteInput{
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		RouteTableId:         aws.String(*PublicRT.RouteTable.RouteTableId),
	}

	_ , err = svc.DeleteRoute(DeleteRT1Routeinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	//Delete routetable1
	DeleteRT1input := &ec2.DeleteRouteTableInput{
		RouteTableId: aws.String(*PublicRT.RouteTable.RouteTableId),
	}

	_, err = svc.DeleteRouteTable(DeleteRT1input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}

	//Dissasociate RouteTable2 to Private Subnet
	DisassociationRT2Subnetinput := &ec2.DisassociateRouteTableInput{
		AssociationId: aws.String(*AssociateRT2.AssociationId),
	}

	_ , err = svc.DisassociateRouteTable(DisassociationRT2Subnetinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	//Delete route 0.0.0.0/0 to NAT Gateway in the route table
	DeleteRT2Routeinput := &ec2.DeleteRouteInput{
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		RouteTableId:         aws.String(*PrivateRT.RouteTable.RouteTableId),
	}

	_ , err = svc.DeleteRoute(DeleteRT2Routeinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
	//Delete routetable2
	DeleteRT2input := &ec2.DeleteRouteTableInput{
		RouteTableId: aws.String(*PrivateRT.RouteTable.RouteTableId),
	}

	_ , err = svc.DeleteRouteTable(DeleteRT2input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}

	//DeleteVPC
	DeleteVPCinput := &ec2.DeleteVpcInput{
		VpcId: aws.String(*VPCresult.Vpc.VpcId),
	}

	_ , err = svc.DeleteVpc(DeleteVPCinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}
*/

}

