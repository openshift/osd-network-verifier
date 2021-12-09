package aws
import (
	"testing"
	//"context"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	//"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)
type mockedEC2 struct {
	ec2iface.EC2API
	RunInstancesMethod func(*ec2.RunInstancesInput) (*ec2.RunInstancesOutput, error)
}
func (m *mockedEC2) RunInstances (in *ec2.RunInstancesInput) (*ec2.RunInstancesOutput, error) {
	if m.RunInstancesMethod != nil {
		return m.RunInstancesMethod(in)
	}
	return nil, nil
}

func  TestCreateEC2Instance( t *testing.T){
	//creds := credentials.NewStaticCredentialsProvider("dummyID", "dummyPassKey", "dummyToken")
    //region := "us-east-1"
    //cli, err := NewClient(creds, region)
	instanceReq := ec2.RunInstancesInput{
         ImageId:      aws.String("amiID"),
         MaxCount:     aws.Int32(int32(1)),
         MinCount:     aws.Int32(int32(1)),
         InstanceType: ec2Types.InstanceType("t2.micro"),
         // Because we're making this VPC aware, we also have to include a network interface specification
         NetworkInterfaces: []ec2Types.InstanceNetworkInterfaceSpecification{
             {
                 AssociatePublicIpAddress: aws.Bool(true),
                 DeviceIndex:              aws.Int32(0),
                 SubnetId:                 aws.String("1213213"),
             },
         },
         UserData: aws.String("mock-userdata"),
     }

	ec2 := &mockedEC2{
		RunInstancesMethod: func(*ec2.RunInstancesInput)  (*ec2.RunInstancesOutput, error) {
			return &ec2.RunInstancesOutput{}, nil 
		},
	}
	_  , err := ec2.RunInstances(&instanceReq)

	if err != nil {
			t.Fatalf("Unexpected test couldn't create EC2 Instance: %v", err)
		}
	//instanceID := *instance.Instances[0].InstanceId
	//if instance.Instances[0].InstanceId == nil {
	//		t.Fatalf("Instance should have been initialized")
	//}
}
