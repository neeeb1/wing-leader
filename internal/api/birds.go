package birds

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/neeeb1/rate_birds/internal/database"
	"github.com/rs/zerolog/log"
)

const maxConcurrent = 50
const cacheTimeoutSeconds = 30

type BirdsJson struct {
	Birds    []Bird `json:"entities"`
	Total    int    `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}
type Bird struct {
	Images      []string `json:"images"`
	LengthMin   string   `json:"lengthMin"`
	LengthMax   string   `json:"lengthMax"`
	Name        string   `json:"name"`
	ID          int      `json:"id"`
	SciName     string   `json:"sciName"`
	Region      []string `json:"region"`
	Family      string   `json:"family"`
	Order       string   `json:"order"`
	Status      string   `json:"status"`
	WingspanMin string   `json:"wingspanMin,omitempty"`
	WingspanMax string   `json:"wingspanMax,omitempty"`
}

func (cfg *ApiConfig) PopulateBirdDB() error {
	log.Info().Msg("---* Populating birds db from Nuthatch API *---")

	intialFetch, err := cfg.GetNuthatchBirds(1, 1)
	if err != nil {
		return err
	}

	birdsToFetch := intialFetch.Total
	maxPageSize := 100
	page := 1

	for i := 0; i < birdsToFetch; i += maxPageSize {
		remaining := birdsToFetch - i
		pageSize := int(math.Min(float64(maxPageSize), float64(remaining)))

		birds, err := cfg.GetNuthatchBirds(page, pageSize)
		if err != nil {
			return err
		}

		for _, b := range birds.Birds {
			var imageArray []string
			if len(b.Images) != 0 {
				imageArray = b.Images
			}

			params := database.CreateBirdParams{
				CommonName:     sql.NullString{String: b.Name, Valid: true},
				ScientificName: sql.NullString{String: b.SciName, Valid: true},
				Family:         sql.NullString{String: b.Family, Valid: true},
				Order:          sql.NullString{String: b.Order, Valid: true},
				Status:         sql.NullString{String: b.Status, Valid: true},
				ImageUrls:      imageArray,
			}

			_, err := cfg.DbQueries.CreateBird(context.Background(), params)
			if err != nil {
				return fmt.Errorf("failed to create database entry for bird: %s", err)
			}
		}
		page++
	}
	count, err := cfg.DbQueries.GetTotalBirdCount(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("failed to count birds in db")
	} else {
		log.Info().Msgf("Success - database contains %d bird entries", count)
	}

	return nil
}

func (cfg *ApiConfig) PopulateRatingsDB() error {
	log.Info().Msg("---* Populating ratings db *---")

	birds, err := cfg.DbQueries.GetAllBirds(context.Background())
	if err != nil {
		return err
	}

	for _, b := range birds {
		params := database.PopulateRatingParams{
			Matches: sql.NullInt32{Int32: 0, Valid: true},
			Rating:  sql.NullInt32{Int32: 1000, Valid: true},
			BirdID:  b.ID,
		}

		//log.Info().Msgf("adding %s to ratings with default values\n", b.CommonName.String)

		err = cfg.DbQueries.PopulateRating(context.Background(), params)
		if err != nil {
			return err
		}
	}

	count, err := cfg.DbQueries.GetTotalRatings(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("failed to count ratings in db")
	} else {
		log.Info().Msgf("Success - database contains %d rating entries", count)
	}
	return nil
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

	client := &http.Client{
		Timeout: cacheTimeoutSeconds * time.Second,
	}

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

				cacheUrl := fmt.Sprintf("http://%s:1337/200x200,sc/%s", cfg.CacheHost, u)

				res, err := client.Get(cacheUrl)
				if err != nil {
					log.Error().Err(err).Msgf("error caching image url (%s)", u)
					mu.Lock()
					totalFailure += 1
					mu.Unlock()
					return
				}

				io.Copy(io.Discard, res.Body)
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
				mu.Unlock()
			}(u)
		}
	}

	wg.Wait()

	log.Info().Msgf("Successfully cached %d images", totalSuccess)
	log.Info().Msgf("%d images failed to cache", totalFailure)
	timeElapsed := time.Since(startTime)
	log.Debug().Msgf("Total cachine time: %s", timeElapsed)
	log.Info().Msg("---* Initial image caching completed --*")

	return nil
}
