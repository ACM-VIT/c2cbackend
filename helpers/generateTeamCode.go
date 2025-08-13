package helpers

import (
	"math/rand"
	"time"
)

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateTeamCode() string {
	seed := rand.NewSource(time.Now().UnixNano())
	r := rand.New(seed)
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}
