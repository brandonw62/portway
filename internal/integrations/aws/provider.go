package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	ectypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/rs/zerolog"

	"github.com/portway/portway/internal/jobs"
)

// Provider implements the jobs.Provisioner interface for AWS.
type Provider struct {
	region    string
	accountID string
	rdsClient *rds.Client
	ecClient  *elasticache.Client
	s3Client  *s3.Client
	sqsClient *sqs.Client
	logger    zerolog.Logger
}

// Config holds the AWS provider configuration.
type Config struct {
	Region    string
	AccountID string
	RoleARN   string
}

// New creates a new AWS provider with the given configuration.
func New(ctx context.Context, cfg Config, logger zerolog.Logger) (*Provider, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("aws: load config: %w", err)
	}

	if cfg.RoleARN != "" {
		stsClient := sts.NewFromConfig(awsCfg)
		creds := stscreds.NewAssumeRoleProvider(stsClient, cfg.RoleARN)
		awsCfg.Credentials = aws.NewCredentialsCache(creds)
	}

	return &Provider{
		region:    cfg.Region,
		accountID: cfg.AccountID,
		rdsClient: rds.NewFromConfig(awsCfg),
		ecClient:  elasticache.NewFromConfig(awsCfg),
		s3Client:  s3.NewFromConfig(awsCfg),
		sqsClient: sqs.NewFromConfig(awsCfg),
		logger:    logger,
	}, nil
}

// portwayTags returns the standard set of AWS tags for a resource.
func portwayTags(spec map[string]any) map[string]string {
	tags := map[string]string{}
	if v, ok := spec["resource_id"].(string); ok {
		tags["portway:resource-id"] = v
	}
	if v, ok := spec["project"].(string); ok {
		tags["portway:project"] = v
	}
	if v, ok := spec["team"].(string); ok {
		tags["portway:team"] = v
	}
	return tags
}

