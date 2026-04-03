package vertex

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"cloud.google.com/go/storage"
)

func UploadLoadBatches[T any](ctx context.Context, s *BQ, gcsClient *storage.Client, bucketName, tableName string, data []T) error {
	if len(data) == 0 {
		return nil
	}

	if err := ensureBucket(ctx, gcsClient, bucketName); err != nil {
		return err
	}

	// Temp file
	timestamp := time.Now().UTC().Format("20060102_150405")
	gcsPath := fmt.Sprintf("temp/%s_%s.jsonl.gz", tableName, timestamp)
	gcsURI := fmt.Sprintf("gs://%s/%s", bucketName, gcsPath)

	// Serialize JSONL + GZIP (efficient)
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	enc := json.NewEncoder(gw)
	for _, item := range data {
		if err := enc.Encode(item); err != nil {
			gw.Close()
			return fmt.Errorf("marshal row: %w", err)
		}
	}
	if err := gw.Close(); err != nil {
		return err
	}

	// Upload GCS
	w := gcsClient.Bucket(bucketName).Object(gcsPath).NewWriter(ctx)
	w.ContentType = "application/json"
	w.ContentEncoding = "gzip"
	if _, err := io.Copy(w, &buf); err != nil {
		w.Close()
		return fmt.Errorf("GCS upload: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("GCS close: %w", err)
	}
	log.Printf("📤 GCS uploaded %s (%d bytes gzipped)", gcsURI, buf.Len())

	// BQ LoadJob
	table := s.BQClient.Dataset(s.DatasetID).Table(tableName)
	loader := table.LoaderFrom(gcsURI).
		SkipInvalidRows(true).
		IgnoreUnknownValues(true)

	job, err := loader.Run(ctx)
	if err != nil {
		return fmt.Errorf("LoadJob run: %w", err)
	}

	status, err := job.Wait(ctx)
	if err != nil {
		return fmt.Errorf("LoadJob wait: %w", err)
	}
	if status.Err() != nil {
		return fmt.Errorf("LoadJob failed: %v", status.Err())
	}
	log.Printf("✅ LoadJob %s: %d rows loaded to %s", job.ID(), status.NumRows, tableName)

	// Cleanup GCS
	if err := gcsClient.Bucket(bucketName).Object(gcsPath).Delete(ctx); err != nil {
		log.Printf("⚠️ GCS cleanup warn: %v", err)
	}

	return nil
}

func ensureBucket(ctx context.Context, client *storage.Client, bucketName string) error {
	bucket := client.Bucket(bucketName)
	if attrs := bucket.Attrs(ctx); attrs == nil {
		if err := bucket.Create(ctx, s.ProjectID, &storage.BucketAttrs{
			Location: "US", // Tune
		}); err != nil {
			return fmt.Errorf("create bucket %s: %w", bucketName, err)
		}
		log.Printf("✅ Created GCS bucket %s", bucketName)
	}
	return nil
}
