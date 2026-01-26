package api

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/neeeb1/rate_birds/internal/auth"
	"github.com/neeeb1/rate_birds/internal/database"
	"github.com/rs/zerolog/log"
)

func (cfg *ApiConfig) handleLoadBirds(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("call to load bird handler")

	rng_bird, err := cfg.DbQueries.GetRandomBirdWithImage(r.Context(), 2)
	if err != nil {
		log.Error().Err(err).Msg("failed to get random birds with images")
	}

	newLeftBird, newRightBird := rng_bird[0], rng_bird[1]

	sessionToken, err := auth.CreateSessionToken(32)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate new match session token")
	}

	sessionParams := database.CreateMatchSessionParams{
		LeftbirdID:   newLeftBird.ID,
		RightbirdID:  newRightBird.ID,
		SessionToken: sessionToken,
		ExpiresAt:    time.Now().Add(600 * time.Second),
		UserIp:       sql.NullString{String: r.RemoteAddr, Valid: true},
		UserAgent:    sql.NullString{String: r.UserAgent(), Valid: true},
	}

	if _, err := cfg.DbQueries.CreateMatchSession(r.Context(), sessionParams); err != nil {
		log.Error().Err(err).Msg("Failed to create match session db entry")
		return
	}

	sessionCookie := &http.Cookie{
		Name:  "sessionToken",
		Value: sessionToken,
		Path:  "/",
		//Max age 10min
		MaxAge:   600,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(w, sessionCookie)

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
	builder.WriteString("<table>\n<tr><th>Rank</th><th>Common Name</th><th>ELO</th></tr>\n")

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

	cacheURL := fmt.Sprintf("http://%s:1337/500x500,sc/%s", cfg.CacheHost, imageURL)

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

/* func (cfg *ApiConfig) handleLoadMatches(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("call to load matches handler")

	matches, err := cfg.DbQueries.GetAllMatches(r.Context(), 100)
	if err != nil {
		log.Error().Err(err).Msg("failed to get matches")
		return
	}

	w.Header().Add("Content-Type", "text/html; charset=utf-8")

	var builder strings.Builder
	builder.WriteString("<table>\n<tr><th>Match ID</th><th>Winner</th><th>Loser</th><th>Date</th></tr>\n")

	for _, match := range matches {
		row := fmt.Sprintf(
			`<tr>
				<td>%s</td>
				<td>%s</td>
				<td>%s</td>
				<td>%s</td>
			</tr>
			`,
			match.ID.String(),
			match.LoserbirdID.String(),
			match.WinnerbirdID.String(),
			match.CreatedAt.Format("2006-01-02 15:04:05"),
		)
		builder.WriteString(row)
	}

	builder.WriteString("</table>\n")
	payload := builder.String()

	w.Write([]byte(payload))
} */
