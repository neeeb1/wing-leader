package birds

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/neeeb1/rate_birds/internal/database"
	"github.com/rs/zerolog/log"
)

type ApiConfig struct {
	NuthatcherApiKey string
	DbURL            string
	DbQueries        *database.Queries
	Db               *sql.DB
	CacheHost        string
}

func (cfg *ApiConfig) GetNuthatchBirds(page, pageSize int) (BirdsJson, error) {
	log.Info().Msgf("fetching birds from Nuthatch API\npage: %d, pagesize: %d\n", page, pageSize)
	var birdsJson BirdsJson

	url := fmt.Sprintf("https://nuthatch.lastelm.software/v2/birds?page=%d&pageSize=%d", page, pageSize)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create request")
	}
	req.Header.Add("API-Key", cfg.NuthatcherApiKey)
	req.Header.Add("accept", "application/json")

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to make request to Nuthatch API")
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatal().Msgf("statuscode error: %d %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read response body")
	}

	json.Unmarshal(data, &birdsJson)

	return birdsJson, nil
}
