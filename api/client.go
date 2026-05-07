package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"resty.dev/v3"
)

const BaseURL = "https://var.elaniin.com/api"

type Client struct {
	resty *resty.Client
}

type Thumbnails struct {
	Medium    string `json:"medium"`
	Thumbnail string `json:"thumbnail"`
}

type Profile struct {
	ID                   int        `json:"id"`
	Name                 string     `json:"name"`
	Code                 string     `json:"code"`
	BirthDate            string     `json:"birth_date"`
	Email                string     `json:"email"`
	Photo                string     `json:"photo"`
	Thumbnails           Thumbnails `json:"thumbnails"`
	Phone                string     `json:"phone"`
	StartedDate          string     `json:"started_date"`
	SlackID              *string    `json:"slack_id"`
	Position             string     `json:"position"`
	DepartmentID         int        `json:"department_id"`
	CompanyID            int        `json:"company_id"`
	IsLead               string     `json:"is_lead"`
	Terms                bool       `json:"terms"`
	New                  bool       `json:"new"`
	StartedDateYears     string     `json:"started_date_years"`
	SharePhone           bool       `json:"share_phone"`
	ShareBirthdate       bool       `json:"share_birthdate"`
	ReceiveNotifications bool       `json:"receive_notifications"`
}

type Project struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TimeEntry struct {
	ID          int     `json:"id"`
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Minutes     int     `json:"minutes"`
	IsBillable  bool    `json:"is_billable"`
	IsEditable  bool    `json:"is_editable"`
	Project     Project `json:"project"`
	Tags        []Tag   `json:"tags"`
}

type NewTimeEntry struct {
	Date        string `json:"date"`
	Description string `json:"description"`
	ProjectID   int    `json:"project_id"`
	TagIDs      []int  `json:"tag_ids,omitempty"`
	Minutes     int    `json:"minutes"`
	IsBillable  bool   `json:"is_billable"`
}

func NewClient(token string) *Client {
	r := resty.New()
	r.SetBaseURL(BaseURL)
	r.SetHeader("Authorization", "Bearer "+token)
	r.SetHeader("Accept", "application/json")
	r.SetTimeout(15 * time.Second)

	return &Client{resty: r}
}

func (c *Client) SetToken(token string) {
	c.resty.SetHeader("Authorization", "Bearer "+token)
}

func decodeList[T any](body []byte) ([]T, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	// Direct array
	if trimmed[0] == '[' {
		var direct []T
		if err := json.Unmarshal(body, &direct); err != nil {
			return nil, err
		}
		return direct, nil
	}

	// Wrapped object: { "data": [...] }
	var wrapped struct {
		Data []T `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, err
	}
	return wrapped.Data, nil
}

func decodeObject[T any](body []byte) (T, error) {
	var zero T
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return zero, fmt.Errorf("empty response body")
	}

	// Try wrapped object first: { "data": {...} }
	var wrapped struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil {
		// Distinguish from empty struct by checking if body actually had a "data" key
		var check map[string]json.RawMessage
		if err := json.Unmarshal(body, &check); err == nil {
			if _, ok := check["data"]; ok {
				return wrapped.Data, nil
			}
		}
	}

	// Try direct object
	var direct T
	if err := json.Unmarshal(body, &direct); err != nil {
		return zero, err
	}
	return direct, nil
}

func (c *Client) GetProfile() (*Profile, error) {
	resp, err := c.resty.NewRequest().Get("/profile/me")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("invalid token or request failed: %s", resp.Status())
	}

	profile, err := decodeObject[Profile](resp.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to decode profile: %w", err)
	}
	return &profile, nil
}

func (c *Client) GetProjects() ([]Project, error) {
	resp, err := c.resty.NewRequest().Get("/projects")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch projects: %s", resp.Status())
	}

	body := resp.Bytes()
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	var result []Project

	// Wrapped object: { "data": { "CLIENT": [...] } }
	var wrapped struct {
		Data map[string][]Project `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && len(wrapped.Data) > 0 {
		for _, projects := range wrapped.Data {
			result = append(result, projects...)
		}
		return result, nil
	}

	// Direct map: { "CLIENT": [...] }
	var direct map[string][]Project
	if err := json.Unmarshal(body, &direct); err == nil {
		for _, projects := range direct {
			result = append(result, projects...)
		}
		return result, nil
	}

	return nil, fmt.Errorf("failed to decode projects response")
}

func (c *Client) GetTags() ([]Tag, error) {
	resp, err := c.resty.NewRequest().Get("/tags?limit=1000")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch tags: %s", resp.Status())
	}
	return decodeList[Tag](resp.Bytes())
}

func (c *Client) GetTimeEntries(startDate, endDate string) ([]TimeEntry, error) {
	resp, err := c.resty.NewRequest().
		SetQueryParams(map[string]string{
			"start_date": startDate,
			"end_date":   endDate,
		}).
		Get("/time-entries")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch time entries: %s", resp.Status())
	}

	body := resp.Bytes()
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	var result []TimeEntry

	// Wrapped object: { "data": { "2026-05-01": [...] } }
	var wrapped struct {
		Data map[string][]TimeEntry `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && len(wrapped.Data) > 0 {
		for _, entries := range wrapped.Data {
			result = append(result, entries...)
		}
		return result, nil
	}

	// Direct map: { "2026-05-01": [...] }
	var direct map[string][]TimeEntry
	if err := json.Unmarshal(body, &direct); err == nil {
		for _, entries := range direct {
			result = append(result, entries...)
		}
		return result, nil
	}

	return nil, fmt.Errorf("failed to decode time entries response")
}

func (c *Client) CreateTimeEntry(entry NewTimeEntry) (*TimeEntry, error) {
	resp, err := c.resty.NewRequest().
		SetBody(entry).
		Post("/time-entries")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusCreated {
		return nil, fmt.Errorf("failed to create time entry: %s", resp.Status())
	}

	created, err := decodeObject[TimeEntry](resp.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to decode created entry: %w", err)
	}
	return &created, nil
}

func (c *Client) UpdateTimeEntry(id int, entry NewTimeEntry) (*TimeEntry, error) {
	resp, err := c.resty.NewRequest().
		SetBody(entry).
		Put(fmt.Sprintf("/time-entries/%d", id))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to update time entry: %s", resp.Status())
	}
	updated, err := decodeObject[TimeEntry](resp.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to decode updated entry: %w", err)
	}
	return &updated, nil
}

func (c *Client) DeleteTimeEntry(id int) error {
	resp, err := c.resty.NewRequest().
		Delete(fmt.Sprintf("/time-entries/%d", id))
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent && resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("failed to delete time entry: %s", resp.Status())
	}
	return nil
}
