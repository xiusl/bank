package token

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xiusl/bank/util"
)

func TestPasetoMaker(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	username := util.RandomOwner()
	duration := time.Minute
	issueAt := time.Now()
	expiredAt := issueAt.Add(duration)

	tokenString, err := maker.CreateToken(username, duration)
	require.NoError(t, err)
	require.NotEmpty(t, tokenString)

	token, err := maker.VerifyToken(tokenString)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	require.NotZero(t, token.ID)
	require.Equal(t, username, token.Username)
	require.WithinDuration(t, issueAt, token.IssuedAt, time.Second)
	require.WithinDuration(t, expiredAt, token.ExpiredAt, time.Second)
}

func TestExpiredPasetoToken(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	username := util.RandomOwner()
	duration := -time.Minute

	tokenString, err := maker.CreateToken(username, duration)
	require.NoError(t, err)
	require.NotEmpty(t, tokenString)

	token, err := maker.VerifyToken(tokenString)
	require.Error(t, err)
	require.EqualError(t, err, ErrExpiredToken.Error())
	require.Empty(t, token)
}
