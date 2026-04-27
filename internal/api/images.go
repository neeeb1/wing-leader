package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/disintegration/imaging"
	"github.com/neeeb1/rate_birds/internal/database"
	"github.com/rs/zerolog/log"
)

const presignTTL = 23 * time.Hour

// PresignImageURL returns a presigned URL for a Tigris object, cached for 23 hours.
// Caching ensures all users get the same URL for the same image, allowing browsers
// to cache the fetched image across page loads.
// Falls back to the stored URL unchanged for non-Tigris images.
func (cfg *ApiConfig) PresignImageURL(storedURL string) string {
	if cfg.S3Client == nil || !strings.Contains(storedURL, "tigris") {
		return storedURL
	}

	// Extract object key from stored URL (either domain format)
	var key string
	for _, prefix := range []string{
		"https://" + cfg.BucketName + ".t3.tigrisfiles.io/",
		"https://" + cfg.BucketName + ".fly.storage.tigris.dev/",
	} {
		if strings.HasPrefix(storedURL, prefix) {
			key = strings.TrimPrefix(storedURL, prefix)
			break
		}
	}
	if key == "" {
		return storedURL
	}

	// Check cache
	cfg.presignMu.RLock()
	if cfg.presignCache != nil {
		if entry, ok := cfg.presignCache[key]; ok && time.Now().Before(entry.expires) {
			cfg.presignMu.RUnlock()
			return entry.url
		}
	}
	cfg.presignMu.RUnlock()

	// Generate new presigned URL
	presignClient := s3.NewPresignClient(cfg.S3Client)
	req, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(cfg.BucketName),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = 24 * time.Hour
	})
	if err != nil {
		return storedURL
	}

	cfg.presignMu.Lock()
	if cfg.presignCache == nil {
		cfg.presignCache = make(map[string]presignEntry)
	}
	cfg.presignCache[key] = presignEntry{url: req.URL, expires: time.Now().Add(presignTTL)}
	cfg.presignMu.Unlock()

	return req.URL
}

func (cfg *ApiConfig) CacheImages() error {
	if cfg.S3Client == nil {
		log.Info().Msg("S3 client not configured, skipping image caching")
		return nil
	}

	log.Info().Msg("---* Starting initial image caching *---")
	startTime := time.Now()

	birds, err := cfg.DbQueries.GetBirdsForCaching(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get birds for caching: %w", err)
	}

	var totalCached, totalConverted, totalSkipped, totalFailure int
	semaphore := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, bird := range birds {
		bird := bird
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			newURLs := make([]string, 0, len(bird.ImageUrls))
			changed := false

			for i, u := range bird.ImageUrls {
				// Already using the correct public CDN domain
				if strings.Contains(u, ".t3.tigrisfiles.io/") {
					newURLs = append(newURLs, u)
					mu.Lock()
					totalSkipped++
					mu.Unlock()
					continue
				}

				// Old format (S3 API domain) — convert in-place, no re-upload needed
				if strings.Contains(u, ".fly.storage.tigris.dev/") {
					corrected := strings.Replace(u,
						cfg.BucketName+".fly.storage.tigris.dev/",
						cfg.BucketName+".t3.tigrisfiles.io/", 1)
					newURLs = append(newURLs, corrected)
					changed = true
					mu.Lock()
					totalConverted++
					mu.Unlock()
					continue
				}

				// External URL — download and upload to Tigris
				key := fmt.Sprintf("birds/%s/%d", bird.ID.String(), i)
				tigrisURL := fmt.Sprintf("https://%s.t3.tigrisfiles.io/%s", cfg.BucketName, key)

				if err := cfg.uploadImageToS3(u, key); err != nil {
					log.Error().Err(err).Str("url", u).Str("key", key).Msg("failed to upload image to Tigris")
					newURLs = append(newURLs, u)
					mu.Lock()
					totalFailure++
					mu.Unlock()
					continue
				}

				newURLs = append(newURLs, tigrisURL)
				changed = true
				mu.Lock()
				totalCached++
				mu.Unlock()
				log.Info().Str("key", key).Msg("image cached to Tigris")
			}

			if changed {
				params := database.UpdateBirdImageUrlsParams{
					ID:        bird.ID,
					ImageUrls: newURLs,
				}
				if err := cfg.DbQueries.UpdateBirdImageUrls(context.Background(), params); err != nil {
					log.Error().Err(err).Str("birdID", bird.ID.String()).Msg("failed to update bird image URLs in DB")
				}
			}
		}()
	}

	wg.Wait()

	log.Info().
		Int("cached", totalCached).
		Int("converted", totalConverted).
		Int("skipped", totalSkipped).
		Int("failed", totalFailure).
		Dur("elapsed", time.Since(startTime)).
		Msg("---* Image caching completed *---")

	return nil
}

func (cfg *ApiConfig) uploadImageToS3(sourceURL, key string) error {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(sourceURL)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	img, err := imaging.Decode(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	resized := imaging.Fit(img, 500, 500, imaging.Lanczos)

	var buf bytes.Buffer
	if err := imaging.Encode(&buf, resized, imaging.JPEG, imaging.JPEGQuality(85)); err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}

	_, err = cfg.S3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.BucketName),
		Key:         aws.String(key),
		Body:        &buf,
		ContentType: aws.String("image/jpeg"),
		ACL:         types.ObjectCannedACLPublicRead,
	})
	return err
}