func rdsTags(m map[string]string) []rdstypes.Tag {
	tags := make([]rdstypes.Tag, 0, len(m))
	for k, v := range m {
		tags = append(tags, rdstypes.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	return tags
}

func ecTags(m map[string]string) []ectypes.Tag {
	tags := make([]ectypes.Tag, 0, len(m))
	for k, v := range m {
		tags = append(tags, ectypes.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	return tags
}

func s3Tags(m map[string]string) []s3types.Tag {
	tags := make([]s3types.Tag, 0, len(m))
	for k, v := range m {
		tags = append(tags, s3types.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	return tags
}

func sqsTags(m map[string]string) map[string]string {
	return m
}

// --- Spec types for each resource ---

type rdsSpec struct {
	Identifier     string `json:"identifier"`
	Engine         string `json:"engine"`
	EngineVersion  string `json:"engine_version"`
	InstanceClass  string `json:"instance_class"`
	AllocatedGB    int32  `json:"allocated_gb"`
	MasterUser     string `json:"master_user"`
	MasterPassword string `json:"master_password"`
	DBName         string `json:"db_name"`
	SubnetGroupName string `json:"subnet_group_name"`
	VPCSecurityGroupIDs []string `json:"vpc_security_group_ids"`
	// Portway metadata (used for tagging).
	ResourceID string `json:"resource_id"`
	Project    string `json:"project"`
	Team       string `json:"team"`
}

type cacheSpec struct {
	ClusterID        string `json:"cluster_id"`
	Engine           string `json:"engine"`
	EngineVersion    string `json:"engine_version"`
	NodeType         string `json:"node_type"`
	NumNodes         int32  `json:"num_nodes"`
	SubnetGroupName  string `json:"subnet_group_name"`
	SecurityGroupIDs []string `json:"security_group_ids"`
	ResourceID       string `json:"resource_id"`
	Project          string `json:"project"`
	Team             string `json:"team"`
}

type s3Spec struct {
	BucketName string `json:"bucket_name"`
	ResourceID string `json:"resource_id"`
	Project    string `json:"project"`
	Team       string `json:"team"`
}

type sqsSpec struct {
	QueueName              string `json:"queue_name"`
	DelaySeconds           int32  `json:"delay_seconds"`
	MessageRetentionPeriod int32  `json:"message_retention_period"`
	VisibilityTimeout      int32  `json:"visibility_timeout"`
	FifoQueue              bool   `json:"fifo_queue"`
	ResourceID             string `json:"resource_id"`
	Project                string `json:"project"`
	Team                   string `json:"team"`
}

// Provision creates an AWS resource based on the resource type slug.
func (p *Provider) Provision(ctx context.Context, resourceTypeSlug string, spec []byte) (*jobs.ProvisionResult, error) {
	p.logger.Info().Str("resource_type", resourceTypeSlug).Msg("aws: provisioning resource")

	switch resourceTypeSlug {
	case "postgres", "rds-postgres":
		return p.provisionRDS(ctx, spec)
	case "redis", "elasticache-redis":
		return p.provisionElastiCache(ctx, spec)
	case "s3", "s3-bucket":
		return p.provisionS3(ctx, spec)
	case "sqs", "sqs-queue":
		return p.provisionSQS(ctx, spec)
	default:
		return nil, fmt.Errorf("aws: unsupported resource type %q", resourceTypeSlug)
	}
}

// Delete tears down an AWS resource.
func (p *Provider) Delete(ctx context.Context, resourceTypeSlug string, providerRef string) error {
	p.logger.Info().Str("resource_type", resourceTypeSlug).Str("provider_ref", providerRef).Msg("aws: deleting resource")

	switch resourceTypeSlug {
	case "postgres", "rds-postgres":
		return p.deleteRDS(ctx, providerRef)
	case "redis", "elasticache-redis":
		return p.deleteElastiCache(ctx, providerRef)
	case "s3", "s3-bucket":
		return p.deleteS3(ctx, providerRef)
	case "sqs", "sqs-queue":
		return p.deleteSQS(ctx, providerRef)
	default:
		return fmt.Errorf("aws: unsupported resource type %q", resourceTypeSlug)
	}
}

// HealthCheck verifies the AWS resource is healthy.
func (p *Provider) HealthCheck(ctx context.Context, resourceTypeSlug string, providerRef string) error {
	switch resourceTypeSlug {
	case "postgres", "rds-postgres":
		return p.healthCheckRDS(ctx, providerRef)
	case "redis", "elasticache-redis":
		return p.healthCheckElastiCache(ctx, providerRef)
	case "s3", "s3-bucket":
		return p.healthCheckS3(ctx, providerRef)
	case "sqs", "sqs-queue":
		return p.healthCheckSQS(ctx, providerRef)
	default:
		return fmt.Errorf("aws: unsupported resource type %q", resourceTypeSlug)
	}
}

// --- RDS PostgreSQL ---

func (p *Provider) provisionRDS(ctx context.Context, spec []byte) (*jobs.ProvisionResult, error) {
	var s rdsSpec
	if err := json.Unmarshal(spec, &s); err != nil {
		return nil, fmt.Errorf("aws: unmarshal rds spec: %w", err)
	}

	if s.Engine == "" {
		s.Engine = "postgres"
	}
	if s.InstanceClass == "" {
		s.InstanceClass = "db.t3.micro"
	}
	if s.AllocatedGB == 0 {
		s.AllocatedGB = 20
	}

	tags := portwayTags(map[string]any{"resource_id": s.ResourceID, "project": s.Project, "team": s.Team})

	input := &rds.CreateDBInstanceInput{
		DBInstanceIdentifier: aws.String(s.Identifier),
		Engine:               aws.String(s.Engine),
		DBInstanceClass:      aws.String(s.InstanceClass),
		AllocatedStorage:     aws.Int32(s.AllocatedGB),
		MasterUsername:       aws.String(s.MasterUser),
		MasterUserPassword:   aws.String(s.MasterPassword),
		StorageEncrypted:     aws.Bool(true),
		Tags:                 rdsTags(tags),
	}
	if s.EngineVersion != "" {
		input.EngineVersion = aws.String(s.EngineVersion)
	}
	if s.DBName != "" {
		input.DBName = aws.String(s.DBName)
	}
	if s.SubnetGroupName != "" {
		input.DBSubnetGroupName = aws.String(s.SubnetGroupName)
	}
	if len(s.VPCSecurityGroupIDs) > 0 {
		input.VpcSecurityGroupIds = s.VPCSecurityGroupIDs
	}

	out, err := p.rdsClient.CreateDBInstance(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("aws: CreateDBInstance: %w", err)
	}

	arn := aws.ToString(out.DBInstance.DBInstanceArn)
	return &jobs.ProvisionResult{
		ProviderRef: arn,
		Message:     fmt.Sprintf("RDS instance %s creating (ARN: %s)", s.Identifier, arn),
	}, nil
}

func (p *Provider) deleteRDS(ctx context.Context, providerRef string) error {
	identifier := rdsIdentifierFromARN(providerRef)
	_, err := p.rdsClient.DeleteDBInstance(ctx, &rds.DeleteDBInstanceInput{
		DBInstanceIdentifier: aws.String(identifier),
		SkipFinalSnapshot:    aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("aws: DeleteDBInstance: %w", err)
	}
	return nil
}

func (p *Provider) healthCheckRDS(ctx context.Context, providerRef string) error {
	identifier := rdsIdentifierFromARN(providerRef)
	out, err := p.rdsClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(identifier),
	})
	if err != nil {
		return fmt.Errorf("aws: DescribeDBInstances: %w", err)
	}
	if len(out.DBInstances) == 0 {
		return fmt.Errorf("aws: RDS instance %s not found", identifier)
	}
	status := aws.ToString(out.DBInstances[0].DBInstanceStatus)
	if status != "available" {
		return fmt.Errorf("aws: RDS instance %s status is %q (expected available)", identifier, status)
	}
	return nil
}

// rdsIdentifierFromARN extracts the DB instance identifier from an ARN.
// ARN format: arn:aws:rds:<region>:<account>:db:<identifier>
func rdsIdentifierFromARN(arn string) string {
	parts := strings.Split(arn, ":")
	if len(parts) >= 7 {
		return parts[len(parts)-1]
	}
	return arn
}

// --- ElastiCache Redis ---

func (p *Provider) provisionElastiCache(ctx context.Context, spec []byte) (*jobs.ProvisionResult, error) {
	var s cacheSpec
	if err := json.Unmarshal(spec, &s); err != nil {
		return nil, fmt.Errorf("aws: unmarshal cache spec: %w", err)
	}

	if s.Engine == "" {
		s.Engine = "redis"
	}
	if s.NodeType == "" {
		s.NodeType = "cache.t3.micro"
	}
	if s.NumNodes == 0 {
		s.NumNodes = 1
	}

	tags := portwayTags(map[string]any{"resource_id": s.ResourceID, "project": s.Project, "team": s.Team})

	input := &elasticache.CreateCacheClusterInput{
		CacheClusterId: aws.String(s.ClusterID),
		Engine:         aws.String(s.Engine),
		CacheNodeType:  aws.String(s.NodeType),
		NumCacheNodes:  aws.Int32(s.NumNodes),
		Tags:           ecTags(tags),
	}
	if s.EngineVersion != "" {
		input.EngineVersion = aws.String(s.EngineVersion)
	}
	if s.SubnetGroupName != "" {
		input.CacheSubnetGroupName = aws.String(s.SubnetGroupName)
	}
	if len(s.SecurityGroupIDs) > 0 {
		input.SecurityGroupIds = s.SecurityGroupIDs
	}

	out, err := p.ecClient.CreateCacheCluster(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("aws: CreateCacheCluster: %w", err)
	}

	arn := aws.ToString(out.CacheCluster.ARN)
	return &jobs.ProvisionResult{
		ProviderRef: arn,
		Message:     fmt.Sprintf("ElastiCache cluster %s creating (ARN: %s)", s.ClusterID, arn),
	}, nil
}

func (p *Provider) deleteElastiCache(ctx context.Context, providerRef string) error {
	clusterID := elasticacheIDFromARN(providerRef)
	_, err := p.ecClient.DeleteCacheCluster(ctx, &elasticache.DeleteCacheClusterInput{
		CacheClusterId: aws.String(clusterID),
	})
	if err != nil {
		return fmt.Errorf("aws: DeleteCacheCluster: %w", err)
	}
	return nil
}

func (p *Provider) healthCheckElastiCache(ctx context.Context, providerRef string) error {
	clusterID := elasticacheIDFromARN(providerRef)
	out, err := p.ecClient.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{
		CacheClusterId: aws.String(clusterID),
	})
	if err != nil {
		return fmt.Errorf("aws: DescribeCacheClusters: %w", err)
	}
	if len(out.CacheClusters) == 0 {
		return fmt.Errorf("aws: ElastiCache cluster %s not found", clusterID)
	}
	status := aws.ToString(out.CacheClusters[0].CacheClusterStatus)
	if status != "available" {
		return fmt.Errorf("aws: ElastiCache cluster %s status is %q (expected available)", clusterID, status)
	}
	return nil
}

// elasticacheIDFromARN extracts the cluster ID from an ARN.
// ARN format: arn:aws:elasticache:<region>:<account>:cluster:<id>
func elasticacheIDFromARN(arn string) string {
	parts := strings.Split(arn, ":")
	if len(parts) >= 7 {
		return parts[len(parts)-1]
	}
	return arn
}

// --- S3 ---

func (p *Provider) provisionS3(ctx context.Context, spec []byte) (*jobs.ProvisionResult, error) {
	var s s3Spec
	if err := json.Unmarshal(spec, &s); err != nil {
		return nil, fmt.Errorf("aws: unmarshal s3 spec: %w", err)
	}

	createInput := &s3.CreateBucketInput{
		Bucket: aws.String(s.BucketName),
	}
	if p.region != "us-east-1" {
		createInput.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(p.region),
		}
	}

	if _, err := p.s3Client.CreateBucket(ctx, createInput); err != nil {
		return nil, fmt.Errorf("aws: CreateBucket: %w", err)
	}

	// Enable default encryption (AES256).
	if _, err := p.s3Client.PutBucketEncryption(ctx, &s3.PutBucketEncryptionInput{
		Bucket: aws.String(s.BucketName),
		ServerSideEncryptionConfiguration: &s3types.ServerSideEncryptionConfiguration{
			Rules: []s3types.ServerSideEncryptionRule{
				{
					ApplyServerSideEncryptionByDefault: &s3types.ServerSideEncryptionByDefault{
						SSEAlgorithm: s3types.ServerSideEncryptionAes256,
					},
				},
			},
		},
	}); err != nil {
		return nil, fmt.Errorf("aws: PutBucketEncryption: %w", err)
	}

	// Enable versioning.
	if _, err := p.s3Client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(s.BucketName),
		VersioningConfiguration: &s3types.VersioningConfiguration{
			Status: s3types.BucketVersioningStatusEnabled,
		},
	}); err != nil {
		return nil, fmt.Errorf("aws: PutBucketVersioning: %w", err)
	}

	// Block all public access.
	if _, err := p.s3Client.PutPublicAccessBlock(ctx, &s3.PutPublicAccessBlockInput{
		Bucket: aws.String(s.BucketName),
		PublicAccessBlockConfiguration: &s3types.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(true),
			BlockPublicPolicy:     aws.Bool(true),
			IgnorePublicAcls:      aws.Bool(true),
			RestrictPublicBuckets: aws.Bool(true),
		},
	}); err != nil {
		return nil, fmt.Errorf("aws: PutPublicAccessBlock: %w", err)
	}

	// Tag the bucket.
	tags := portwayTags(map[string]any{"resource_id": s.ResourceID, "project": s.Project, "team": s.Team})
	if _, err := p.s3Client.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
		Bucket:  aws.String(s.BucketName),
		Tagging: &s3types.Tagging{TagSet: s3Tags(tags)},
	}); err != nil {
		return nil, fmt.Errorf("aws: PutBucketTagging: %w", err)
	}

	arn := fmt.Sprintf("arn:aws:s3:::%s", s.BucketName)
	return &jobs.ProvisionResult{
		ProviderRef: arn,
		Message:     fmt.Sprintf("S3 bucket %s created with encryption, versioning, and public access blocked", s.BucketName),
	}, nil
}

