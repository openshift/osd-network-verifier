package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/networkfirewall"
	"github.com/aws/aws-sdk-go/service/networkfirewall/networkfirewalliface"

	"k8s.io/apimachinery/pkg/util/wait"
)

type Client struct {
	ec2Client      ec2iface.EC2API
	firewallClient networkfirewalliface.NetworkFirewallAPI
}

func main() {
	var profile string
	var region string
	var awsClient Client

	awsProfile := flag.String("p", "", "aws profile")
	awsRegion := flag.String("r", "", "aws region")
	flag.Parse()
	if *awsProfile != "" {
		fmt.Println("Profile: ", *awsProfile)
		profile = *awsProfile
	} else {
		profile = os.Getenv("AWS_PROFILE")
	}
	if *awsRegion != "" {
		fmt.Println("Region: ", *awsRegion)
		region = *awsRegion
	} else {
		region = os.Getenv("AWS_REGION")
	}

	if profile == "" {
		fmt.Println("Profile is not provided, will take in ENV")
		awsClient = NewClient(region, "")
	} else {
		awsClient = NewClient(region, profile)
	}
	//Create VPC
	Vpc, err := awsClient.CreateVPC()
	if err != nil {
		fmt.Println("Failed to create VPC ")
		return
	}
	//Create Internet Gateway
	IG, err := awsClient.CreateInternetGatewayForVpc(*Vpc.Vpc.VpcId)
	if err != nil {
		fmt.Println("Failed to create IGW")
		return
	}
	//Create Public Subnet
	PublicSubnet, err := awsClient.CreateSubnet("10.0.0.0/24", *Vpc.Vpc.VpcId)
	if err != nil {
		fmt.Println("Failed to create PublicSubnet")
		return
	}
	//Create Private Subnet
	PrivateSubnet, err := awsClient.CreateSubnet("10.0.1.0/24", *Vpc.Vpc.VpcId)
	if err != nil {
		fmt.Println("Failed to create PrivateSubnet")
		return
	}
	//Create Firewall Subnet
	FirewallSubnet, err := awsClient.CreateSubnet("10.0.2.0/24", *Vpc.Vpc.VpcId)
	if err != nil {
		fmt.Println("Failed to create Firewall Subnet")
		return
	}
	//Create PublicSubnet Route Table
	PublicRT, err := awsClient.CreateRouteTableForSubnet(*Vpc.Vpc.VpcId, *PublicSubnet.Subnet.SubnetId)
	if err != nil {
		fmt.Println("Failed to create Public Subnet Route Table")
		return
	}
	//Create PrivateSubnet Route Table
	PrivateRT, err := awsClient.CreateRouteTableForSubnet(*Vpc.Vpc.VpcId, *PrivateSubnet.Subnet.SubnetId)
	if err != nil {
		fmt.Println("Failed to create Private Subnet Route Table")
		return
	}
	//Create FirewallSubnet Route Table
	FirewallRT, err := awsClient.CreateRouteTableForSubnet(*Vpc.Vpc.VpcId, *FirewallSubnet.Subnet.SubnetId)
	if err != nil {
		fmt.Println("Failed to create Firewall Subnet Route Table")
		return
	}
	//Create IGW Route Table
	IgRT, err := awsClient.CreateRouteTableForIGW(*Vpc.Vpc.VpcId, *IG.InternetGateway.InternetGatewayId)
	if err != nil {
		fmt.Println("Failed to create IGW Route Table")
		return
	}
	//Create NAT Gateway
	NatGateway, err := awsClient.CreateNatGateway(*PublicSubnet.Subnet.SubnetId)
	if err != nil {
		fmt.Println("Failed to create NAT Gateway")
		return
	}

	//Create route 0.0.0.0/0 in PrivateRT for NatGateway
	err = awsClient.CreateRouteForGateway("0.0.0.0/0", *NatGateway.NatGateway.NatGatewayId, *PrivateRT.RouteTable.RouteTableId)
	if err != nil {
		fmt.Println("Failed to create route 0.0.0.0/0 in Private Subnet Route Tabel to NAT Gateway")
		return
	}
	fmt.Println("Successfully Created a route 0.0.0.0/0 to NatGateway in Private Subnet")
	//Create route 0.0.0.0/0 in FirewallSubnet for IG
	err = awsClient.CreateRouteForGateway("0.0.0.0/0", *IG.InternetGateway.InternetGatewayId, *FirewallRT.RouteTable.RouteTableId)
	if err != nil {
		fmt.Println("Failed to create route 0.0.0.0/0 in Firewall Subnet to IGW")
		return
	}
	fmt.Println("Successfully Created a route 0.0.0.0/0 to IGW in Firewall Subnet")

	//Create Firewall
	Firewall, err := awsClient.CreateFirewall(FirewallSubnet, Vpc)
	if err != nil {
		fmt.Println("Failed to create Firewall")
		return
	}
	//Wait for the Firewall to be ready
	if awsClient.IsFirewallReady(*Firewall.Firewall.FirewallName) != nil {
		fmt.Println(awsClient.IsFirewallReady(*Firewall.Firewall.FirewallName).Error())
	}
	fmt.Println("VpcEndpoint is Now Available!")

	DescribeFirewall, err := awsClient.firewallClient.DescribeFirewall(&networkfirewall.DescribeFirewallInput{
		FirewallName: aws.String(*Firewall.Firewall.FirewallName),
	})
	if err != nil {
		fmt.Println("Failed to Describe Firewall")
		return
	}
	//Create route 0.0.0.0/0 in PublicRT for FirewallEndpoint
	firewallEndpointId := *DescribeFirewall.FirewallStatus.SyncStates[*FirewallSubnet.Subnet.AvailabilityZone].Attachment.EndpointId
	//Check to see if the Firewall VpcEndpoint is available
	err = awsClient.CreateRouteToFirewall("0.0.0.0/0", firewallEndpointId, PublicRT)
	if err != nil {
		fmt.Println("Failed to create route 0.0.0.0/0 in Public Subnet Route Table to Firewall Endpoint")
		return
	}
	fmt.Println("Successfully route 0.0.0.0/0 to the Firewall Endpoint in PublicRT ")
	//Create route 10.0.0.0/24 in IgRt to FirewallEndpoint
	err = awsClient.CreateRouteToFirewall("10.0.0.0/24", firewallEndpointId, IgRT)
	if err != nil {
		fmt.Println("Failed to create route 10.0.0.0/24 in IGW Route Table to Firewall Endpoint")
		return
	}
	fmt.Println("Successfully route 10.0.0.0/24 to the Firewall Endpoint in IgRT ")
	fmt.Println("Successfully Created VPC and Firewall")

}

