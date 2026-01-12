package birds

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func RegisterEndpoints(mux *http.ServeMux, cfg *ApiConfig) {
	mux.Handle("/", http.FileServer(http.Dir("./web")))
	mux.HandleFunc("GET /api/scorematch/", cfg.handleScoreMatch)
	mux.HandleFunc("GET /api/leaderboard/", cfg.handleLoadLeaderboard)
	mux.HandleFunc("GET /api/image/", cfg.handleCachedImage)
	mux.HandleFunc("GET /api/loadbirds/", cfg.handleLoadBirds)
}

func (cfg *ApiConfig) handleScoreMatch(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("call to score match handler")

	leftBirdID, err := uuid.Parse(r.URL.Query().Get("leftBirdID"))
	if err != nil {
		log.Error().Err(err).Msg("failed to parse left bird ID")
	}
	leftBird, err := cfg.DbQueries.GetBirdByID(r.Context(), leftBirdID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get left bird by ID")
	}

	rightBirdID, err := uuid.Parse(r.URL.Query().Get("rightBirdID"))
	if err != nil {
		log.Error().Err(err).Msg("failed to parse right bird ID")
	}
	rightBird, err := cfg.DbQueries.GetBirdByID(r.Context(), rightBirdID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get right bird by ID")
	}

	winner := r.URL.Query().Get("winner")
	switch winner {
	case "left":
		//log.Info().Msgf("Winner: %s, Loser: %s\n", leftBird.CommonName.String, rightBird.CommonName.String)
		cfg.ScoreMatch(leftBird, rightBird)
	case "right":
		//log.Info().Msgf("Winner: %s, Loser: %s\n", rightBird.CommonName.String, leftBird.CommonName.String)
		cfg.ScoreMatch(rightBird, leftBird)
	}

	cfg.handleLoadBirds(w, r)
}

func (cfg *ApiConfig) handleLoadBirds(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("call to load bird handler")
	rng_bird, err := cfg.DbQueries.GetRandomBirdWithImage(r.Context(), 2)
	if err != nil {
		log.Error().Err(err).Msg("failed to get random birds with images")
	}

	newLeftBird := rng_bird[0]
	newRightBird := rng_bird[1]

	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	payload := fmt.Sprintf(
		`<div id="bird-wrapper" class="w-full max-w-6xl p-6 flex flex-row sm:flex-col gap-6 items-stretch justify-center">
            <!-- Left Bird Card -->
			<div class="m-32 shadow-lg rounded-sm w-2/3 p-6 flex flex-col align-items-center bg-zinc-300" id="left-bird">
                <img class="card-image object-cover aspect-square object-contain h-2/3" src="/api/image?url=%s">
                <div class="flex flex-col text-center">
                    <p>%s</p>
                    <p><em>%s</em></p>
                    <Button class="rounded-md outline border-green-800 text-green-800 hover:border-transparent hover:bg-green-800 hover:text-white active:bg-green-900"
						hx-get="/api/scorematch/"
                        hx-trigger="click"
                        hx-target="#bird-wrapper"
                        hx-swap="outerHTML"
                        hx-vals='{"winner": "left", "leftBirdID": "%s", "rightBirdID": "%s"}'>
                        This one!
                    </Button>
                </div>
			</div>
			<!-- Right Bird Card -->
            <div class="m-32 shadow-lg rounded-sm w-2/3 p-6 flex flex-col align-items-center bg-zinc-300" id="right-bird">
                <img  class="card-image object-cover aspect-square box-content h-2/3" src="/api/image?url=%s">
                <div class="flex flex-col text-center">
                    <p>%s</p>
                    <p><em>%s</em></p>
                    <Button class="rounded-md outline border-green-800 text-green-800 hover:border-transparent hover:bg-green-800 hover:text-white active:bg-green-900"
					hx-get="/api/scorematch/"
                        hx-trigger="click"
                        hx-target="#bird-wrapper"
                        hx-swap="outerHTML"
                        hx-vals='{"winner": "right", "leftBirdID": "%s", "rightBirdID": "%s"}'>
                        This one!
                    </Button>
                </div>
            </div>
        </div>`,
		url.QueryEscape(newLeftBird.ImageUrls[0]),
		newLeftBird.CommonName.String,
		newLeftBird.ScientificName.String,
		newLeftBird.ID.String(),
		newRightBird.ID.String(),
		url.QueryEscape(newRightBird.ImageUrls[0]),
		newRightBird.CommonName.String,
		newRightBird.ScientificName.String,
		newLeftBird.ID.String(),
		newRightBird.ID.String(),
	)

	w.Write([]byte(payload))
}

func (cfg *ApiConfig) handleLoadLeaderboard(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("call to load leaderboard handler")

	listLength, err := strconv.Atoi(r.URL.Query().Get("listLength"))
	if err != nil {
		log.Error().Err(err).Msg("failed to parse listLength")
		return
	}

	if listLength <= 0 || listLength > 1000 {
		err := fmt.Errorf("listLength must be between 1-1000")
		log.Error().Err(err).Msg("invalid listLength")
		listLength = 10
	}

	topBirds, err := cfg.DbQueries.GetTopRatings(r.Context(), int32(listLength))
	if err != nil {
		log.Error().Err(err).Msg("failed to get top ratings")
		return
	}

	w.Header().Add("Content-Type", "text/html; charset=utf-8")

	var builder strings.Builder
	builder.WriteString("<table>\n")

	for i, rating := range topBirds {
		row := fmt.Sprintf(
			`<tr>
				<td>%d.</td>
				<td>%s</td>
				<td>%d</td>
			</tr>
			`,
			i+1,
			rating.CommonName.String,
			rating.Rating.Int32,
		)
		builder.WriteString(row)
	}

	builder.WriteString("</table>\n")

	payload := builder.String()

	w.Write([]byte(payload))
}

func (cfg *ApiConfig) handleCachedImage(w http.ResponseWriter, r *http.Request) {
	imageURL := r.URL.Query().Get("url")
	if imageURL == "" {
		http.Error(w, "missing url parameter", http.StatusBadRequest)
		return
	}

	cacheURL := fmt.Sprintf("http://%s:1337/sc/%s", cfg.CacheHost, imageURL)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := client.Get(cacheURL)
	if err != nil {
		http.Error(w, "failed to fetch image", http.StatusInternalServerError)
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
