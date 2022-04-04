package main

import (
	"fmt"
	"os"

	credentials "github.com/aws/aws-sdk-go/aws/credentials"
	aws "github.com/aws/aws-sdk-go/aws"
	ec2 "github.com/aws/aws-sdk-go/service/ec2"
	session "github.com/aws/aws-sdk-go/aws/session"
	networkfirewall "github.com/aws/aws-sdk-go/service/networkfirewall"
	"github.com/aws/aws-sdk-go/aws/awserr"
)
func awsTest() {

	creds := credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), os.Getenv("AWS_SESSION_TOKEN"))
	region := os.Getenv("AWS_DEFAULT_REGION")

//	session, err := session.NewSession(&aws.Config{
//		Region:      &region,
//		Credentials: creds,
//	})
	
	svc := ec2.New(session.New(&aws.Config{
		Region:      &region,
		Credentials: creds,
	}))
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
	}
	//Enable DNSHostname
	DNSHostnameinput := &ec2.ModifyVpcAttributeInput{
		EnableDnsHostnames: &ec2.AttributeBooleanValue{
			Value: aws.Bool(true),
		},
		VpcId: aws.String(*VPCresult.Vpc.VpcId),
	}

	_ , err = svc.ModifyVpcAttribute(DNSHostnameinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
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
	}
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
	}
	
	//Create a subnet named Public with range 10.0.0.0/17 in some AZ in the VPC
	Subnetinput := &ec2.CreateSubnetInput{
		CidrBlock: aws.String("10.0.0.0/17"),
		VpcId:     aws.String(*VPCresult.Vpc.VpcId),
	}
	
	Subnetresult, err := svc.CreateSubnet(Subnetinput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}
	//create a subnet named Private with range 10.0.128.0/17 in the same AZ as step 3 in the VPC
	Subnet2input := &ec2.CreateSubnetInput{
		CidrBlock: aws.String("10.0.128.0/17"),
		VpcId:     aws.String(*VPCresult.Vpc.VpcId),
	}
	
	Subnet2result, err := svc.CreateSubnet(Subnet2input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}

	//Create a route table
	RouteTable1input := &ec2.CreateRouteTableInput{
		VpcId: aws.String(*VPCresult.Vpc.VpcId),
	}

	RouteTable1result, err := svc.CreateRouteTable(RouteTable1input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}

	//Associate the RouteTable to the public subnet
	AssociateRT1input := &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(*RouteTable1result.RouteTable.RouteTableId),
		SubnetId:     aws.String(*Subnetresult.Subnet.SubnetId),
	}

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
	}
	//In the route table, create a rule for 0.0.0.0/0 to the InternetGateway
	Rule1input := &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(*IGresult.InternetGateway.InternetGatewayId),
		RouteTableId:         aws.String(*RouteTable1result.RouteTable.RouteTableId),
	}

	_ , err = svc.CreateRoute(Rule1input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}

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
	}

	//Create NAT Gateway in the public subnet(Subnetresult) with an EIP
	NGinput := &ec2.CreateNatGatewayInput{
		AllocationId: aws.String(*EIPresult.AllocationId),
		SubnetId:     aws.String(*Subnetresult.Subnet.SubnetId),
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
	}	
	
	//Create RouteTable2
	RouteTable2input := &ec2.CreateRouteTableInput{
		VpcId: aws.String(*VPCresult.Vpc.VpcId),
	}

	RouteTable2result, err := svc.CreateRouteTable(RouteTable2input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}

	//Associate RouteTable2 to the Private Subnet
	AssociateRT2input := &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(*RouteTable2result.RouteTable.RouteTableId),
		SubnetId:     aws.String(*Subnet2result.Subnet.SubnetId),
	}

	 _, err = svc.AssociateRouteTable(AssociateRT2input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}

	//In the RouteTable, create rule for 0.0.0.0/0 to the NAT Gateway
	Rule2input := &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(*NGresult.NatGateway.NatGatewayId),
		RouteTableId:         aws.String(*RouteTable2result.RouteTable.RouteTableId),
	}

	_ , err = svc.CreateRoute(Rule2input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	}

	//Create a firewall Policy
	firewall_client := networkfirewall.New(session.Must(session.NewSession()),&aws.Config{
		Region:      &region,
		Credentials: creds,
	})

	FirewallPolicyInput := &networkfirewall.CreateFirewallPolicyInput{

		
	}



}