func NewClient(region, profile string) Client {
	var awsClient Client
	if profile == ""{
		creds := credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), os.Getenv("AWS_SESSION_TOKEN"))

		ec2Client := ec2.New(session.New(&aws.Config{
			Region:      &region,
			Credentials: creds,
		}))

		firewallClient := networkfirewall.New(session.Must(session.NewSession()), &aws.Config{
			Region:      &region,
			Credentials: creds,
		})

		awsClient = Client{
			ec2Client,
			firewallClient,
		}
	}else{
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			Config: aws.Config{
				Region: &region,
			},
			Profile: profile,
		}))
		if _, err := sess.Config.Credentials.Get(); err != nil {
			if err != nil {
				fmt.Println("could not create AWS session: ", err)
			}
		}

		ec2Client := ec2.New(sess)
		firewallClient := networkfirewall.New(sess)

		awsClient = Client{
			ec2Client,
			firewallClient,
		}
	}
	return awsClient
}

func (c Client) CreateVPC() (ec2.CreateVpcOutput, error) {
	VPC, err := c.ec2Client.CreateVpc(&ec2.CreateVpcInput{
		CidrBlock: aws.String("10.0.0.0/16"),
	})
	if err != nil {
		fmt.Println(err.Error())
		return ec2.CreateVpcOutput{}, err
	}

	fmt.Println("Successfully created a vpc with ID:", string(*VPC.Vpc.VpcId))
	//Enable DNSHostname
	_, err = c.ec2Client.ModifyVpcAttribute(&ec2.ModifyVpcAttributeInput{
		EnableDnsHostnames: &ec2.AttributeBooleanValue{
			Value: aws.Bool(true),
		},
		VpcId: aws.String(*VPC.Vpc.VpcId),
	})
	if err != nil {
		fmt.Println(err.Error())
		return ec2.CreateVpcOutput{}, err
	}
	fmt.Println("Successfully enabled DNSHostname for the newly created VPC")
	return *VPC, nil
}

