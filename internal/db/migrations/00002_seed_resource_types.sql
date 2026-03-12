-- +goose Up

-- Seed the resource type catalog with common infrastructure types.
INSERT INTO resource_types (name, slug, category, description, default_spec, spec_schema, enabled) VALUES
(
  'PostgreSQL Database',
  'postgresql',
  'database',
  'Managed PostgreSQL database instance (RDS)',
  '{"instance_class": "db.t3.micro", "allocated_gb": 20, "engine_version": "16.4", "multi_az": false}',
  '{"type": "object", "properties": {"instance_class": {"type": "string", "enum": ["db.t3.micro", "db.t3.small", "db.t3.medium", "db.r6g.large", "db.r6g.xlarge"]}, "allocated_gb": {"type": "integer", "minimum": 20, "maximum": 1000}, "engine_version": {"type": "string", "enum": ["16.4", "15.8", "14.12"]}, "multi_az": {"type": "boolean"}}, "required": ["instance_class", "allocated_gb"]}',
  true
),
(
  'MySQL Database',
  'mysql',
  'database',
  'Managed MySQL database instance (RDS)',
  '{"instance_class": "db.t3.micro", "allocated_gb": 20, "engine_version": "8.0", "multi_az": false}',
  '{"type": "object", "properties": {"instance_class": {"type": "string", "enum": ["db.t3.micro", "db.t3.small", "db.t3.medium", "db.r6g.large"]}, "allocated_gb": {"type": "integer", "minimum": 20, "maximum": 1000}, "engine_version": {"type": "string", "enum": ["8.0", "8.4"]}, "multi_az": {"type": "boolean"}}, "required": ["instance_class", "allocated_gb"]}',
  true
),
(
  'Redis Cache',
  'redis',
  'cache',
  'Managed Redis cache cluster (ElastiCache)',
  '{"node_type": "cache.t3.micro", "num_nodes": 1, "engine_version": "7.0"}',
  '{"type": "object", "properties": {"node_type": {"type": "string", "enum": ["cache.t3.micro", "cache.t3.small", "cache.t3.medium", "cache.r6g.large"]}, "num_nodes": {"type": "integer", "minimum": 1, "maximum": 6}, "engine_version": {"type": "string", "enum": ["7.0", "6.2"]}}, "required": ["node_type"]}',
  true
),
(
  'S3 Bucket',
  's3-bucket',
  'storage',
  'Object storage bucket with encryption and versioning',
  '{"versioning": true, "encryption": "AES256"}',
  '{"type": "object", "properties": {"versioning": {"type": "boolean"}, "encryption": {"type": "string", "enum": ["AES256", "aws:kms"]}}}',
  true
),
(
  'SQS Queue',
  'sqs-queue',
  'messaging',
  'Managed message queue (SQS)',
  '{"fifo": false, "visibility_timeout_seconds": 30, "message_retention_days": 4}',
  '{"type": "object", "properties": {"fifo": {"type": "boolean"}, "visibility_timeout_seconds": {"type": "integer", "minimum": 0, "maximum": 43200}, "message_retention_days": {"type": "integer", "minimum": 1, "maximum": 14}}}',
  true
),
(
  'SNS Topic',
  'sns-topic',
  'messaging',
  'Managed pub/sub notification topic (SNS)',
  '{"fifo": false}',
  '{"type": "object", "properties": {"fifo": {"type": "boolean"}}}',
  true
);

-- Seed a default team and project for development.
INSERT INTO teams (id, name, slug, description) VALUES
  ('default-team', 'Default Team', 'default-team', 'Default team for development')
ON CONFLICT (id) DO NOTHING;

INSERT INTO projects (id, team_id, name, slug, description) VALUES
  ('default-project', 'default-team', 'Default Project', 'default-project', 'Default project for development')
ON CONFLICT (id) DO NOTHING;

-- Create a dev user and link them to the default team/project.
INSERT INTO users (id, email, name, issuer_sub) VALUES
  ('dev-user-001', 'dev@portway.dev', 'Dev User', 'dev|001')
ON CONFLICT (id) DO NOTHING;

INSERT INTO team_members (team_id, user_id, role) VALUES
  ('default-team', 'dev-user-001', 'owner')
ON CONFLICT (team_id, user_id) DO NOTHING;

INSERT INTO memberships (user_id, project_id, role) VALUES
  ('dev-user-001', 'default-project', 'admin')
ON CONFLICT (user_id, project_id) DO NOTHING;

-- +goose Down

DELETE FROM memberships WHERE user_id = 'dev-user-001' AND project_id = 'default-project';
DELETE FROM team_members WHERE team_id = 'default-team' AND user_id = 'dev-user-001';
DELETE FROM users WHERE id = 'dev-user-001';
DELETE FROM projects WHERE id = 'default-project';
DELETE FROM teams WHERE id = 'default-team';
DELETE FROM resource_types WHERE slug IN ('postgresql', 'mysql', 'redis', 's3-bucket', 'sqs-queue', 'sns-topic');