func (p *Provider) deleteS3(ctx context.Context, providerRef string) error {
	bucket := s3BucketFromARN(providerRef)
	_, err := p.s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return fmt.Errorf("aws: DeleteBucket: %w", err)
	}
	return nil
}

func (p *Provider) healthCheckS3(ctx context.Context, providerRef string) error {
	bucket := s3BucketFromARN(providerRef)
	_, err := p.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return fmt.Errorf("aws: HeadBucket %s: %w", bucket, err)
	}
	return nil
}

// s3BucketFromARN extracts the bucket name from an ARN.
// ARN format: arn:aws:s3:::<bucket>
func s3BucketFromARN(arn string) string {
	const prefix = "arn:aws:s3:::"
	if strings.HasPrefix(arn, prefix) {
		return strings.TrimPrefix(arn, prefix)
	}
	return arn
}

// --- SQS ---

func (p *Provider) provisionSQS(ctx context.Context, spec []byte) (*jobs.ProvisionResult, error) {
	var s sqsSpec
	if err := json.Unmarshal(spec, &s); err != nil {
		return nil, fmt.Errorf("aws: unmarshal sqs spec: %w", err)
	}

	tags := portwayTags(map[string]any{"resource_id": s.ResourceID, "project": s.Project, "team": s.Team})

	attrs := map[string]string{}
	if s.DelaySeconds > 0 {
		attrs[string(sqstypes.QueueAttributeNameDelaySeconds)] = fmt.Sprintf("%d", s.DelaySeconds)
	}
	if s.MessageRetentionPeriod > 0 {
		attrs[string(sqstypes.QueueAttributeNameMessageRetentionPeriod)] = fmt.Sprintf("%d", s.MessageRetentionPeriod)
	}
	if s.VisibilityTimeout > 0 {
		attrs[string(sqstypes.QueueAttributeNameVisibilityTimeout)] = fmt.Sprintf("%d", s.VisibilityTimeout)
	}
	if s.FifoQueue {
		attrs[string(sqstypes.QueueAttributeNameFifoQueue)] = "true"
	}

	input := &sqs.CreateQueueInput{
		QueueName:  aws.String(s.QueueName),
		Attributes: attrs,
		Tags:       sqsTags(tags),
	}

	out, err := p.sqsClient.CreateQueue(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("aws: CreateQueue: %w", err)
	}

	queueURL := aws.ToString(out.QueueUrl)

	// Get the queue ARN.
	attrOut, err := p.sqsClient.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(queueURL),
		AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameQueueArn},
	})
	if err != nil {
		return nil, fmt.Errorf("aws: GetQueueAttributes: %w", err)
	}

	arn := attrOut.Attributes[string(sqstypes.QueueAttributeNameQueueArn)]
	return &jobs.ProvisionResult{
		ProviderRef: arn,
		Message:     fmt.Sprintf("SQS queue %s created (URL: %s)", s.QueueName, queueURL),
	}, nil
}