func (c Client) CreateInternetGatewayForVpc(vpcID string) (ec2.CreateInternetGatewayOutput, error) {
	IGresult, err := c.ec2Client.CreateInternetGateway(&ec2.CreateInternetGatewayInput{})
	if err != nil {
		fmt.Println(err.Error())
		return ec2.CreateInternetGatewayOutput{}, err
	}

	fmt.Println("Successfully created IG")
	//Attach the InternetGateway to the VPC
	_, err = c.ec2Client.AttachInternetGateway(&ec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(*IGresult.InternetGateway.InternetGatewayId),
		VpcId:             aws.String(vpcID),
	})
	if err != nil {
		fmt.Println(err.Error())
		return ec2.CreateInternetGatewayOutput{}, err
	}
	fmt.Println("Successfully attached IG to VPC")

	return *IGresult, nil
}

func (c Client) CreateSubnet(CidrBlock string, vpcID string) (ec2.CreateSubnetOutput, error) {

	Subnet, err := c.ec2Client.CreateSubnet(&ec2.CreateSubnetInput{
		CidrBlock: aws.String(CidrBlock),
		VpcId:     aws.String(vpcID),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return ec2.CreateSubnetOutput{}, err
	}

	return *Subnet, nil
}

func (c Client) CreateRouteTableForSubnet(vpcID string, subnetID string) (ec2.CreateRouteTableOutput, error) {
	RT, err := c.ec2Client.CreateRouteTable(&ec2.CreateRouteTableInput{
		VpcId: aws.String(vpcID),
	})
	if err != nil {
		fmt.Println(err.Error())
		return ec2.CreateRouteTableOutput{}, err
	}
	Associateinput := &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(*RT.RouteTable.RouteTableId),
		SubnetId:     aws.String(subnetID),
	}
	_, err = c.ec2Client.AssociateRouteTable(Associateinput)
	if err != nil {
		fmt.Println(err.Error())
		return ec2.CreateRouteTableOutput{}, err
	}
	return *RT, nil
}

func (c Client) CreateRouteTableForIGW(vpcId string, IgwId string) (ec2.CreateRouteTableOutput, error) {
	RouteTable1input := &ec2.CreateRouteTableInput{
		VpcId: aws.String(vpcId),
	}
	RT, err := c.ec2Client.CreateRouteTable(RouteTable1input)
	if err != nil {
		fmt.Println(err.Error())
		return ec2.CreateRouteTableOutput{}, err
	}
	Associateinput := &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(*RT.RouteTable.RouteTableId),
		GatewayId:    aws.String(IgwId),
	}
	_, err = c.ec2Client.AssociateRouteTable(Associateinput)
	if err != nil {
		fmt.Println(err.Error())
		return ec2.CreateRouteTableOutput{}, err
	}
	return *RT, nil
}

