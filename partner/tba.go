// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Read-only helpers for retrieving data from The Blue Alliance. Publishing is disabled in lite.

package partner

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Team254/cheesy-arena-lite/model"
	"io"
	"net/http"
	"os"
)

const (
	tbaBaseUrl = "https://www.thebluealliance.com"
	tbaAuthKey = "MAApv9MCuKY9MSFkXLuzTSYBCdosboxDq8Q3ujUE2Mn8PD3Nmv64uczu5Lvy0NQ3"
	AvatarsDir = "static/img/avatars"
)

type TbaClient struct {
	BaseUrl         string
	eventCode       string
	secretId        string
	secret          string
	eventNamesCache map[string]string
}

type TbaTeam struct {
	TeamNumber int    `json:"team_number"`
	Name       string `json:"name"`
	Nickname   string `json:"nickname"`
	City       string `json:"city"`
	StateProv  string `json:"state_prov"`
	Country    string `json:"country"`
	RookieYear int    `json:"rookie_year"`
}

type TbaRobot struct {
	RobotName string `json:"robot_name"`
	Year      int    `json:"year"`
}

type TbaAward struct {
	Name      string `json:"name"`
	EventKey  string `json:"event_key"`
	Year      int    `json:"year"`
	EventName string
}

type TbaEvent struct {
	Name string `json:"name"`
}

type TbaMediaItem struct {
	Details map[string]any `json:"details"`
	Type    string         `json:"type"`
}

func NewTbaClient(eventCode, secretId, secret string) *TbaClient {
	return &TbaClient{
		BaseUrl:         tbaBaseUrl,
		eventCode:       eventCode,
		secretId:        secretId,
		secret:          secret,
		eventNamesCache: make(map[string]string),
	}
}

func (client *TbaClient) GetTeam(teamNumber int) (*TbaTeam, error) {
	path := fmt.Sprintf("/api/v3/team/%s", getTbaTeam(teamNumber))
	var teamData TbaTeam
	if err := client.getJson(path, &teamData); err != nil {
		return nil, err
	}
	return &teamData, nil
}

func (client *TbaClient) GetRobotName(teamNumber int, year int) (string, error) {
	path := fmt.Sprintf("/api/v3/team/%s/robots", getTbaTeam(teamNumber))
	var robots []*TbaRobot
	if err := client.getJson(path, &robots); err != nil {
		return "", err
	}
	for _, robot := range robots {
		if robot.Year == year {
			return robot.RobotName, nil
		}
	}
	return "", nil
}

func (client *TbaClient) GetTeamAwards(teamNumber int) ([]*TbaAward, error) {
	path := fmt.Sprintf("/api/v3/team/%s/awards", getTbaTeam(teamNumber))
	var awards []*TbaAward
	if err := client.getJson(path, &awards); err != nil {
		return nil, err
	}
	for _, award := range awards {
		eventName, err := client.getEventName(award.EventKey)
		if err != nil {
			return nil, err
		}
		award.EventName = eventName
	}
	return awards, nil
}

func (client *TbaClient) DownloadTeamAvatar(teamNumber, year int) error {
	path := fmt.Sprintf("/api/v3/team/%s/media/%d", getTbaTeam(teamNumber), year)
	var mediaItems []*TbaMediaItem
	if err := client.getJson(path, &mediaItems); err != nil {
		return err
	}
	for _, mediaItem := range mediaItems {
		if mediaItem.Type != "avatar" {
			continue
		}
		imageData, ok := mediaItem.Details["base64Image"].(string)
		if !ok || imageData == "" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(imageData)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(AvatarsDir, 0755); err != nil {
			return err
		}
		return os.WriteFile(fmt.Sprintf("%s/%d.png", AvatarsDir, teamNumber), decoded, 0644)
	}
	return nil
}

func (client *TbaClient) PublishTeams(database *model.Database) error {
	return nil
}

func (client *TbaClient) PublishMatches(database *model.Database) error {
	return nil
}

func (client *TbaClient) PublishRankings(database *model.Database) error {
	return nil
}

func (client *TbaClient) PublishAlliances(database *model.Database) error {
	return nil
}

func (client *TbaClient) PublishAwards(database *model.Database) error {
	return nil
}

func (client *TbaClient) DeletePublishedMatches() error {
	return nil
}

func checkTbaPostResponse(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if err != nil {
			return fmt.Errorf("Got status code %d from TBA and failed to read response body: %w", resp.StatusCode, err)
		}
		if closeErr != nil {
			return fmt.Errorf("Got status code %d from TBA: %s; failed to close response body: %w",
				resp.StatusCode, body, closeErr)
		}
		return fmt.Errorf("Got status code %d from TBA: %s", resp.StatusCode, body)
	}
	if err := resp.Body.Close(); err != nil {
		return fmt.Errorf("Failed to close TBA response body: %w", err)
	}
	return nil
}

func (client *TbaClient) getEventName(eventCode string) (string, error) {
	if eventName, ok := client.eventNamesCache[eventCode]; ok {
		return eventName, nil
	}

	path := fmt.Sprintf("/api/v3/event/%s", eventCode)
	var event TbaEvent
	if err := client.getJson(path, &event); err != nil {
		return "", err
	}
	client.eventNamesCache[eventCode] = event.Name
	return event.Name, nil
}

// Converts an integer team number into the "frcXXXX" format TBA expects.
func getTbaTeam(team int) string {
	return fmt.Sprintf("frc%d", team)
}

func (client *TbaClient) getJson(path string, target any) error {
	resp, err := client.getRequest(path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Got status code %d from TBA and failed to read response body: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("Got status code %d from TBA: %s", resp.StatusCode, body)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

// Sends a GET request to the TBA API.
func (client *TbaClient) getRequest(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", client.BaseUrl+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-TBA-Auth-Key", tbaAuthKey)
	return http.DefaultClient.Do(req)
}
