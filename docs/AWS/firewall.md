# Firewall Config Script

A Go script to create a VPC resources and a Firewall for osd-network-verifier testing

## Overview

[firewallConfig.go](../../examples/aws/firewall/firewallConfig.go) script will create the resources and networking required for a VPC and Firewall, which are
1. VPC with CIDR range 10.0.0.0/16
2. An Internet Gateway
3. A Public Subnet, a Private Subnet, and a Firewall Subnet
4. A Route Table for each of these components: Public Subnet, Private Subnet, Firewall Subnet, and Internet Gateway Subnet. Each route table will have the necessary route and destination
5. A NAT Gateway
6. A Firewall with a stateful rule group and a firewall policy

### IAM permissions ###

Ensure that the AWS credentials being used have the following permissions.

```json
 {
   "Version": "2012-10-17",
   "Statement": [
     {
       "Effect": "Allow",
       "Action": [
         "ec2:CreateVpc",
         "ec2:CreateSubnet",
         "ec2:CreateInternetGateway",
         "ec2:CreateRouteTable",
         "ec2:CreateRoute",
         "ec2:CreateNatGateway",
         "network-firewall:DescribeFirewall",
         "network-firewall:CreateFirewall",
         "network-firewall:CreateFirewallPolicy",
         "network-firewall:CreateRuleGroup",
       ],
       "Resource": "*"
     }
   ]
 }
```



## How to run the script

-`go build firewallConfig.go`: create the binary
- Currently the script supports 3 ways of passing in the aws credentials:
1. ./firewallConfig -p $your-profile -r $region
2. ./firewallConfig AWS_PROFILE=$profile AWS_REGION=$region
3. ./firewallConfig AWS_ACCESS_KEY_ID= AWS_SECRET_ACCESS_KEY= AWS_SESSION_TOKEN= REGION=

## Clean up resources created by the script

### Remove the resources in the order suggested
- Remove all routes in all the routes tables
- Delete NAT Gateway
- Delete the Firewall, it will take about 5 minutes to be deleted
- After the Firewall is done deleting, delete the three subnets
- Detach the Internet Gateway from the VPC and delete the Internet Gateway
- Delete the route tables (the one that’s marked main will be deleted with the vpc)-> 4 route tables to delete
- Delete the VPC
- Delete the firewall policy
- Delete the rule group

