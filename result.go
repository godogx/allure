package allure

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/afero"
)

// Container lists all results.
type Container struct {
	UUID     string      `json:"uuid,omitempty"`
	Name     string      `json:"name"`
	Children []string    `json:"children"`
	Start    TimestampMs `json:"start,omitempty"`
	Stop     TimestampMs `json:"stop,omitempty"`
}

// Result is the top level report object for a test.
//
// 18 known properties: "start", "descriptionHtml", "parameters", "name", "historyId",
// "statusDetails", "status", "links", "fullName", "uuid", "description", "testCaseId",
// "stage", "labels", "stop", "steps", "rerunOf", "attachments".
type Result struct {
	UUID          string         `json:"uuid,omitempty"`
	HistoryID     string         `json:"historyId,omitempty"`
	Name          string         `json:"name,omitempty"`
	Description   string         `json:"description,omitempty"`
	Status        Status         `json:"status,omitempty"`
	StatusDetails *StatusDetails `json:"statusDetails,omitempty"`
	Stage         string         `json:"stage,omitempty"` // "finished"
	Steps         []Step         `json:"steps,omitempty"`
	Attachments   []Attachment   `json:"attachments,omitempty"`
	Parameters    []Parameter    `json:"parameters,omitempty"`
	Start         TimestampMs    `json:"start,omitempty"`
	Stop          TimestampMs    `json:"stop,omitempty"`
	Children      []string       `json:"children,omitempty"`
	FullName      string         `json:"fullName,omitempty"`
	Labels        []Label        `json:"labels,omitempty"`
	Links         []Link         `json:"links,omitempty"`
}

// Available statuses.
const (
	Broken  = Status("broken")
	Passed  = Status("passed")
	Failed  = Status("failed")
	Skipped = Status("skipped")
	Unknown = Status("unknown")
)

// TimestampMs is a timestamp in milliseconds.
type TimestampMs int64

// LinkType is a type of link.
type LinkType string

// Types of links.
const (
	Issue  LinkType = "issue"
	TMS    LinkType = "tms"
	Custom LinkType = "custom"
)

// Link references additional resources.
type Link struct {
	Name string   `json:"name,omitempty"`
	Type LinkType `json:"type,omitempty"`
	URL  string   `json:"url,omitempty"`
}

// Status describes test result.
type Status string

// StatusDetails provides additional information on status.
type StatusDetails struct {
	Known   bool   `json:"known,omitempty"`
	Muted   bool   `json:"muted,omitempty"`
	Flaky   bool   `json:"flaky,omitempty"`
	Message string `json:"message,omitempty"`
	Trace   string `json:"trace,omitempty"`
}

// Step is a part of scenario result.
type Step struct {
	Name          string         `json:"name,omitempty"`
	Status        Status         `json:"status,omitempty"`
	StatusDetails *StatusDetails `json:"statusDetails,omitempty"`
	Stage         string         `json:"stage"`
	ChildrenSteps []Step         `json:"steps"`
	Attachments   []Attachment   `json:"attachments"`
	Parameters    []Parameter    `json:"parameters"`
	Start         TimestampMs    `json:"start"`
	Stop          TimestampMs    `json:"stop"`
}

// Attachment can be attached.
type Attachment struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Type   string `json:"type"`
}

// NewAttachment creates and stores attachment.
func NewAttachment(fs afero.Fs, name string, mimeType string, resultsPath string, content []byte) (*Attachment, error) {
	var ext string

	switch mimeType {
	case "application/json":
		ext = ".json"
	case "image/png":
		ext = ".png"
	case "image/jpeg":
		ext = ".jpg"
	case "image/gif":
		ext = ".gif"
	case csvMime:
		ext = ".csv"
	case "application/xml":
		ext = ".xml"
	default:
		ext = ".txt"
	}

	a := Attachment{
		Name:   name,
		Type:   mimeType,
		Source: fmt.Sprintf("%s-attachment%s", uuid.New().String(), ext),
	}

	if err := afero.WriteFile(fs, filepath.Join(resultsPath, a.Source), content, 0600); err != nil {
		return nil, err
	}

	return &a, nil
}

// Parameter is a named value.
type Parameter struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// Label is a named value.
type Label struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}
