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
	if cfg.DbQueries == nil {
		RespondWithError(w, 503, "Database unavailable")
		return
	}
	log.Info().Msg("call to load detail handler")

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

	var imageURL string
	if len(bird.ImageUrls) > 0 {
		imageURL = cfg.PresignImageURL(bird.ImageUrls[0])
	}

	payload, err := buildBirdDetail(bird, imageURL)
	if err != nil {
		log.Error().Err(err).Msgf("Unable to format template")
	}

	w.Write(payload.Bytes())
}

func buildBirdDetail(bird database.Bird, imageURL string) (bytes.Buffer, error) {
	var payload bytes.Buffer

	tmpl, err := template.New("").ParseFS(embedTemplates, "templates/*.html")
	if err != nil {
		log.Error().Err(err).Msg("failed to build bird template")
	}

	if imageURL == "" {
		imageURL = `https://placehold.co/600x600?text=Image\nNot\nFound&font=raleway`
	}

	err = tmpl.ExecuteTemplate(&payload, "detail.html", struct {
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
		imageURL,
	})
	if err != nil {
		return payload, err
	}

	return payload, nil
}
