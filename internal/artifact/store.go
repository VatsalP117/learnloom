package artifact

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/VatsalP117/learnloom/internal/domain"
	"github.com/VatsalP117/learnloom/internal/dossier"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

const (
	maximumArtifactBytes       = 20 << 20
	maximumStoredArtifactBytes = 21 << 20
	artifactFormatVersion      = 1
	artifactRenderVersion      = 1
)

var ErrNotFound = errors.New("Dossier Artifact not found")

type Config struct {
	Bucket          string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
	CacheBytes      int64
}

type Store struct {
	bucket string
	client *s3.Client
	cache  *artifactCache
}

func New(ctx context.Context, cfg Config) (*Store, error) {
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, errors.New("artifact bucket is required")
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.CacheBytes == 0 {
		cfg.CacheBytes = 64 << 20
	}
	options := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}
	if cfg.AccessKeyID != "" || cfg.SecretAccessKey != "" {
		if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
			return nil, errors.New("both artifact access key fields are required")
		}
		options = append(options, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("load artifact configuration: %w", err)
	}
	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.UsePathStyle = cfg.UsePathStyle
		if cfg.Endpoint != "" {
			options.BaseEndpoint = aws.String(strings.TrimRight(cfg.Endpoint, "/"))
		}
	})
	return &Store{
		bucket: cfg.Bucket,
		client: client,
		cache:  newArtifactCache(cfg.CacheBytes),
	}, nil
}

type PutInput struct {
	AccountID    string
	NewsletterID string
	IssueID      string
	GenerationID string
	Artifact     domain.DossierArtifact
}

type PutResult struct {
	Key      string
	Checksum string
	Bytes    int
}

func KeyFor(accountID, newsletterID, issueID, generationID string) (string, error) {
	for name, value := range map[string]string{
		"Account ID": accountID, "Newsletter ID": newsletterID,
		"Issue ID": issueID, "Generation ID": generationID,
	} {
		if !safePart(value) {
			return "", fmt.Errorf("%s is invalid", name)
		}
	}
	return path.Join(
		"accounts", accountID,
		"newsletters", newsletterID,
		"issues", issueID,
		generationID+".json.gz",
	), nil
}

type storedArtifact struct {
	FormatVersion int            `json:"formatVersion"`
	RenderVersion int            `json:"renderVersion"`
	Dossier       domain.Dossier `json:"dossier"`
}

func (s *Store) Put(ctx context.Context, input PutInput) (PutResult, error) {
	key, err := KeyFor(
		input.AccountID, input.NewsletterID, input.IssueID, input.GenerationID,
	)
	if err != nil {
		return PutResult{}, err
	}
	if input.Artifact.Dossier.Version < 1 {
		return PutResult{}, errors.New("Dossier Artifact is incomplete")
	}
	body, err := json.Marshal(storedArtifact{
		FormatVersion: artifactFormatVersion,
		RenderVersion: artifactRenderVersion,
		Dossier:       input.Artifact.Dossier,
	})
	if err != nil {
		return PutResult{}, fmt.Errorf("marshal Dossier Artifact: %w", err)
	}
	if len(body) > maximumArtifactBytes {
		return PutResult{}, errors.New("Dossier Artifact exceeds the size limit")
	}
	checksum := sha256.Sum256(body)
	compressed, err := compressArtifact(body)
	if err != nil {
		return PutResult{}, fmt.Errorf("compress Dossier Artifact: %w", err)
	}
	storedChecksum := sha256.Sum256(compressed)
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:          aws.String(s.bucket),
		Key:             aws.String(key),
		Body:            bytes.NewReader(compressed),
		ContentLength:   aws.Int64(int64(len(compressed))),
		ContentType:     aws.String("application/json"),
		ContentEncoding: aws.String("gzip"),
		CacheControl:    aws.String("private, max-age=31536000, immutable"),
		ChecksumSHA256:  aws.String(base64.StdEncoding.EncodeToString(storedChecksum[:])),
		Metadata: map[string]string{
			"sha256": hex.EncodeToString(checksum[:]),
		},
	})
	if err != nil {
		return PutResult{}, fmt.Errorf("store Dossier Artifact: %w", err)
	}
	artifactValue := renderArtifact(input.Artifact.Dossier)
	s.cache.put(key, artifactValue, cacheBytes(artifactValue, len(body)))
	return PutResult{
		Key: key, Checksum: hex.EncodeToString(checksum[:]), Bytes: len(compressed),
	}, nil
}

