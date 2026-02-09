package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const defaultImageSize = "500x500"

func (cfg *ApiConfig) handleCachedImage(w http.ResponseWriter, r *http.Request) {
	imageURL := r.URL.Query().Get("url")
	if imageURL == "" {
		RespondWithError(w, 400, "missing url parameter")
		return
	}

	cacheURL := cfg.buildImageProxyURL(imageURL)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := client.Get(cacheURL)
	if err != nil {
		http.Error(w, "failed to fetch image", http.StatusInternalServerError)
		RespondWithError(w, 500, "Failed to fetch image")
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		http.Error(w, "image not found", res.StatusCode)
		return
	}

	w.Header().Set("Content-Type", res.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", res.Header.Get("Cache-Control"))

	io.Copy(w, res.Body)
}

func (cfg *ApiConfig) CacheImages() error {
	log.Info().Msg("---* Starting initial image caching --*")
	startTime := time.Now()

	imageUrls, err := cfg.DbQueries.GetAllImageUrls(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get all image urls in db: %s", err)
	}

	var totalSuccess int
	var totalFailure int
	var totalBytes int64

	semaphore := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, urls := range imageUrls {
		for _, u := range urls {
			wg.Add(1)

			go func(url string) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				cacheUrl := cfg.buildImageProxyURL(url)

				res, err := http.Get(cacheUrl)
				if err != nil {
					log.Error().Err(err).Msgf("error caching image url (%s)", url)
					mu.Lock()
					totalFailure += 1
					mu.Unlock()
					return
				}

				written, err := io.Copy(io.Discard, res.Body)
				if err != nil {
					log.Error().Err(err).Msgf("error reading response body for url (%s)", u)
				}
				res.Body.Close()

				if res.StatusCode == http.StatusNotFound {
					log.Info().Msgf("response: not found for url (%s)", u)
					mu.Lock()
					totalFailure += 1
					mu.Unlock()
					return
				}

				log.Info().Msgf("Image cached (%s)", u)
				mu.Lock()
				totalSuccess += 1
				totalBytes += written
				mu.Unlock()
			}(u)
		}
	}

	wg.Wait()

	log.Info().Msgf("Successfully cached %d images", totalSuccess)
	log.Info().Msgf("%d images failed to cache", totalFailure)
	log.Info().Msgf("Total bytes cached: %d", totalBytes)
	timeElapsed := time.Since(startTime)
	log.Debug().Msgf("Total cachine time: %s", timeElapsed)
	log.Info().Msg("---* Initial image caching completed --*")

	return nil
}

func (cfg *ApiConfig) buildImageProxyURL(imageURL string) string {
	return fmt.Sprintf("http://%s:1337/%s,sc/%s", cfg.CacheHost, defaultImageSize, imageURL)
}