func (c Client) CreateNatGateway(SubnetId string) (ec2.CreateNatGatewayOutput, error) {
	EIPinput := &ec2.AllocateAddressInput{
		Domain: aws.String("vpc"),
	}

	EIPresult, err := c.ec2Client.AllocateAddress(EIPinput)
	if err != nil {
		fmt.Println(err.Error())
		return ec2.CreateNatGatewayOutput{}, err
	}
	NGinput := &ec2.CreateNatGatewayInput{
		AllocationId: aws.String(*EIPresult.AllocationId),
		SubnetId:     aws.String(SubnetId),
	}

	NGresult, err := c.ec2Client.CreateNatGateway(NGinput)
	if err != nil {
		fmt.Println(err.Error())
		return ec2.CreateNatGatewayOutput{}, err
	}
	fmt.Println("Waiting 2 minutes for NAT Gateway to become ready")
	//Wait for NAT Gatway to be ready
	err = c.ec2Client.WaitUntilNatGatewayAvailable(&ec2.DescribeNatGatewaysInput{
		NatGatewayIds: []*string{aws.String(*NGresult.NatGateway.NatGatewayId)},
	})
	if err != nil {
		fmt.Println(err.Error())
		return ec2.CreateNatGatewayOutput{}, err
	}
	fmt.Println("Successfully create a NAT Gateway")
	return *NGresult, nil
}
func (c Client) CreateRouteForGateway(CidrBlock string, GatewayID string, RouteTableId string) error {
	Ruleinput := &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String(CidrBlock),
		GatewayId:            aws.String(GatewayID),
		RouteTableId:         aws.String(RouteTableId),
	}

	_, err := c.ec2Client.CreateRoute(Ruleinput)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	return nil
}
func (c Client) CreateFirewall(FirewallSubnet ec2.CreateSubnetOutput, Vpc ec2.CreateVpcOutput) (networkfirewall.CreateFirewallOutput, error) {
	RuleGroupinput := &networkfirewall.CreateRuleGroupInput{
		Capacity:      aws.Int64(100),
		RuleGroupName: aws.String("test-firewall"),
		Type:          aws.String("STATEFUL"),
		RuleGroup: &networkfirewall.RuleGroup{
			RulesSource: &networkfirewall.RulesSource{
				RulesSourceList: &networkfirewall.RulesSourceList{
					GeneratedRulesType: aws.String("DENYLIST"),
					TargetTypes:        []*string{aws.String("TLS_SNI")},
					Targets:            []*string{aws.String(".quay.io"), aws.String("api.openshift.com"), aws.String(".redhat.io")},
				},
			},
		},
	}

	statefulRuleGroup, err := c.firewallClient.CreateRuleGroup(RuleGroupinput)
	if err != nil {
		fmt.Println(err.Error())
		return networkfirewall.CreateFirewallOutput{}, err
	}
	fmt.Println("Successfully creates a Stateful Rule Group")

	FirewallPolicyInput := &networkfirewall.CreateFirewallPolicyInput{
		Description:        aws.String("test"),
		FirewallPolicyName: aws.String("testPolicy"),
		FirewallPolicy: &networkfirewall.FirewallPolicy{
			StatefulRuleGroupReferences: []*networkfirewall.StatefulRuleGroupReference{&networkfirewall.StatefulRuleGroupReference{
				ResourceArn: statefulRuleGroup.RuleGroupResponse.RuleGroupArn},
			},
			StatelessDefaultActions:         []*string{aws.String("aws:forward_to_sfe")},
			StatelessFragmentDefaultActions: []*string{aws.String("aws:forward_to_sfe")},
		},
	}
	testFirewallPolicy, err := c.firewallClient.CreateFirewallPolicy(FirewallPolicyInput)
	if err != nil {
		fmt.Println(err.Error())
		return networkfirewall.CreateFirewallOutput{}, err
	}
	fmt.Println("Successfully created a Firewall Policy")

	testFirewallInput := &networkfirewall.CreateFirewallInput{
		FirewallName:      aws.String("testFirewall"),
		FirewallPolicyArn: testFirewallPolicy.FirewallPolicyResponse.FirewallPolicyArn,
		SubnetMappings: []*networkfirewall.SubnetMapping{&networkfirewall.SubnetMapping{
			SubnetId: FirewallSubnet.Subnet.SubnetId},
		},
		VpcId: aws.String(*Vpc.Vpc.VpcId),
	}

	Firewall, err := c.firewallClient.CreateFirewall(testFirewallInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return networkfirewall.CreateFirewallOutput{}, err
	}
	fmt.Println("Successfully created a Firewall!")
	return *Firewall, nil

}

func (c Client) IsFirewallReady(Firewall string) error {
	err := wait.PollImmediate(2*time.Second, 240 * time.Second, func() (bool, error) {
		DescribeFirewall, _ := c.firewallClient.DescribeFirewall(&networkfirewall.DescribeFirewallInput{
			FirewallName: aws.String(Firewall),
		})
		fmt.Println("Current Firewall Status: ", *DescribeFirewall.FirewallStatus.Status)
		if *DescribeFirewall.FirewallStatus.Status == "READY" {
			return true, nil
		}
		return false, nil
	})
	return err
}

func (c Client) CreateRouteToFirewall(CidrBlock string, VPCEndpointId string, RouteTable ec2.CreateRouteTableOutput) error {
	Ruleinput := &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String(CidrBlock),
		VpcEndpointId:        aws.String(VPCEndpointId),
		RouteTableId:         aws.String(*RouteTable.RouteTable.RouteTableId),
	}

	_, err := c.ec2Client.CreateRoute(Ruleinput)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	return nil
}
