package api

import (
	"database/sql"
	"fmt"

	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/neeeb1/rate_birds/internal/auth"
	"github.com/neeeb1/rate_birds/internal/database"
	"github.com/rs/zerolog/log"
)

const (
	sessionExpiration = 600 * time.Second
)

func (cfg *ApiConfig) handleLoadBirds(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("call to load bird handler")

	rng_bird, err := cfg.DbQueries.GetRandomBirdWithImage(r.Context(), 2)
	if err != nil {
		log.Error().Err(err).Msg("failed to get random birds with images")
		RespondWithError(w, 503, "Failed to get random birds with images")
		return
	}

	newLeftBird, newRightBird := rng_bird[0], rng_bird[1]

	sessionToken, err := auth.CreateSessionToken(32)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate new match session token")
		RespondWithError(w, 500, "Failed to get random birds with images")
		return
	}

	sessionParams := database.CreateMatchSessionParams{
		LeftbirdID:   newLeftBird.ID,
		RightbirdID:  newRightBird.ID,
		SessionToken: sessionToken,
		ExpiresAt:    time.Now().Add(sessionExpiration),
		UserIp:       sql.NullString{String: r.RemoteAddr, Valid: true},
		UserAgent:    sql.NullString{String: r.UserAgent(), Valid: true},
	}

	if _, err := cfg.DbQueries.CreateMatchSession(r.Context(), sessionParams); err != nil {
		log.Error().Err(err).Msg("Failed to create match session db entry")
		RespondWithError(w, 503, "Failed to create match session db entry")
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
		`<div id="bird-wrapper" class="w-full max-w-6xl p-6 flex flex-row sm:flex-col gap-12 items-stretch justify-center">
		<!-- Left Bird Card -->
			<div class="flex-1 max-w-md shadow-lg rounded-lg p-6 flex flex-col bg-zinc-300" id="left-bird">
				<div id="left-bird-content" class="flex-1 flex flex-col">
					<div class="w-full aspect-square mb-4 overflow-hidden rounded-md bg-gray-100 flex items-center justify-center"
						id="left-bird">
						<img class="w-full h-full object-cover loading-shimmer" width="250" height="250"
							src="/api/image?url=%s">
					</div>
					<div class="flex flex-col text-center mb-4">
						<p>%s</p>
						<p><em>%s</em></p>
					</div>	
					<button
						class="rounded-md outline border-green-800 text-green-800 hover:border-transparent hover:bg-green-800 hover:text-white active:bg-green-900"
						hx-get="/api/scorematch/" hx-trigger="click" hx-target="#bird-wrapper" hx-swap="outerHTML"
						hx-vals='{"winner": "left", "leftBirdID": "%s", "rightBirdID": "%s"}'>
						This one!
					</button>
				</div>
			</div>
			
			<!-- Right Bird Card -->
			<div class="flex-1 max-w-md shadow-lg rounded-lg p-6 flex flex-col bg-zinc-300" id="right-bird">
				<div id="right-bird-content" class="flex-1 flex flex-col">
					<div class="w-full aspect-square mb-4 overflow-hidden rounded-md bg-gray-100 flex items-center justify-center"
						id="right-bird">
						<img class="w-full h-full object-cover loading-shimmer" width="250" height="250"
							src="/api/image?url=%s">
					</div>
					<div class="flex flex-col text-center mb-4">
						<p>%s</p>
						<p><em>%s</em></p>
					</div>
					<button
						class="rounded-md outline border-green-800 text-green-800 hover:border-transparent hover:bg-green-800 hover:text-white active:bg-green-900"
						hx-get="/api/scorematch/" hx-trigger="click" hx-target="#bird-wrapper" hx-swap="outerHTML"
						hx-vals='{"winner": "right", "leftBirdID": "%s", "rightBirdID": "%s"}'>
						This one!
					</button>
				</div>
			</div>`,
		newLeftBird.ImageUrls[0],
		newLeftBird.CommonName.String,
		newLeftBird.ScientificName.String,
		newLeftBird.ID.String(),
		newRightBird.ID.String(),
		newRightBird.ImageUrls[0],
		newRightBird.CommonName.String,
		newRightBird.ScientificName.String,
		newLeftBird.ID.String(),
		newRightBird.ID.String(),
	)

	w.Write([]byte(payload))
}

func (cfg *ApiConfig) handleLoadLeaderboard(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("call to load leaderboard handler")

	listLength, err := validateListLength(r.URL.Query().Get("listLength"))
	if err != nil {
		log.Error().Err(err).Msg("invalid listLength")
		RespondWithError(w, 500, "invalid listLength: must be between 1-1000")
		return
	}

	topBirds, err := cfg.DbQueries.GetTopRatings(r.Context(), int32(listLength))
	if err != nil {
		log.Error().Err(err).Msg("failed to get top ratings")
		RespondWithError(w, 503, "Failed to get top ratings")
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

func validateListLength(input string) (int, error) {
	listLength, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("invalid format: %w", err)
	}

	if listLength <= 0 || listLength > 1000 {
		return 0, fmt.Errorf("must be between 1-1000")
	}

	return listLength, nil
}
