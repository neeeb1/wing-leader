package auth

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/rs/zerolog/log"
)

func CreateSessionToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		log.Error().Err(err).Msg("Unable to create session token")
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func DecodeSessionToken(cookie string) (string, error) {
	bytes, err := base64.URLEncoding.DecodeString(cookie)
	if err != nil {
		log.Error().Err(err).Msg("Unable to decode session token")
		return "", err
	}
	token := string(bytes)
	return token, nil
}