func (p *Provider) deleteSQS(ctx context.Context, providerRef string) error {
	queueURL, err := p.sqsQueueURLFromARN(ctx, providerRef)
	if err != nil {
		return err
	}
	_, err = p.sqsClient.DeleteQueue(ctx, &sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})
	if err != nil {
		return fmt.Errorf("aws: DeleteQueue: %w", err)
	}
	return nil
}

func (p *Provider) healthCheckSQS(ctx context.Context, providerRef string) error {
	queueURL, err := p.sqsQueueURLFromARN(ctx, providerRef)
	if err != nil {
		return err
	}
	_, err = p.sqsClient.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(queueURL),
		AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameQueueArn},
	})
	if err != nil {
		return fmt.Errorf("aws: SQS queue health check failed: %w", err)
	}
	return nil
}

// sqsQueueURLFromARN derives the queue URL from an ARN.
// ARN format: arn:aws:sqs:<region>:<account>:<queue-name>
func (p *Provider) sqsQueueURLFromARN(ctx context.Context, arn string) (string, error) {
	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return "", fmt.Errorf("aws: invalid SQS ARN %q", arn)
	}
	queueName := parts[len(parts)-1]
	out, err := p.sqsClient.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return "", fmt.Errorf("aws: GetQueueUrl for %s: %w", queueName, err)
	}
	return aws.ToString(out.QueueUrl), nil
}
