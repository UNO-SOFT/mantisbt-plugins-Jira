// Copyright 2022, 2025 Tamás Gulácsi. All rights reserved.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/UNO-SOFT/zlog/v2/loghttp"
	"github.com/google/renameio/v2"
	"github.com/klauspost/compress/gzhttp"
	"github.com/rogpeppe/retry"
)

// https://docs.atlassian.com/software/jira/docs/api/REST/9.12.2/#api/2/issue-addComment
// https://docs.atlassian.com/software/jira/docs/api/REST/9.12.2/#api/2/issue-doTransition
// https://partnerapi-uat.aegon.hu/partner/v1/ticket/update/openapi.json
type Jira struct {
	URL        *url.URL
	tokens     map[string]*Token
	HTTPClient *http.Client `json:"-"`
	token      *Token
	// socket     string
	tokensFile string
}

type JIRAIssueType struct {
	Self        string `json:"self,omitempty"`
	ID          string `json:"id,omitempty"`
	Description string `json:"description,omitempty"`
	IconURL     string `json:"iconUrl,omitempty"`
	Name        string `json:"name,omitempty"`
	Subtask     bool   `json:"subtask,omitempty"`
	AvatarID    int    `json:"avatarId,omitempty"`
}

// https://mholt.github.io/json-to-go/
type JIRAIssue struct {
	Expand string          `json:"expand,omitempty"`
	ID     string          `json:"id,omitempty"`
	Self   string          `json:"self,omitempty"`
	Key    string          `json:"key,omitempty"`
	Fields jiraIssueFields `json:"fields,omitempty"`
}

