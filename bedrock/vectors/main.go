// alpaca/bedrock/vectors/main.go
package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"google.golang.org/api/iterator"
)

type VectorDoc struct {
	ID       string                 `json:"id"`
	Vector   []float32              `json:"vector"`
	Metadata map[string]interface{} `json:"metadata"`
}

const (
	embeddingModelID = "amazon.titan-embed-text-v1"  // Or "cohere.embed-english-v3.0" (1024 dim)
	bucketName       = "alpaca-vectors-your-account" // Unique! Append acct ID
	s3Prefix         = "vectors/reviews.jsonl.gz"
	chunkTokens      = 512      // Approx words/8
	similarityMetric = "cosine" // For later KB/search
)

func main() {
	ctx := context.Background()

	// GCP BQ (ENV GCP creds)
	bqClient, err := bigquery.NewClient(ctx, "your-gcp-project")
	if err != nil {
		log.Fatal("BQ client:", err)
	}
	defer bqClient.Close()

	// AWS (ENV AWS_* )
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatal("AWS config:", err)
	}
	s3Client := s3.NewFromConfig(awsCfg)
	bedrockClient := bedrockruntime.NewFromConfig(awsCfg)
	uploader := manager.NewUploader(s3Client)

	// Create S3 bucket if not exist
	_, err = s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint("us-east-1"),
		},
	})
	if err != nil && !strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") && !strings.Contains(err.Error(), "BucketAlreadyExists") {
		log.Fatal("S3 bucket create:", err)
	}
	log.Printf("✅ S3 bucket %s ready", bucketName)

	// Fetch BQ data (join hotels/cities if needed)
	reviews := fetchReviews(ctx, bqClient) // Impl below
	log.Printf("Loaded %d reviews from BQ", len(reviews))

	// Embed + chunk + upload
	if err := embedAndUpload(ctx, bedrockClient, uploader, bucketName, reviews); err != nil {
		log.Fatal("Embed/upload:", err)
	}
	log.Println("✅ Vectors in S3 %s/%s", bucketName, s3Prefix)

	// create index and query

	if err := createIndex(ctx, s3Client, bucketName); err != nil {
		log.Fatal(err)
	}

	// Wait index ready (poll Status)
	time.Sleep(30 * time.Second) // Or poll DescribeVectorIndex

	// Query demo
	neighbors, err := queryNeighbors(ctx, s3Client, bucketName, []float32{0.1, 0.2 /* embed "NYC hotel" */}, 5)
	log.Printf("Top 5: %+v", neighbors)

}

// fetchReviews: BQ query (join hotels?)
func fetchReviews(ctx context.Context, client *bigquery.Client) []map[string]interface{} {
	q := client.Query(`
        SELECT source_reviews_id as id, review_text, rating, hotel_id, city_id, 
               h.name as hotel_name, c.name as city_name
        FROM dataset.hotel_reviews r
        LEFT JOIN dataset.hotels h ON r.hotel_id = h.id
        LEFT JOIN dataset.cities c ON h.city_id = c.id
    `)
	it, err := q.Read(ctx)
	if err != nil {
		log.Fatal("BQ query:", err)
	}
	var reviews []map[string]interface{}
	for {
		var row map[string]bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Row err: %v", err)
			continue
		}
		reviews = append(reviews, row)
	}
	return reviews
}

// embedAndUpload
func embedAndUpload(ctx context.Context, bedrock *bedrockruntime.Client, uploader *manager.Uploader, bucket string, reviews []map[string]interface{}) error {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	enc := json.NewEncoder(gw)
	for _, review := range reviews {
		text := review["review_text"].(string)
		chunks := chunkText(text, chunkTokens)
		for ci, chunk := range chunks {
			vector, err := embedText(ctx, bedrock, chunk)
			if err != nil {
				log.Printf("Embed fail %s: %v", review["id"], err)
				continue
			}
			doc := VectorDoc{
				ID:       fmt.Sprintf("%s_chunk%d", review["id"], ci),
				Vector:   vector,
				Metadata: review, // Full meta
			}
			if err := enc.Encode(doc); err != nil {
				log.Printf("Encode fail: %v", err)
			}
		}
	}
	gw.Close()

	// Upload
	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:          aws.String(bucket),
		Key:             aws.String(s3Prefix),
		Body:            bytes.NewReader(buf.Bytes()),
		ContentType:     aws.String("application/json"),
		ContentEncoding: aws.String("gzip"),
	})
	return err
}

// chunkText simple word split
func chunkText(text string, maxTokens int) []string {
	words := strings.Fields(text)
	chunks := []string{}
	cur := []string{}
	for _, w := range words {
		cur = append(cur, w)
		if len(cur)*4/5 > maxTokens { // Rough token est
			chunks = append(chunks, strings.Join(cur, " "))
			cur = nil
		}
	}
	if len(cur) > 0 {
		chunks = append(chunks, strings.Join(cur, " "))
	}
	return chunks
}

// embedText Titan (1536 dim)
func embedText(ctx context.Context, client *bedrockruntime.Client, text string) ([]float32, error) {
	input := map[string]interface{}{
		"inputText": text,
	}
	inputJSON, _ := json.Marshal(input)
	out, err := client.InvokeModelWithResponse(ctx, &bedrockruntime.InvokeModelWithResponseInput{
		ModelId:     aws.String(embeddingModelID),
		Body:        bytes.NewReader(inputJSON),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("*/*"),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()

	var resp map[string]interface{}
	if err := json.NewDecoder(out.Body).Decode(&resp); err != nil {
		return nil, err
	}
	embeddingIface := resp["embedding"].([]interface{})
	vector := make([]float32, len(embeddingIface))
	for i, v := range embeddingIface {
		vector[i] = float32(v.(float64))
	}
	return vector, nil
}

// ... imports + prev code (embedAndUpload to S3 JSONL)

const (
	indexName = "hotel-reviews-index"
	dim       = 1536     // Titan v1
	metric    = "cosine" // /dot/euclid
)

func createIndex(ctx context.Context, s3Client *s3.Client, bucket string) error {
	_, err := s3Client.CreateVectorIndex(ctx, &s3.CreateVectorIndexInput{
		Bucket:    aws.String(bucket),
		Name:      aws.String(indexName),
		Metric:    types.VectorMetric(metric), // types.VectorMetricCosine
		Dimension: aws.Int32(dim),
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}
	log.Printf("✅ Index %s created (dim=%d, %s)", indexName, dim, metric)
	return nil
}
