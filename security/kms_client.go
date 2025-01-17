package security

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/go-akka/configuration"
)

func NewKmsClient(conf *configuration.Config) (*kms.KMS, error) {
	awsRegion, err := getAwsRegionForCassandra(conf)
	if err != nil {
		return nil, err
	}

	awsEndpoint := conf.GetString("webconfig.security.kms.endpoint")
	if len(awsEndpoint) == 0 {
		return nil, fmt.Errorf("%s", "AWS KMS endpoint is not provided")
	}

	awsConfig := &aws.Config{
		Region:   aws.String(awsRegion),
		Endpoint: aws.String(awsEndpoint),
	}

	roleBasedAccessEnabled := conf.GetBoolean("webconfig.security.kms.role_based_access_enabled")
	if !roleBasedAccessEnabled {
		accessKeyId := conf.GetString("webconfig.security.kms.access_key_id")
		secretAccessKey := conf.GetString("webconfig.security.kms.secret_access_key")
		sessionToken := conf.GetString("webconfig.security.kms.session_token")
		awsConfig.Credentials = credentials.NewStaticCredentials(accessKeyId, secretAccessKey, sessionToken)
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}

	return kms.New(sess), nil
}

func getAwsRegionForCassandra(conf *configuration.Config) (string, error) {
	awsRegion := conf.GetString("webconfig.security.kms.aws_region")

	if len(awsRegion) == 0 {
		awsRegion = os.Getenv("AWS_REGION")
	}

	if len(awsRegion) == 0 {
		return "", fmt.Errorf("%s", "AWS region for KMS is not provided")
	}

	return awsRegion, nil
}