//betteralign:skip
type JiraFields struct {
	Timetracking struct {
	} `json:"timetracking,omitempty"`
	Aggregatetimeoriginalestimate interface{} `json:"aggregatetimeoriginalestimate,omitempty"`
	Resolution                    interface{} `json:"resolution,omitempty"`
	Environment                   interface{} `json:"environment,omitempty"`
	Duedate                       interface{} `json:"duedate,omitempty"`
	Timeestimate                  interface{} `json:"timeestimate,omitempty"`
	Aggregatetimeestimate         interface{} `json:"aggregatetimeestimate,omitempty"`
	Timespent                     interface{} `json:"timespent,omitempty"`
	Aggregatetimespent            interface{} `json:"aggregatetimespent,omitempty"`
	Resolutiondate                interface{} `json:"resolutiondate,omitempty"`
	Timeoriginalestimate          interface{} `json:"timeoriginalestimate,omitempty"`
	MantisID                      string      `json:"customfield_15902"` // Mantis,omitemptyID
	Customfield11100              struct {
		ID    string `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
		Links struct {
			Self string `json:"self,omitempty"`
		} `json:"_links,omitempty"`
		CompletedCycles []interface{} `json:"completedCycles,omitempty"`
		OngoingCycle    struct {
			GoalDuration struct {
				Friendly string `json:"friendly,omitempty"`
				Millis   int    `json:"millis,omitempty"`
			} `json:"goalDuration,omitempty"`
			ElapsedTime struct {
				Friendly string `json:"friendly,omitempty"`
				Millis   int    `json:"millis,omitempty"`
			} `json:"elapsedTime,omitempty"`
			RemainingTime struct {
				Friendly string `json:"friendly,omitempty"`
				Millis   int    `json:"millis,omitempty"`
			} `json:"remainingTime,omitempty"`
			StartTime struct {
				Iso8601     string `json:"iso8601,omitempty"`
				Jira        string `json:"jira,omitempty"`
				Friendly    string `json:"friendly,omitempty"`
				EpochMillis int64  `json:"epochMillis,omitempty"`
			} `json:"startTime,omitempty"`
			BreachTime struct {
				Iso8601     string `json:"iso8601,omitempty"`
				Jira        string `json:"jira,omitempty"`
				Friendly    string `json:"friendly,omitempty"`
				EpochMillis int64  `json:"epochMillis,omitempty"`
			} `json:"breachTime,omitempty"`
			Breached            bool `json:"breached,omitempty"`
			Paused              bool `json:"paused,omitempty"`
			WithinCalendarHours bool `json:"withinCalendarHours,omitempty"`
		} `json:"ongoingCycle,omitempty"`
	} `json:"customfield_11100,omitempty"`
	Status           JIRAStatus `json:"status,omitempty"`
	Customfield14326 JIRAUser   `json:"customfield_14326,omitempty"`
	Assignee         JIRAUser   `json:"assignee,omitempty"`
	Reporter         JIRAUser   `json:"reporter,omitempty"`
	Creator          JIRAUser   `json:"creator,omitempty"`
	Project          struct {
		Self           string `json:"self,omitempty"`
		ID             string `json:"id,omitempty"`
		Key            string `json:"key,omitempty"`
		Name           string `json:"name,omitempty"`
		ProjectTypeKey string `json:"projectTypeKey,omitempty"`
	} `json:"project,omitempty"`
	Security struct {
		Self        string `json:"self,omitempty"`
		ID          string `json:"id,omitempty"`
		Description string `json:"description,omitempty"`
		Name        string `json:"name,omitempty"`
	} `json:"security,omitempty"`
	Priority         JIRAPriority `json:"priority,omitempty"`
	Summary          string       `json:"summary,omitempty"`
	Description      string       `json:"description,omitempty"`
	LastViewed       string       `json:"lastViewed,omitempty"`
	Updated          string       `json:"updated,omitempty"`
	Created          string       `json:"created,omitempty"`
	Customfield10009 struct {
		Links struct {
			JiraRest string `json:"jiraRest,omitempty"`
			Web      string `json:"web,omitempty"`
			Self     string `json:"self,omitempty"`
		} `json:"_links,omitempty"`
		RequestType struct {
			ID    string `json:"id,omitempty"`
			Links struct {
				Self string `json:"self,omitempty"`
			} `json:"_links,omitempty"`
			Name          string   `json:"name,omitempty"`
			Description   string   `json:"description,omitempty"`
			HelpText      string   `json:"helpText,omitempty"`
			ServiceDeskID string   `json:"serviceDeskId,omitempty"`
			GroupIds      []string `json:"groupIds,omitempty"`
		} `json:"requestType,omitempty"`
		CurrentStatus struct {
			Status     string `json:"status,omitempty"`
			StatusDate struct {
				Iso8601     string `json:"iso8601,omitempty"`
				Jira        string `json:"jira,omitempty"`
				Friendly    string `json:"friendly,omitempty"`
				EpochMillis int64  `json:"epochMillis,omitempty"`
			} `json:"statusDate,omitempty"`
		} `json:"currentStatus,omitempty"`
	} `json:"customfield_10009,omitempty"`
	Customfield14342 struct {
		ID    string `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
		Links struct {
			Self string `json:"self,omitempty"`
		} `json:"_links,omitempty"`
		CompletedCycles []interface{} `json:"completedCycles,omitempty"`
	} `json:"customfield_14342,omitempty"`
	Customfield14344 struct {
		ID    string `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
		Links struct {
			Self string `json:"self,omitempty"`
		} `json:"_links,omitempty"`
		CompletedCycles []interface{} `json:"completedCycles,omitempty"`
	} `json:"customfield_14344,omitempty"`
	Customfield11101 struct {
		ID    string `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
		Links struct {
			Self string `json:"self,omitempty"`
		} `json:"_links,omitempty"`
		CompletedCycles []struct {
			GoalDuration struct {
				Friendly string `json:"friendly,omitempty"`
				Millis   int    `json:"millis,omitempty"`
			} `json:"goalDuration,omitempty"`
			ElapsedTime struct {
				Friendly string `json:"friendly,omitempty"`
				Millis   int    `json:"millis,omitempty"`
			} `json:"elapsedTime,omitempty"`
			RemainingTime struct {
				Friendly string `json:"friendly,omitempty"`
				Millis   int    `json:"millis,omitempty"`
			} `json:"remainingTime,omitempty"`
			StartTime struct {
				Iso8601     string `json:"iso8601,omitempty"`
				Jira        string `json:"jira,omitempty"`
				Friendly    string `json:"friendly,omitempty"`
				EpochMillis int64  `json:"epochMillis,omitempty"`
			} `json:"startTime,omitempty"`
			StopTime struct {
				Iso8601     string `json:"iso8601,omitempty"`
				Jira        string `json:"jira,omitempty"`
				Friendly    string `json:"friendly,omitempty"`
				EpochMillis int64  `json:"epochMillis,omitempty"`
			} `json:"stopTime,omitempty"`
			Breached bool `json:"breached,omitempty"`
		} `json:"completedCycles,omitempty"`
	} `json:"customfield_11101,omitempty"`
	Customfield14343 struct {
		ID    string `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
		Links struct {
			Self string `json:"self,omitempty"`
		} `json:"_links,omitempty"`
		CompletedCycles []interface{} `json:"completedCycles,omitempty"`
	} `json:"customfield_14343,omitempty"`
	Customfield15113 struct {
		Self     string `json:"self,omitempty"`
		Value    string `json:"value,omitempty"`
		ID       string `json:"id,omitempty"`
		Disabled bool   `json:"disabled,omitempty"`
	} `json:"customfield_15113,omitempty"`
	Customfield14451 struct {
		Self     string `json:"self,omitempty"`
		Value    string `json:"value,omitempty"`
		ID       string `json:"id,omitempty"`
		Disabled bool   `json:"disabled,omitempty"`
	} `json:"customfield_14451,omitempty"`
	Customfield15109 struct {
		Self     string `json:"self,omitempty"`
		Value    string `json:"value,omitempty"`
		ID       string `json:"id,omitempty"`
		Disabled bool   `json:"disabled,omitempty"`
	} `json:"customfield_15109,omitempty"`
	Customfield14408 struct {
		Self     string `json:"self,omitempty"`
		Value    string `json:"value,omitempty"`
		ID       string `json:"id,omitempty"`
		Disabled bool   `json:"disabled,omitempty"`
	} `json:"customfield_14408,omitempty"`
	Customfield14321 struct {
		Self     string `json:"self,omitempty"`
		Value    string `json:"value,omitempty"`
		ID       string `json:"id,omitempty"`
		Disabled bool   `json:"disabled,omitempty"`
	} `json:"customfield_14321,omitempty"`
	Customfield15143 struct {
		Self     string `json:"self,omitempty"`
		Value    string `json:"value,omitempty"`
		ID       string `json:"id,omitempty"`
		Disabled bool   `json:"disabled,omitempty"`
	} `json:"customfield_15143,omitempty"`
	Customfield14325 struct {
		Self     string `json:"self,omitempty"`
		Value    string `json:"value,omitempty"`
		ID       string `json:"id,omitempty"`
		Disabled bool   `json:"disabled,omitempty"`
	} `json:"customfield_14325,omitempty"`
	Customfield15104 struct {
		Self     string `json:"self,omitempty"`
		Value    string `json:"value,omitempty"`
		ID       string `json:"id,omitempty"`
		Disabled bool   `json:"disabled,omitempty"`
	} `json:"customfield_15104,omitempty"`
	Customfield15114 struct {
		Self     string `json:"self,omitempty"`
		Value    string `json:"value,omitempty"`
		ID       string `json:"id,omitempty"`
		Disabled bool   `json:"disabled,omitempty"`
	} `json:"customfield_15114,omitempty"`
	Customfield14339 struct {
		Self     string `json:"self,omitempty"`
		Value    string `json:"value,omitempty"`
		ID       string `json:"id,omitempty"`
		Disabled bool   `json:"disabled,omitempty"`
	} `json:"customfield_14339,omitempty"`
	Customfield14423 struct {
		Self     string `json:"self,omitempty"`
		Value    string `json:"value,omitempty"`
		ID       string `json:"id,omitempty"`
		Disabled bool   `json:"disabled,omitempty"`
	} `json:"customfield_14423,omitempty"`
	Customfield14404 struct {
		Self     string `json:"self,omitempty"`
		Value    string `json:"value,omitempty"`
		ID       string `json:"id,omitempty"`
		Disabled bool   `json:"disabled,omitempty"`
	} `json:"customfield_14404,omitempty"`
	Customfield14449 struct {
		Self     string `json:"self,omitempty"`
		Value    string `json:"value,omitempty"`
		ID       string `json:"id,omitempty"`
		Disabled bool   `json:"disabled,omitempty"`
	} `json:"customfield_14449,omitempty"`
	Customfield14425 struct {
		Self     string `json:"self,omitempty"`
		Value    string `json:"value,omitempty"`
		ID       string `json:"id,omitempty"`
		Disabled bool   `json:"disabled,omitempty"`
	} `json:"customfield_14425,omitempty"`
	Worklog struct {
		Worklogs   []interface{} `json:"worklogs,omitempty"`
		StartAt    int           `json:"startAt,omitempty"`
		MaxResults int           `json:"maxResults,omitempty"`
		Total      int           `json:"total,omitempty"`
	} `json:"worklog,omitempty"`
	Versions         []interface{} `json:"versions,omitempty"`
	Components       []interface{} `json:"components,omitempty"`
	Labels           []interface{} `json:"labels,omitempty"`
	Customfield10008 []JIRAUser    `json:"customfield_10008,omitempty"`
	Subtasks         []struct {
		ID     string `json:"id,omitempty"`
		Key    string `json:"key,omitempty"`
		Self   string `json:"self,omitempty"`
		Fields struct {
			Summary   string        `json:"summary,omitempty"`
			Status    JIRAStatus    `json:"status,omitempty"`
			Priority  JIRAPriority  `json:"priority,omitempty"`
			IssueType JIRAIssueType `json:"issuetype,omitempty"`
		} `json:"fields,omitempty"`
	} `json:"subtasks,omitempty"`
	Attachment       []interface{} `json:"attachment,omitempty"`
	Customfield15216 []struct {
		Active bool `json:"active,omitempty"`
	} `json:"customfield_15216,omitempty"`
	Customfield14336 []struct {
		Name string `json:"name,omitempty"`
		Self string `json:"self,omitempty"`
	} `json:"customfield_14336,omitempty"`
	Issuelinks       []interface{} `json:"issuelinks,omitempty"`
	FixVersions      []interface{} `json:"fixVersions,omitempty"`
	Customfield15217 []struct {
		Active bool `json:"active,omitempty"`
	} `json:"customfield_15217,omitempty"`
	IssueType JIRAIssueType `json:"issuetype,omitempty"`
	Watches   struct {
		Self       string `json:"self,omitempty"`
		WatchCount int    `json:"watchCount,omitempty"`
		IsWatching bool   `json:"isWatching,omitempty"`
	} `json:"watches,omitempty"`
	Votes struct {
		Self     string `json:"self,omitempty"`
		Votes    int    `json:"votes,omitempty"`
		HasVoted bool   `json:"hasVoted,omitempty"`
	} `json:"votes,omitempty"`
	Comment struct {
		Comments   []interface{} `json:"comments,omitempty"`
		MaxResults int           `json:"maxResults,omitempty"`
		Total      int           `json:"total,omitempty"`
		StartAt    int           `json:"startAt,omitempty"`
	} `json:"comment,omitempty"`
	Progress struct {
		Progress int `json:"progress,omitempty"`
		Total    int `json:"total,omitempty"`
	} `json:"progress,omitempty"`
	Aggregateprogress struct {
		Progress int `json:"progress,omitempty"`
		Total    int `json:"total,omitempty"`
	} `json:"aggregateprogress,omitempty"`
	Workratio int `json:"workratio,omitempty"`
}

type JIRAIssueStatus struct {
	Self           string `json:"self,omitempty"`
	Description    string `json:"description,omitempty"`
	IconURL        string `json:"iconUrl,omitempty"`
	Name           string `json:"name,omitempty"`
	ID             string `json:"id,omitempty"`
	StatusCategory struct {
		Self      string `json:"self,omitempty"`
		Key       string `json:"key,omitempty"`
		ColorName string `json:"colorName,omitempty"`
		Name      string `json:"name,omitempty"`
		ID        int    `json:"id,omitempty"`
	} `json:"statusCategory,omitempty"`
}

/*
    "self": "http://localhost:8090/jira/rest/api/2.0/status/10000",
    "description": "The issue is currently being worked on.",
    "iconUrl": "http://localhost:8090/jira/images/icons/progress.gif",
    "name": "In Progress",
    "id": "10000",
    "statusCategory": {
        "self": "http://localhost:8090/jira/rest/api/2.0/statuscategory/1",
        "id": 1,
        "key": "in-flight",
        "colorName": "yellow",
        "name": "In Progress"
    }
}
*/
//betteralign:skip
type JIRAStatus struct {
	Self           string             `json:"self,omitempty"`
	Description    string             `json:"description,omitempty"`
	IconURL        string             `json:"iconUrl,omitempty"`
	Name           string             `json:"name,omitempty"`
	ID             string             `json:"id,omitempty"`
	StatusCategory JIRAStatusCategory `json:"statusCategory,omitempty"`
}

//betteralign:skip
type JIRAStatusCategory struct {
	Self      string `json:"self,omitempty"`
	Key       string `json:"key,omitempty"`
	ColorName string `json:"colorName,omitempty"`
	Name      string `json:"name,omitempty"`
	ID        int    `json:"id,omitempty"`
}

type jiraIssueFields struct {
	CustomFields
	JiraFields
}

func (ji JIRAIssue) String() string {
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(ji); err != nil {
		panic(err)
	}
	return buf.String()
}

type CustomFields map[string]json.RawMessage

func (cf CustomFields) String() string {
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.Encode(cf)
	return buf.String()
}

var jiraIssueFieldsOnce sync.Once
var jiraIssueFieldsAlreadyStored map[string]struct{}

func (issue *jiraIssueFields) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &issue.JiraFields); err != nil {
		return err
	}
	if err := json.Unmarshal(b, &issue.CustomFields); err != nil {
		return err
	}
	const customFieldPrefix = "customfield_"
	jiraIssueFieldsOnce.Do(func() {
		t := reflect.TypeOf(JiraFields{})
		jiraIssueFieldsAlreadyStored = make(map[string]struct{})
		for i, n := 0, t.NumField(); i < n; i++ {
			if s := t.Field(i).Tag.Get("json"); strings.HasPrefix(s, customFieldPrefix) {
				jiraIssueFieldsAlreadyStored[s] = struct{}{}
			}
		}
	})
	for k, v := range issue.CustomFields {
		if bytes.Equal(v, []byte("null")) ||
			bytes.Equal(v, []byte(`""`)) ||
			bytes.Equal(v, []byte(`{}`)) ||
			bytes.Equal(v, []byte(`[]`)) {
			delete(issue.CustomFields, k)
			continue
		}
		if !strings.HasPrefix(k, customFieldPrefix) {
			delete(issue.CustomFields, k)
			continue
		}
		if _, ok := jiraIssueFieldsAlreadyStored[k]; ok {
			delete(issue.CustomFields, k)
		}
	}
	return nil
}
func (issue jiraIssueFields) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(issue.JiraFields)
	b = bytes.TrimSpace(b)
	if err != nil || len(b) == 0 || len(issue.CustomFields) == 0 {
		return b, err
	}
	c, err := json.Marshal(issue.CustomFields)
	if err != nil {
		return b, err
	}
	c = bytes.TrimSpace(c)
	if b[len(b)-1] == '}' && c[0] == '{' && c[len(c)-1] == '}' {
		return append(append(b[:len(b)-1], ','), c[1:]...), nil
	}
	return b, fmt.Errorf("b[-1]=%c c=%s", b[len(b)-1], c)
}

// IssueGet gets the data for the issue, possibly filtering the returned fields.
func (svc *Jira) IssueGet(ctx context.Context, issueID string, fields []string) (JIRAIssue, error) {
	URL := svc.URLFor("issue", issueID, "")
	if len(fields) != 0 {
		q := URL.Query()
		q["fields"] = fields
		URL.RawQuery = q.Encode()
	}
	var issue struct {
		JIRAIssue
		JIRAError
	}
	req, err := svc.NewRequest(ctx, "GET", URL, nil)
	if err != nil {
		return issue.JIRAIssue, err
	}
	resp, err := svc.Do(ctx, req)
	if err == nil {
		logger.Debug("IssueGet do", "resp", resp, "error", err)
	} else {
		logger.Error("IssueGet do", "resp", resp, "error", err)
		return issue.JIRAIssue, err
	}
	if err = json.Unmarshal(resp, &issue); err == nil && len(issue.JIRAError.Messages) != 0 {
		err = &issue.JIRAError
	}
	return issue.JIRAIssue, err
}

// IssueGetStatus gets the status for the issue.
func (svc *Jira) IssueGetStatus(ctx context.Context, issueID string) (JIRAStatus, error) {
	URL := svc.URLFor("issue", issueID, "status")
	var status JIRAStatus
	req, err := svc.NewRequest(ctx, "GET", URL, nil)
	if err != nil {
		return status, err
	}
	resp, err := svc.Do(ctx, req)
	if err == nil {
		logger.Debug("IssueGetStatus do", "resp", resp, "error", err)
	} else {
		logger.Error("IssueGetStatus do", "resp", resp, "error", err)
		return status, err
	}
	err = json.Unmarshal(resp, &status)
	return status, err
}

// IssueTransiton sends a transition to Jira.
//
// POST /rest/api/2/issue/{issueIdOrKey}/transitions
//
// https://docs.atlassian.com/software/jira/docs/api/REST/9.12.2/#api/2/issue-doTransition
func (svc *Jira) IssueTransition(ctx context.Context, issueID, transitionID string) error {
	URL := svc.URLFor("issue", issueID, "transitions")
	type transitionReq struct {
		ID string `json:"id"`
	}
	b, err := json.Marshal(transitionReq{ID: transitionID})
	if err != nil {
		return err
	}
	req, err := svc.NewRequest(ctx, "POST", URL, b)
	if err != nil {
		return err
	}
	resp, err := svc.Do(ctx, req)
	if err == nil {
		logger.Debug("IssueTransition do", "resp", resp, "error", err)
	} else {
		logger.Error("IssueTransition do", "resp", resp, "error", err)
		return err
	}
	return err
}

// IssuePut puts (updates) an issue.
func (svc *Jira) IssuePut(ctx context.Context, issue JIRAIssue) error {
	b, err := json.Marshal(issue)
	if err != nil {
		return err
	}
	req, err := svc.NewRequest(ctx, "PUT", svc.URLFor("issue", issue.ID, ""), b)
	if err != nil {
		return err
	}
	resp, err := svc.Do(ctx, req)
	logger.Info("IssuePut", "resp", resp, "error", err)
	return err
}

// Load the tokens file.
func (svc *Jira) Load(tokensFile, jiraUser, jiraPassword string) {
	if svc.token == nil {
		svc.token = &Token{Username: jiraUser, Password: jiraPassword}
	} else {
		svc.token.Username, svc.token.Password = jiraUser, jiraPassword
	}
	svc.token.AuthURL = svc.URL.JoinPath("auth").String()
	if tokensFile == "" {
		return
	}
	svc.tokensFile = tokensFile
	fh, err := os.Open(tokensFile)
	if err != nil {
		logger.Error("open", "file", tokensFile, "error", err)
		return
	}
	var m map[string]*Token
	err = json.NewDecoder(fh).Decode(&m)
	fh.Close()
	if err == nil {
		svc.tokens = make(map[string]*Token, len(m))
		for k, v := range m {
			if v.IsValid() {
				svc.tokens[k] = v
			}
		}
		old := svc.token
		if act := svc.tokens[redactedURL(svc.URL)]; act != nil {
			if old.Username != "" {
				act.Username, act.Password = old.Username, old.Password
			}
			if act.AuthURL == "" {
				act.AuthURL = old.AuthURL
			}
			svc.token = act
		}
		return
	}
	if err != nil {
		logger.Error("parse", "file", fh.Name(), "error", err)
	} else {
		logger.Info("not valid", "file", fh.Name())
	}
	_ = os.Remove(fh.Name())
}

// URLFor returns the canonical url for the issue and the action.
func (svc *Jira) URLFor(typ, id, action string) *url.URL {
	URL := svc.URL.JoinPath("/"+typ, url.PathEscape(id))
	if action != "" {
		URL = URL.JoinPath(action)
	}
	return URL
}

// NewRequest creates a new request.
func (svc *Jira) NewRequest(ctx context.Context, method string, URL *url.URL, body []byte) (*http.Request, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, URL.String(), r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

type JIRAUser struct {
	Self         string `json:"self,omitempty"`
	Name         string `json:"name,omitempty"`
	Key          string `json:"key,omitempty"`
	EmailAddress string `json:"emailAddress,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	TimeZone     string `json:"timeZone,omitempty"`
	Active       bool   `json:"active,omitempty"`
}

type JIRAVisibility struct {
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}
type JIRAComment struct {
	Visibility   JIRAVisibility `json:"visibility,omitempty"`
	Self         string         `json:"self,omitempty"`
	ID           string         `json:"id,omitempty"`
	Body         string         `json:"body,omitempty"`
	Created      string         `json:"created,omitempty"`
	Updated      string         `json:"updated,omitempty"`
	Author       JIRAUser       `json:"author,omitempty"`
	UpdateAuthor JIRAUser       `json:"updateAuthor,omitempty"`
}

type JIRAPriority struct {
	Self    string `json:"self,omitempty"`
	IconURL string `json:"iconUrl,omitempty"`
	Name    string `json:"name,omitempty"`
	ID      string `json:"id,omitempty"`
}

type getCommentsResp struct {
	Comments   []JIRAComment `json:"comments,omitempty"`
	StartAt    int32         `json:"startAt,omitempty"`
	MaxResults int32         `json:"maxResults,omitempty"`
	Total      int32         `json:"total,omitempty"`
}

// IssueComments returns the comments for the issue.
func (svc *Jira) IssueComments(ctx context.Context, issueID string) ([]JIRAComment, error) {
	URL := svc.URLFor("issue", issueID, "comment")
	q := URL.Query()
	q.Set("startAt", "0")
	q.Set("maxResults", "65536")
	URL.RawQuery = q.Encode()
	req, err := svc.NewRequest(ctx, "GET", URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := svc.Do(ctx, req)
	logger.Info("IssueComments", "resp", resp, "error", err)
	if err != nil {
		return nil, err
	}
	var comments getCommentsResp
	err = json.Unmarshal(resp, &comments)
	return comments.Comments, err
}

type JSONCommentBody struct {
	Body string `json:"body,omitempty"`
	//Visibility JIRAVisibility `json:"visibility,omitempty"`
}

// IssueAddComment adds a comment to the issue.
func (svc *Jira) IssueAddComment(ctx context.Context, issueID, body string) error {
	URL := svc.URLFor("issue", issueID, "comment")
	b, err := json.Marshal(JSONCommentBody{Body: body}) //, Visibility: JIRAVisibility{Type: "role", Value: "Administrators"}})
	if err != nil {
		return err
	}
	req, err := svc.NewRequest(ctx, "POST", URL, b)
	if err != nil {
		return err
	}
	resp, err := svc.Do(ctx, req)
	logger.Info("IssueAddComment", "resp", resp, "error", err)
	if err != nil {
		return err
	}
	var comment JIRAComment
	return json.Unmarshal(resp, &comment)
}

// IssueAddAttachment uploads the attachment to the issue.
func (svc *Jira) IssueAddAttachment(ctx context.Context, issueID, fileName, mimeType string, body io.Reader) error {
	// This resource expects a multipart post. The media-type multipart/form-data is defined in RFC 1867. Most client libraries have classes that make dealing with multipart posts simple. For instance, in Java the Apache HTTP Components library provides a MultiPartEntity that makes it simple to submit a multipart POST.
	//
	// In order to protect against XSRF attacks, because this method accepts multipart/form-data, it has XSRF protection on it. This means you must submit a header of X-Atlassian-Token: no-check with the request, otherwise it will be blocked.
	//
	// The name of the multipart/form-data parameter that contains attachments must be "file"
	//
	// curl -D- -u admin:admin -X POST -H "X-Atlassian-Token: no-check" -F "file=@myfile.txt" http://myhost/rest/api/2/issue/TEST-123/attachments
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	w, err := mw.CreateFormFile("file", fileName)
	if err != nil {
		return err
	}
	if _, err = io.Copy(w, body); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}
	URL := svc.URLFor("issue", issueID, "attachments")
	req, err := http.NewRequestWithContext(ctx, "POST", URL.String(), bytes.NewReader(buf.Bytes()))
	if err != nil {
		return err
	}
	req.Header.Set("X-Atlassian-Token", "no-check")
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := svc.Do(ctx, req)
	logger.Info("IssueAddAttachment", "resp", resp, "error", err)
	if err != nil {
		return err
	}
	attachments := make([]JIRAAttachment, 0, 1)
	return json.Unmarshal(resp, &attachments)
}

type JIRAAttachment struct {
	Self      string   `json:"self,omitempty"`
	Filename  string   `json:"filename,omitempty"`
	Created   string   `json:"created,omitempty"`
	MimeType  string   `json:"mimeType,omitempty"`
	Content   string   `json:"content,omitempty"`
	Thumbnail string   `json:"thumbnail,omitempty"`
	Author    JIRAUser `json:"author,omitempty"`
	Size      int      `json:"size,omitempty"`
}

// IssueTransitions returns the possible transitions for the issue.
//
// GET /rest/api/2/issue/{issueIdOrKey}/transitions
func (svc *Jira) IssueTransitions(ctx context.Context, issueID string, fields bool) ([]JIRATransition, error) {
	URL := svc.URLFor("issue", issueID, "transitions")
	if fields {
		q := URL.Query()
		q.Set("expand", "transitions.fields")
		URL.RawQuery = q.Encode()
	}
	req, err := svc.NewRequest(ctx, "GET", URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := svc.Do(ctx, req)
	logger.Info("IssueTransitions", "resp", resp, "error", err)
	if err != nil {
		return nil, err
	}
	var transitions getTransitionsResp
	err = json.Unmarshal(resp, &transitions)
	return transitions.Transitions, err
}

type (
	getTransitionsResp struct {
		Expand      string           `json:"expand"`
		Transitions []JIRATransition `json:"transitions"`
	}

	JIRATransition struct {
		Fields         map[string]JIRAField `json:"fields"`
		ID             string               `json:"id"`
		Name           string               `json:"name"`
		To             JIRAStatus           `json:"to"`
		OpsbarSequence int                  `json:"opsbarSequence"`
	}

	/*
		"assignee": {
		        "autoCompleteUrl": "https://dc1jralappvt101.hu.emea.aegon.com/rest/api/latest/user/assignable/search?issueKey=INCIDENT-66591\u0026username=",
		        "fieldId": "assignee",
		        "name": "Assignee",
		        "operations": [
		          "set"
		        ],
		        "required": false,
		        "schema": {
		          "system": "assignee",
		          "type": "user"
		        }
		      },
		      "customfield_14324": {
		        "allowedValues": [
		          {
		            "disabled": false,
		            "id": "17162",
		            "self": "https://dc1jralappvt101.hu.emea.aegon.com/rest/api/2/customFieldOption/17162",
		            "value": "AAT (Advanced Analytics Team)"
		          },
	*/
	JIRAField struct {
		AllowedValues   []JIRAAllowedValues
		AutoCompleteURL string     `json:"autoCompleteUrl"`
		ID              string     `json:"fieldId"`
		Name            string     `json:"name"`
		Operations      []string   `json:"operations"`
		Required        bool       `json:"required"`
		Schema          JIRASchema `json:"schema"`
	}

	JIRASchema struct {
		System   string `json:"system"`
		Type     string `json:"type"`
		Items    string `json:"items"`
		Custom   string `json:"string"`
		CustomID int    `json:"customId"`
	}

	JIRAAllowedValues struct {
		ID       string `json:"id"`
		Self     string `json:"self"`
		Value    string `json:"value"`
		Disabled bool   `json:"disabled"`
	}
	/*
			"update": {
		        "comment": [
		            {
		                "add": {
		                    "body": "Bug has been fixed."
		                }
		            }
		        ]
		    },
	*/
	JIRACommentOpAdd struct {
		Body string `json:"body"`
	}
	JIRACommentOp struct {
		Add JIRACommentOpAdd `json:"add"`
	}
	JIRATransitionBody struct {
		Update struct {
			Comment []JIRACommentOp `json:"comment,omitempty"`
		} `json:"update,omitempty"`
		Transition struct {
			ID string `json:"id"`
		} `json:"transition"`
	}
)

// IssueDoTransition transits the issue's status.
func (svc *Jira) IssueDoTransition(ctx context.Context, issueID, transition, comment string) error {
	URL := svc.URLFor("issue", issueID, "transitions")
	var body JIRATransitionBody
	if comment != "" {
		body.Update.Comment = append(body.Update.Comment,
			JIRACommentOp{Add: JIRACommentOpAdd{Body: comment}},
		)
	}
	body.Transition.ID = transition
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := svc.NewRequest(ctx, "POST", URL, b)
	if err != nil {
		return err
	}
	resp, err := svc.Do(ctx, req)
	logger.Info("IssueDoTransition", "resp", resp, "error", err)
	if err != nil {
		return err
	}
	// var comment JIRAComment
	// return json.Unmarshal(resp, &comment)
	return nil
}

/*
   "expand": "transitions",
    "transitions": [
        {
            "id": "2",
            "name": "Close Issue",
            "opsbarSequence": 10,
            "to": {
                "self": "http://localhost:8090/jira/rest/api/2.0/status/10000",
                "description": "The issue is currently being worked on.",
                "iconUrl": "http://localhost:8090/jira/images/icons/progress.gif",
                "name": "In Progress",
                "id": "10000",
                "statusCategory": {
                    "self": "http://localhost:8090/jira/rest/api/2.0/statuscategory/1",
                    "id": 1,
                    "key": "in-flight",
                    "colorName": "yellow",
                    "name": "In Progress"
                }
            },
            "fields": {
                "summary": {
                    "required": false,
                    "schema": {
                        "type": "array",
                        "items": "option",
                        "custom": "com.atlassian.jira.plugin.system.customfieldtypes:multiselect",
                        "customId": 10001
                    },
                    "name": "My Multi Select",
                    "fieldId": "customfield_10000",
                    "hasDefaultValue": false,
                    "operations": [
                        "set",
                        "add"
                    ],
                    "allowedValues": [
                        "red",
                        "blue",
                        "default value"
                    ]
                }
            }
        },
*/

// IssueDoTransitionTo transits the issue's status.
func (svc *Jira) IssueDoTransitionTo(ctx context.Context, issueID, targetStatus, comment string) error {
	/*
		Jira státuszváltás	Jira Transition ID
		„New”  „In progress”	11
		„On hold”  „In progress”	51
		„In progress”  „Resolved”	21
		„On hold” „Resolved”	61
		„In progress”  „On hold”	41
	*/
	transitions, err := svc.IssueTransitions(ctx, issueID, false)
	logger.Warn("IssueTransitions", "issueID", issueID, "error", err)
	if err != nil {
		return fmt.Errorf("Get status of %q: %w", issueID, err)
	}
	possible := make(map[string]JIRATransition, len(transitions))
	for _, t := range transitions {
		possible[t.ID] = t
	}

	wanted := make([]string, 0, 2)
	switch targetStatus {
	case "IN_PROGRESS":
		wanted = append(wanted, "51")
	case "CLOSED", "RESOLVED":
		wanted = append(wanted, "21", "61")
	case "ON_HOLD":
		wanted = append(wanted, "41")
	}
	for _, w := range wanted {
		if t, ok := possible[w]; ok {
			err := svc.IssueDoTransition(ctx, issueID, t.ID, comment)
			if err != nil {
				err = fmt.Errorf("%q transition %q: %w",
					issueID, t.ID, err)
			}
			return err
		}
	}
	logger.Warn("no wanted found possible", "target", targetStatus, "wanted", wanted, "possible", possible)
	return nil
}

// Do the request with the tokens.
func (svc *Jira) Do(ctx context.Context, req *http.Request) (json.RawMessage, error) {
	b, changed, err := svc.token.do(ctx, svc.HTTPClient, req)
	if changed {
		if svc.tokens == nil {
			svc.tokens = make(map[string]*Token)
		}
		svc.tokens[redactedURL(svc.URL)] = svc.token
		if svc.tokensFile != "" {
			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(svc.tokens); err != nil {
				logger.Error("marshal tokens", "error", err)
			} else if err := renameio.WriteFile(svc.tokensFile, buf.Bytes(), 0600); err != nil {
				logger.Error("write token", "file", svc.tokensFile, "error", err)
			}
		}
	}
	if err != nil {
		return b, err
	}
	return b, nil
}

type rawToken struct {
	JSessionID   string `json:"JSESSIONID,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	IssuedAt     string `json:"issued_at,omitempty"`
	ExpiresIn    string `json:"expires_in,omitempty"`
	RefreshCount string `json:"refresh_count,omitempty"`
	JIRAError
}

type Token struct {
	till               time.Time
	AuthURL            string
	Username, Password string
	rawToken
	mu sync.Mutex
}

func (t *Token) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &t.rawToken); err != nil {
		return err
	}
	//logger.Debug("UnmarshalJSON", "b", string(b), "raw", fmt.Sprintf("%#v", t.rawToken))
	return t.init()
}
func (t *Token) init() error {
	logger.Debug("init", "raw", t.rawToken)
	if t.rawToken.JIRAError.IsValid() {
		return &t.rawToken.JIRAError
	}
	issuedAt, err := strconv.ParseInt(t.IssuedAt, 10, 64)
	if err != nil {
		return fmt.Errorf("parse issuedAt(%q): %w", t.IssuedAt, err)
	}
	expiresIn, err := strconv.ParseInt(t.ExpiresIn, 10, 64)
	if err != nil {
		return fmt.Errorf("parse expiresIn(%q): %w", t.ExpiresIn, err)
	}
	issued := time.Unix(issuedAt/1000, issuedAt%1000)
	t.till = issued.Add(time.Duration(expiresIn) * time.Second)
	logger.Debug("Unmarshal", "issuedAt", issuedAt, "issued", issued, "expiresIn", expiresIn, "till", t.till)
	return nil
}
func (t *Token) IsValid() bool {
	return t != nil && t.JSessionID != "" && time.Now().Before(t.till) && !t.rawToken.JIRAError.IsValid()
}

type JIRAError struct {
	Code     string   `json:"ErrorCode,omitempty"`
	Message  string   `json:"Error,omitempty"`
	Fault    Fault    `json:"fault,omitempty"`
	Messages []string `json:"errorMessages,omitempty"`
}
type Fault struct {
	Code   string      `json:"faultstring,omitempty"`
	Detail FaultDetail `json:"detail,omitempty"`
}
type FaultDetail struct {
	Message string `json:"errorcode,omitempty"`
}
type userPass struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func (je *JIRAError) Error() string {
	var buf strings.Builder
	if je.Code != "" {
		buf.WriteString(je.Code + ": " + je.Message)
	} else if je.Fault.Code != "" {
		buf.WriteString(je.Fault.Code + ": " + je.Fault.Detail.Message)
	}
	for _, m := range je.Messages {
		if buf.Len() != 0 {
			buf.WriteString("; ")
		}
		buf.WriteString(m)
	}
	return buf.String()
}
func (je *JIRAError) IsValid() bool {
	return je != nil && (je.Code != "" || je.Fault.Code != "" || len(je.Messages) != 0)
}

var (
	errAuthenticate = errors.New("authentication error")

	authStrategy = retry.Strategy{
		Delay: time.Second, MaxDelay: 5 * time.Second, MaxCount: 3,
		MaxDuration: time.Minute,
	}
	requestStrategy = retry.Strategy{
		Delay: time.Second, MaxDelay: 5 * time.Minute,
		MaxDuration: 24 * time.Hour,
	}
)

func (t *Token) do(ctx context.Context, httpClient *http.Client, req *http.Request) ([]byte, bool, error) {
	logEnabled := logger.Enabled(ctx, slog.LevelDebug)
	if httpClient == nil {
		httpClient = http.DefaultClient
		if httpClient.Transport == nil {
			httpClient.Transport = http.DefaultTransport
		}
		if logEnabled {
			httpClient.Transport = loghttp.Transport(httpClient.Transport)
		} else {
			httpClient.Transport = gzhttp.Transport(httpClient.Transport)
		}
		logger.Debug("logEnabled", "logtransport", httpClient.Transport)
	}
	var respBuf bytes.Buffer
	t.mu.Lock()
	defer t.mu.Unlock()
	changed, err := t.ensure(ctx, httpClient)
	if err != nil {
		return nil, false, err
	}
	if req == nil {
		return nil, changed, nil
	}
	/*
	   2.
	   request:
	   curl --location --request POST 'https://partnerapi-uat.aegon.hu/partner/v1/ticket/update/issue' \
	   --header 'Content-Type: application/json' \
	   --header 'Cookie: JSESSIONID=...; TS0126a004=015d4139a83807c002e8dd16d46fa16563299b17c4a228ff33b64e12ada62f8cd7829575e919a595aefcd7736d6717351a163defa1; atlassian.xsrf.token=B0BO-X7QB-KBRG-M4RU_23574bc6e7a2f17160a6128c30ee1a58a7ec4eb5_lin' \
	   --header 'Authorization: Bearer ...' \
	*/
	req.Header.Set("Cookie", "JSESSIONID="+t.JSessionID)
	req.Header.Set("Authorization", "Bearer "+t.AccessToken)
	var reqBody *bytes.Buffer
	try := func() error {
		start := time.Now()
		if reqBody == nil {
			reqBody = new(bytes.Buffer)
			req.Body = struct {
				io.Reader
				io.Closer
			}{io.TeeReader(req.Body, reqBody), io.NopCloser(nil)}
		} else {
			req.Body = struct {
				io.Reader
				io.Closer
			}{bytes.NewReader(reqBody.Bytes()), io.NopCloser(nil)}
		}
		resp, err := httpClient.Do(req.WithContext(ctx))
		if err != nil {
			logger.Error("do", "url", req.URL.String(), "method", req.Method, "dur", time.Since(start).String(), "error", err)
			return err
		}
		if resp == nil {
			return fmt.Errorf("empty response")
		}
		logger.Info("do", "url", req.URL.String(), "method", req.Method, "dur", time.Since(start).String(), "hasBody", resp.Body != nil, "status", resp.Status)
		if resp.Body == nil {
			return nil
		}
		respBuf.Reset()
		_, err = io.Copy(&respBuf, resp.Body)
		resp.Body.Close()
		if err != nil {
			logger.Error("read request", "error", err)
			return err
		}
		if bytes.Contains(respBuf.Bytes(), []byte(`"ErrorCode"`)) ||
			bytes.Contains(respBuf.Bytes(), []byte(`"Error"`)) ||
			bytes.Contains(respBuf.Bytes(), []byte(`"fault"`)) {
			var jerr JIRAError
			err = json.Unmarshal(respBuf.Bytes(), &jerr)
			if err != nil {
				logger.Error("Unmarshal JIRAError", "jErr", jerr, "jErrS", fmt.Sprintf("%#v", jerr), "buf", respBuf.String(), "error", err)
			} else {
				logger.Debug("Unmarshal JIRAError", "jErr", jerr, "jErrS", fmt.Sprintf("%#v", jerr))
			}
			if err == nil && jerr.IsValid() {
				if jerr.Code == "" {
					jerr.Code = resp.Status
				}
				return &jerr
			}
		}
		if resp.StatusCode >= 400 {
			return &JIRAError{Code: resp.Status, Message: respBuf.String()}
		}
		return nil
	}
	if err = try(); err != nil {
		for iter := requestStrategy.Start(); ; {
			var jerr *JIRAError
			if err = try(); err == nil || errors.As(err, &jerr) {
				err = nil
				break
			}
			logger.Error("try", "count", iter.Count(), "error", err)
			if !iter.Next(ctx.Done()) {
				break
			}
		}
	}

	return respBuf.Bytes(), changed, err
}

func (t *Token) ensure(ctx context.Context, httpClient *http.Client) (bool, error) {
	var reqBuf, respBuf bytes.Buffer
	logEnabled := logger.Enabled(ctx, slog.LevelDebug)
	if logEnabled {
		logger.Debug("IsValid", "token", t, "valid", t.IsValid())
	}
	var changed bool
	if !t.IsValid() {
		if t.Username == "" || t.Password == "" || t.AuthURL == "" {
			return changed, fmt.Errorf("empty JIRA username/password/AuthURL")
		}
		/*
		   curl --location --request POST 'https://partnerapi-uat.aegon.hu/partner/v1/ticket/update/auth?grant_type=password' \
		   --header 'Content-Type: application/json' \
		   --header 'Authorization: Basic ...' \
		   --data-raw '{ "username": "svc_unosoft", "password": "5h9RP97@qK6l"}'
		*/

		if err := json.NewEncoder(&reqBuf).Encode(userPass{
			Username: t.Username, Password: t.Password,
		}); err != nil {
			return changed, err
		}
		try := func() error {
			req, err := http.NewRequestWithContext(ctx, "POST", t.AuthURL+"?grant_type=password", bytes.NewReader(reqBuf.Bytes()))
			if err != nil {
				return err
			}
			req.GetBody = func() (io.ReadCloser, error) {
				return struct {
					io.Reader
					io.Closer
				}{bytes.NewReader(reqBuf.Bytes()), io.NopCloser(nil)}, nil
			}
			if logEnabled {
				logger.Debug("authenticate", "url", t.AuthURL, "body", reqBuf.String())
			}
			req.Header.Set("Content-Type", "application/json")
			start := time.Now()
			resp, err := httpClient.Do(req.WithContext(ctx))
			if err != nil {
				logger.Error("authenticate", "dur", time.Since(start).String(), "url", t.AuthURL, "error", err)
				return fmt.Errorf("%w: %w", errAuthenticate, err)
			}
			if resp == nil || resp.Body == nil {
				return fmt.Errorf("empty response")
			}
			respBuf.Reset()
			_, err = io.Copy(&respBuf, resp.Body)
			resp.Body.Close()
			if logEnabled {
				logger.Debug("authenticate", "response", respBuf.String())
			}
			if err != nil {
				return fmt.Errorf("%w: %w", errAuthenticate, err)
			} else if respBuf.Len() == 0 {
				return fmt.Errorf("%s: empty response", errAuthenticate)
			} else if err = json.Unmarshal(respBuf.Bytes(), &t); err != nil {
				return fmt.Errorf("%w: decode %q: %w", errAuthenticate, respBuf.String(), err)
			} else if !t.IsValid() {
				return fmt.Errorf("%w: got invalid token: %+v", errAuthenticate, t)
			}
			return nil
		}
		if err := try(); err != nil {
			for iter := authStrategy.Start(); ; {
				if err = try(); err == nil {
					break
				}
				logger.Error("try", "count", iter.Count(), "error", err)
				if !iter.Next(ctx.Done()) {
					return changed, err
				}
			}
		}

		changed = true
		/*
		   answer:
		   {
		       "JSESSIONID": "1973D50D4C576BFBAA889B8726A2FF77",
		       "issued_at": "1658754363080",
		       "access_token": "iugVuMjlGng4Lwgdj3LbcE3ehGIB",
		       "expires_in": "7199",
		       "refresh_count": "0"
		   }
		*/
	}
	return changed, nil
}

func redactedURL(u *url.URL) string {
	if u.User == nil {
		return u.String()
	}
	if p, _ := u.User.Password(); p != "" && u.User.Username() == "" {
		return u.String()
	}
	ru := *u
	ru.User = nil
	return ru.String()
}
