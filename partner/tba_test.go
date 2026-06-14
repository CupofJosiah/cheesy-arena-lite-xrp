// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package partner

import (
	"github.com/Team254/cheesy-arena-lite/model"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strings"
	"testing"
)

func TestPublishingIsDisabled(t *testing.T) {
	database := setupTestDb(t)
	defer database.Close()

	client := NewTbaClient("my_event_code", "my_secret_id", "my_secret")
	assert.Nil(t, client.PublishTeams(database))
	assert.Nil(t, client.PublishMatches(database))
	assert.Nil(t, client.PublishRankings(database))
	assert.Nil(t, client.PublishAlliances(database))
	assert.Nil(t, client.PublishAwards(database))
	assert.Nil(t, client.DeletePublishedMatches())
}

func TestCheckTbaPostResponseClosesBody(t *testing.T) {
	body := &closeTrackingBody{Reader: strings.NewReader("ok")}
	err := checkTbaPostResponse(&http.Response{StatusCode: 200, Body: body})
	assert.Nil(t, err)
	assert.True(t, body.closed)

	body = &closeTrackingBody{Reader: strings.NewReader("oh noes")}
	err = checkTbaPostResponse(&http.Response{StatusCode: 500, Body: body})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Got status code 500 from TBA: oh noes")
	assert.True(t, body.closed)
}

func TestGetTbaTeam(t *testing.T) {
	assert.Equal(t, "frc254", getTbaTeam(254))
	assert.Equal(t, "frc1114", getTbaTeam(1114))
}

func setupTestDb(t *testing.T) *model.Database {
	return model.SetupTestDb(t)
}

type closeTrackingBody struct {
	*strings.Reader
	closed bool
}

func (body *closeTrackingBody) Close() error {
	body.closed = true
	return nil
}
