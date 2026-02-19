package api

import (
	"bytes"
	"embed"
	"net/http"
	"text/template"

	"github.com/google/uuid"
	"github.com/neeeb1/rate_birds/internal/database"
	"github.com/rs/zerolog/log"
)

//go:embed templates/*.html
var embedTemplates embed.FS

func (cfg *ApiConfig) handleBirdDetail(w http.ResponseWriter, r *http.Request) {
	param := r.PathValue("id")
	birdID, err := uuid.Parse(param)
	if err != nil {
		log.Error().Err(err).Msgf("unable to parse uuid: %s", param)
		return
	}

	bird, err := cfg.DbQueries.GetBirdByID(r.Context(), birdID)
	if err != nil {
		log.Error().Err(err).Msg("unable to get bird by UUID")
	}

	payload, err := buildBirdDetail(bird)

	w.Write(payload.Bytes())
}

func buildBirdDetail(bird database.Bird) (bytes.Buffer, error) {
	var payload bytes.Buffer

	tmpl, err := template.ParseFS(embedTemplates, "detail.html")
	if err != nil {
		log.Error().Err(err).Msg("failed to build bird template")
	}

	// Time format string
	// Mon Jan 2 15:04:05 MST 2006
	err = tmpl.Execute(&payload, struct {
		LastUpdated    string
		CommonName     string
		ScientificName string
		Family         string
		Order          string
		Status         string
		Image          string
	}{
		bird.UpdatedAt.Format("Mon, Jan 2 3:04PM"),
		bird.CommonName.String,
		bird.ScientificName.String,
		bird.Family.String,
		bird.Order.String,
		bird.Status.String,
		bird.ImageUrls[0],
	})
	if err != nil {
		return payload, err
	}

	return payload, nil
}
