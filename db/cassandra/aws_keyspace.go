package cassandra

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sigv4-auth-cassandra-gocql-driver-plugin/sigv4"
	"github.com/go-akka/configuration"
	"github.com/gocql/gocql"
	"github.com/rdkcentral/webconfig/common"
	"github.com/rdkcentral/webconfig/security"
)

func awsKeyspaceClient(conf *configuration.Config, testOnly bool) (*CassandraClient, error) {
	var codec *security.AesCodec
	var err error

	// build codec
	if testOnly {
		codec = security.NewTestCodec(conf)
	} else {
		codec, err = security.NewAesCodec(conf)
		if err != nil {
			return nil, common.NewError(err)
		}
	}

	dbconf := conf.GetConfig("webconfig.database.cassandra")

	// init
	hosts := dbconf.GetStringList("hosts")
	cluster := gocql.NewCluster(hosts...)

	cluster.Consistency = gocql.LocalQuorum
	cluster.ProtoVersion = ProtocolVersion
	cluster.DisableInitialHostLookup = DisableInitialHostLookup
	cluster.Timeout = time.Duration(dbconf.GetInt32("timeout_in_sec", 1)) * time.Second
	cluster.ConnectTimeout = time.Duration(dbconf.GetInt32("connect_timeout_in_sec", 1)) * time.Second
	cluster.NumConns = int(dbconf.GetInt32("connections", DefaultConnections))
	cluster.Port = int(dbconf.GetInt64("port", DefaultPort))

	cluster.RetryPolicy = &gocql.DowngradingConsistencyRetryPolicy{
		ConsistencyLevelsToTry: []gocql.Consistency{
			gocql.LocalQuorum,
			gocql.LocalOne,
			gocql.One,
		},
	}

	localDc := dbconf.GetString("local_dc")
	if len(localDc) > 0 {
		cluster.PoolConfig.HostSelectionPolicy = gocql.DCAwareRoundRobinPolicy(localDc)
	}

	awsRegion, err := getAwsRegionForCassandra(dbconf)
	if err != nil {
		return nil, err
	}

	var auth sigv4.AwsAuthenticator = sigv4.NewAwsAuthenticator()
	auth.Region = awsRegion

	isRoleBasedAccessEnabled := dbconf.GetBoolean("role_based_access_enabled")
	if isRoleBasedAccessEnabled {
		sess, err := session.NewSession(&aws.Config{
			Region: aws.String(awsRegion)},
		)
		if err != nil {
			return nil, err
		}

		// Set up the callback to refresh credentials
		auth.CredentialsCallback = func() (sigv4.SigV4Credentials, error) {
			creds, err := sess.Config.Credentials.Get()
			if err != nil {
				return sigv4.SigV4Credentials{}, err
			}

			return sigv4.SigV4Credentials{
				AccessKeyId:     creds.AccessKeyID,
				SecretAccessKey: creds.SecretAccessKey,
				SessionToken:    creds.SessionToken,
			}, nil
		}
	} else {
		auth.AccessKeyId = dbconf.GetString("access_key_id")
		auth.SecretAccessKey = dbconf.GetString("secret_access_key")
	}
	cluster.Authenticator = auth

	awsKeySpaceCaPath := dbconf.GetString("aws_keyspace_ca_path")
	cluster.SslOpts = &gocql.SslOptions{
		CaPath:                 awsKeySpaceCaPath,
		EnableHostVerification: false,
	}

	// check and create test_keyspace
	if testOnly {
		cluster.Keyspace = dbconf.GetString("test_keyspace", DefaultTestKeyspace)
	} else {
		cluster.Keyspace = dbconf.GetString("keyspace", DefaultKeyspace)
	}

	// now point to the real keyspace
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, common.NewError(err)
	}
	session.SetPageSize(int(dbconf.GetInt32("page_size", DefaultPageSize)))

	blockedSubdocIds := conf.GetStringList("webconfig.blocked_subdoc_ids")
	encryptedSubdocIds := conf.GetStringList("webconfig.encrypted_subdoc_ids")
	stateCorrectionEnabled := conf.GetBoolean("webconfig.state_correction_enabled")
	lockRootDocumentEnabled := conf.GetBoolean("webconfig.lock_root_document_enabled")

	return &CassandraClient{
		Session:                 session,
		ClusterConfig:           cluster,
		AesCodec:                codec,
		concurrentQueries:       make(chan bool, dbconf.GetInt32("concurrent_queries", 500)),
		localDc:                 localDc,
		blockedSubdocIds:        blockedSubdocIds,
		encryptedSubdocIds:      encryptedSubdocIds,
		stateCorrectionEnabled:  stateCorrectionEnabled,
		lockRootDocumentEnabled: lockRootDocumentEnabled,
		awsKeyspaceEnabled:      true,
	}, nil
}

func getAwsRegionForCassandra(dbconf *configuration.Config) (string, error) {
	awsRegion := dbconf.GetString("aws_region")

	if len(awsRegion) == 0 {
		awsRegion = os.Getenv("AWS_REGION")
	}

	if len(awsRegion) == 0 {
		return "", fmt.Errorf("%s", "Aws region is not provided")
	}

	return awsRegion, nil
}
