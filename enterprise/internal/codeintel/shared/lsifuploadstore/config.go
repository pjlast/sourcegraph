package lsifuploadstore

import (
	"strings"
	"time"

	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

type Config struct {
	env.BaseConfig

	Backend      string
	ManageBucket bool
	Bucket       string
	TTL          time.Duration

	S3Region          string
	S3Endpoint        string
	S3UsePathStyle    bool
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3SessionToken    string

	GCSProjectID               string
	GCSCredentialsFile         string
	GCSCredentialsFileContents string
}

func (c *Config) Load() {
	// TODO(blobstore): default to blobstore here
	c.Backend = strings.ToLower(c.Get("PRECISE_CODE_INTEL_UPLOAD_BACKEND", "MinIO", "The target file service for code intelligence uploads. S3, GCS, Minio, and Blobstore are supported."))
	c.ManageBucket = c.GetBool("PRECISE_CODE_INTEL_UPLOAD_MANAGE_BUCKET", "false", "Whether or not the client should manage the target bucket configuration.")
	c.Bucket = c.Get("PRECISE_CODE_INTEL_UPLOAD_BUCKET", "lsif-uploads", "The name of the bucket to store LSIF uploads in.")
	c.TTL = c.GetInterval("PRECISE_CODE_INTEL_UPLOAD_TTL", "168h", "The maximum age of an upload before deletion.")

	// TODO(blobstore): remove minio support
	if c.Backend != "minio" && c.Backend != "blobstore" && c.Backend != "s3" && c.Backend != "gcs" {
		c.AddError(errors.Errorf("invalid backend %q for PRECISE_CODE_INTEL_UPLOAD_BACKEND: must be S3, GCS, Minio, or Blobstore", c.Backend))
	}

	// TODO(blobstore): remove minio support
	if c.Backend == "minio" || c.Backend == "blobstore" || c.Backend == "s3" {
		c.S3Region = c.Get("PRECISE_CODE_INTEL_UPLOAD_AWS_REGION", "us-east-1", "The target AWS region.")
		// TODO(blobstore): remove minio support
		c.S3Endpoint = c.Get("PRECISE_CODE_INTEL_UPLOAD_AWS_ENDPOINT", "http://minio:9000", "The target AWS endpoint.")
		c.S3UsePathStyle = c.GetBool("PRECISE_CODE_INTEL_UPLOAD_AWS_USE_PATH_STYLE", "false", "Whether to use path calling (vs subdomain calling).")
		ec2RoleCredentials := c.GetBool("PRECISE_CODE_INTEL_UPLOAD_AWS_USE_EC2_ROLE_CREDENTIALS", "false", "Whether to use the EC2 metadata API, or use the provided static credentials.")

		if !ec2RoleCredentials {
			c.S3AccessKeyID = c.Get("PRECISE_CODE_INTEL_UPLOAD_AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE", "An AWS access key associated with a user with access to S3.")
			c.S3SecretAccessKey = c.Get("PRECISE_CODE_INTEL_UPLOAD_AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", "An AWS secret key associated with a user with access to S3.")
			c.S3SessionToken = c.GetOptional("PRECISE_CODE_INTEL_UPLOAD_AWS_SESSION_TOKEN", "An optional AWS session token associated with a user with access to S3.")
		}
	} else if c.Backend == "gcs" {
		c.GCSProjectID = c.Get("PRECISE_CODE_INTEL_UPLOAD_GCP_PROJECT_ID", "", "The project containing the GCS bucket.")
		c.GCSCredentialsFile = c.GetOptional("PRECISE_CODE_INTEL_UPLOAD_GOOGLE_APPLICATION_CREDENTIALS_FILE", "The path to a service account key file with access to GCS.")
		c.GCSCredentialsFileContents = c.GetOptional("PRECISE_CODE_INTEL_UPLOAD_GOOGLE_APPLICATION_CREDENTIALS_FILE_CONTENT", "The contents of a service account key file with access to GCS.")
	}
}