func (s *Store) Get(ctx context.Context, key string) (domain.DossierArtifact, error) {
	if !safeKey(key) {
		return domain.DossierArtifact{}, errors.New("Dossier Artifact key is invalid")
	}
	if artifact, ok := s.cache.get(key); ok {
		return artifact, nil
	}
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var apiError smithy.APIError
		if errors.As(err, &apiError) &&
			(apiError.ErrorCode() == "NoSuchKey" || apiError.ErrorCode() == "NotFound") {
			return domain.DossierArtifact{}, fmt.Errorf("%w: %s", ErrNotFound, key)
		}
		return domain.DossierArtifact{}, fmt.Errorf("load Dossier Artifact: %w", err)
	}
	defer result.Body.Close()
	compressed, err := io.ReadAll(io.LimitReader(result.Body, maximumStoredArtifactBytes+1))
	if err != nil {
		return domain.DossierArtifact{}, fmt.Errorf("read Dossier Artifact: %w", err)
	}
	if len(compressed) > maximumStoredArtifactBytes {
		return domain.DossierArtifact{}, errors.New("stored Dossier Artifact exceeds the size limit")
	}
	if !strings.EqualFold(strings.TrimSpace(aws.ToString(result.ContentEncoding)), "gzip") {
		return domain.DossierArtifact{}, errors.New("stored Dossier Artifact encoding is invalid")
	}
	body, err := decompressArtifact(compressed)
	if err != nil {
		return domain.DossierArtifact{}, fmt.Errorf("decompress Dossier Artifact: %w", err)
	}
	checksum := sha256.Sum256(body)
	expected := strings.ToLower(strings.TrimSpace(result.Metadata["sha256"]))
	if expected == "" || !strings.EqualFold(expected, hex.EncodeToString(checksum[:])) {
		return domain.DossierArtifact{}, errors.New("stored Dossier Artifact checksum is invalid")
	}
	var stored storedArtifact
	if err := json.Unmarshal(body, &stored); err != nil {
		return domain.DossierArtifact{}, fmt.Errorf("decode Dossier Artifact: %w", err)
	}
	if stored.FormatVersion != artifactFormatVersion ||
		stored.RenderVersion != artifactRenderVersion {
		return domain.DossierArtifact{}, errors.New("stored Dossier Artifact version is unsupported")
	}
	if stored.Dossier.Version < 1 {
		return domain.DossierArtifact{}, errors.New("stored Dossier Artifact is incomplete")
	}
	artifactValue := renderArtifact(stored.Dossier)
	s.cache.put(key, artifactValue, cacheBytes(artifactValue, len(body)))
	return artifactValue, nil
}

func compressArtifact(body []byte) ([]byte, error) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(body); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return compressed.Bytes(), nil
}

func decompressArtifact(compressed []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	body, readErr := io.ReadAll(io.LimitReader(reader, maximumArtifactBytes+1))
	closeErr := reader.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if len(body) > maximumArtifactBytes {
		return nil, errors.New("decompressed Dossier Artifact exceeds the size limit")
	}
	return body, nil
}

func renderArtifact(value domain.Dossier) domain.DossierArtifact {
	return domain.DossierArtifact{
		Dossier:  value,
		Markdown: dossier.RenderMarkdown(value),
		HTML:     dossier.RenderHTML(value, ""),
	}
}

func cacheBytes(artifactValue domain.DossierArtifact, canonicalBytes int) int64 {
	return int64(canonicalBytes + len(artifactValue.Markdown) + len(artifactValue.HTML))
}

func (s *Store) Delete(ctx context.Context, key string) error {
	if !safeKey(key) {
		return errors.New("Dossier Artifact key is invalid")
	}
	s.cache.remove(key)
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete Dossier Artifact: %w", err)
	}
	return nil
}

func (s *Store) DeleteAccount(ctx context.Context, accountID string) error {
	if !safePart(accountID) {
		return errors.New("Account ID is invalid")
	}
	prefix := path.Join("accounts", accountID) + "/"
	var continuation *string
	for {
		list, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(s.bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: continuation,
		})
		if err != nil {
			return fmt.Errorf("list Account artifacts: %w", err)
		}
		if len(list.Contents) > 0 {
			objects := make([]types.ObjectIdentifier, 0, len(list.Contents))
			for _, object := range list.Contents {
				objects = append(objects, types.ObjectIdentifier{Key: object.Key})
			}
			deleted, deleteErr := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(s.bucket),
				Delete: &types.Delete{Objects: objects, Quiet: aws.Bool(true)},
			})
			if deleteErr != nil {
				return fmt.Errorf("delete Account artifacts: %w", deleteErr)
			}
			if len(deleted.Errors) > 0 {
				return fmt.Errorf(
					"delete Account artifacts: object store rejected %d objects",
					len(deleted.Errors),
				)
			}
		}
		if !aws.ToBool(list.IsTruncated) {
			return nil
		}
		continuation = list.NextContinuationToken
	}
}

func (s *Store) Ready(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	if err != nil {
		return fmt.Errorf("artifact bucket readiness: %w", err)
	}
	return nil
}

func safePart(value string) bool {
	if value == "" || value == "." || value == ".." || len(value) > 200 {
		return false
	}
	return !strings.ContainsAny(value, `/\`)
}

func safeKey(value string) bool {
	if value == "" || strings.HasPrefix(value, "/") || len(value) > 1024 {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if !safePart(part) {
			return false
		}
	}
	return true
}
