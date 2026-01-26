package api

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/neeeb1/rate_birds/internal/database"
	"github.com/rs/zerolog/log"
)

const (
	// k-factor, maximum rating change
	k = 32
	d = 400
)

func (cfg *ApiConfig) handleScoreMatch(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("call to score match handler")

	cookie, err := r.Cookie("sessionToken")
	if err != nil {
		log.Error().Err(err).Msg("Failed to read sessionToken cookie")
		return
	}

	session, err := cfg.DbQueries.GetMatchSessionByToken(r.Context(), cookie.Value)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get session by token")
		return
	}

	if session.Voted.Bool {
		log.Error().Err(err).Msg("Session has already been voted on")
		return
	}

	if time.Now().After(session.ExpiresAt) {
		log.Error().Err(err).Msg("Session token has already expired")
		return
	}

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
		err := cfg.ScoreMatch(leftBird, rightBird)
		if err != nil {
			log.Error().Err(err).Msg("failed to score match")
			break
		}

		voteParams := database.VoteMatchParams{
			ID:           session.ID,
			WinnerbirdID: uuid.NullUUID{UUID: leftBird.ID, Valid: true},
		}
		cfg.DbQueries.VoteMatch(r.Context(), voteParams)

	case "right":
		//log.Info().Msgf("Winner: %s, Loser: %s\n", rightBird.CommonName.String, leftBird.CommonName.String)
		err := cfg.ScoreMatch(rightBird, leftBird)
		if err != nil {
			log.Error().Err(err).Msg("failed to score match")
			break
		}

		voteParams := database.VoteMatchParams{
			ID:           session.ID,
			WinnerbirdID: uuid.NullUUID{UUID: rightBird.ID, Valid: true},
		}
		cfg.DbQueries.VoteMatch(r.Context(), voteParams)
	}

	cfg.handleLoadBirds(w, r)
}

func (cfg *ApiConfig) ScoreMatch(winner, loser database.Bird) error {
	ctx := context.Background()

	tx, err := cfg.Db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("unable to start sql transaction: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	qtx := cfg.DbQueries.WithTx(tx)

	winnerDb, err := qtx.GetRatingByBirdID(ctx, winner.ID)
	if err != nil {
		return err
	}
	winnerRating := winnerDb.Rating.Int32

	loserDb, err := qtx.GetRatingByBirdID(ctx, loser.ID)
	if err != nil {
		return err
	}
	loserRating := loserDb.Rating.Int32

	winnerExpected, loserExpected := calculateExpected(int(winnerRating), int(loserRating))
	winnerDelta := calculateDelta(winnerExpected, 1.0)
	loserDelta := calculateDelta(loserExpected, 0.0)

	winnerNewRating := winnerRating + winnerDelta
	loserNewRating := loserRating + loserDelta

	winParams := database.UpdateRatingByBirdIDParams{
		Rating: sql.NullInt32{Int32: winnerNewRating, Valid: true},
		BirdID: winnerDb.BirdID,
	}
	_, err = qtx.UpdateRatingByBirdID(ctx, winParams)
	if err != nil {
		return err
	}

	loseParams := database.UpdateRatingByBirdIDParams{
		Rating: sql.NullInt32{Int32: loserNewRating, Valid: true},
		BirdID: loserDb.BirdID,
	}
	_, err = qtx.UpdateRatingByBirdID(ctx, loseParams)
	if err != nil {
		log.Error().Err(err).Msg("failed to update loser rating")
		return err
	}

	matchParms := database.RecordMatchParams{
		LoserbirdID:  loser.ID,
		WinnerbirdID: winner.ID,
	}
	_, err = qtx.RecordMatch(ctx, matchParms)
	if err != nil {
		log.Error().Err(err).Msg("failed to record match")
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("unable to commit sql transaction: %w", err)
	}

	//log.Info().Msgf("Updated ratings for %s and %s\n", winner.CommonName.String, loser.CommonName.String)
	log.Info().Msgf("(W) '%s': %d || (L) '%s': %d\n", winner.CommonName.String, winnerNewRating, loser.CommonName.String, loserNewRating)
	return nil
}

func calculateExpected(ratingA, ratingB int) (expectedA, expectedB float64) {
	qA := math.Pow10(ratingA / d)
	qB := math.Pow10(ratingB / d)

	expectedA = qA / (qA + qB)
	expectedB = qB / (qA + qB)
	return
}

func calculateDelta(expected, actual float64) int32 {
	return int32(k * (actual - expected))
}
