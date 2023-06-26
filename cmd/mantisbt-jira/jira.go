// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/renameio"
	"github.com/klauspost/compress/gzhttp"

	"golang.org/x/exp/slog"
)

// https://partnerapi-uat.aegon.hu/partner/v1/ticket/update/openapi.json
type Jira struct {
	URL        *url.URL
	tokens     map[string]Token
	HTTPClient *http.Client
	token      Token
	tokensFile string
}

type JIRAIssueType struct {
	Self        string `json:"self"`
	ID          string `json:"id"`
	Description string `json:"description"`
	IconURL     string `json:"iconUrl"`
	Name        string `json:"name"`
	Subtask     bool   `json:"subtask"`
	AvatarID    int    `json:"avatarId"`
}

type JIRAIssue struct {
	jiraIssue
	customFields
}

// https://mholt.github.io/json-to-go/
type jiraIssue struct {
	Fields struct {
		Timetracking struct {
		} `json:"timetracking"`
		Aggregatetimeoriginalestimate interface{} `json:"aggregatetimeoriginalestimate"`
		Resolution                    interface{} `json:"resolution"`
		Environment                   interface{} `json:"environment"`
		Duedate                       interface{} `json:"duedate"`
		Timeestimate                  interface{} `json:"timeestimate"`
		Aggregatetimeestimate         interface{} `json:"aggregatetimeestimate"`
		Timespent                     interface{} `json:"timespent"`
		Aggregatetimespent            interface{} `json:"aggregatetimespent"`
		Resolutiondate                interface{} `json:"resolutiondate"`
		Timeoriginalestimate          interface{} `json:"timeoriginalestimate"`
		MantisID                      string      `json:"customfield_15902"` // MantisID
		Customfield11100              struct {
			OngoingCycle struct {
				GoalDuration struct {
					Friendly string `json:"friendly"`
					Millis   int    `json:"millis"`
				} `json:"goalDuration"`
				ElapsedTime struct {
					Friendly string `json:"friendly"`
					Millis   int    `json:"millis"`
				} `json:"elapsedTime"`
				RemainingTime struct {
					Friendly string `json:"friendly"`
					Millis   int    `json:"millis"`
				} `json:"remainingTime"`
				StartTime struct {
					Iso8601     string `json:"iso8601"`
					Jira        string `json:"jira"`
					Friendly    string `json:"friendly"`
					EpochMillis int64  `json:"epochMillis"`
				} `json:"startTime"`
				BreachTime struct {
					Iso8601     string `json:"iso8601"`
					Jira        string `json:"jira"`
					Friendly    string `json:"friendly"`
					EpochMillis int64  `json:"epochMillis"`
				} `json:"breachTime"`
				Breached            bool `json:"breached"`
				Paused              bool `json:"paused"`
				WithinCalendarHours bool `json:"withinCalendarHours"`
			} `json:"ongoingCycle"`
			ID    string `json:"id"`
			Name  string `json:"name"`
			Links struct {
				Self string `json:"self"`
			} `json:"_links"`
			CompletedCycles []interface{} `json:"completedCycles"`
		} `json:"customfield_11100"`
		Status struct {
			Self           string `json:"self"`
			Description    string `json:"description"`
			IconURL        string `json:"iconUrl"`
			Name           string `json:"name"`
			ID             string `json:"id"`
			StatusCategory struct {
				Self      string `json:"self"`
				Key       string `json:"key"`
				ColorName string `json:"colorName"`
				Name      string `json:"name"`
				ID        int    `json:"id"`
			} `json:"statusCategory"`
		} `json:"status"`
		Customfield14326 JIRAUser `json:"customfield_14326"`
		Assignee         JIRAUser `json:"assignee"`
		Reporter         JIRAUser `json:"reporter"`
		Creator          JIRAUser `json:"creator"`
		Project          struct {
			Self           string `json:"self"`
			ID             string `json:"id"`
			Key            string `json:"key"`
			Name           string `json:"name"`
			ProjectTypeKey string `json:"projectTypeKey"`
		} `json:"project"`
		Security struct {
			Self        string `json:"self"`
			ID          string `json:"id"`
			Description string `json:"description"`
			Name        string `json:"name"`
		} `json:"security"`
		Priority struct {
			Self    string `json:"self"`
			IconURL string `json:"iconUrl"`
			Name    string `json:"name"`
			ID      string `json:"id"`
		} `json:"priority"`
		Summary          string `json:"summary"`
		Description      string `json:"description"`
		LastViewed       string `json:"lastViewed"`
		Updated          string `json:"updated"`
		Created          string `json:"created"`
		Customfield10009 struct {
			Links struct {
				JiraRest string `json:"jiraRest"`
				Web      string `json:"web"`
				Self     string `json:"self"`
			} `json:"_links"`
			RequestType struct {
				ID    string `json:"id"`
				Links struct {
					Self string `json:"self"`
				} `json:"_links"`
				Name          string   `json:"name"`
				Description   string   `json:"description"`
				HelpText      string   `json:"helpText"`
				ServiceDeskID string   `json:"serviceDeskId"`
				GroupIds      []string `json:"groupIds"`
			} `json:"requestType"`
			CurrentStatus struct {
				Status     string `json:"status"`
				StatusDate struct {
					Iso8601     string `json:"iso8601"`
					Jira        string `json:"jira"`
					Friendly    string `json:"friendly"`
					EpochMillis int64  `json:"epochMillis"`
				} `json:"statusDate"`
			} `json:"currentStatus"`
		} `json:"customfield_10009"`
		Customfield14342 struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Links struct {
				Self string `json:"self"`
			} `json:"_links"`
			CompletedCycles []interface{} `json:"completedCycles"`
		} `json:"customfield_14342"`
		Customfield14344 struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Links struct {
				Self string `json:"self"`
			} `json:"_links"`
			CompletedCycles []interface{} `json:"completedCycles"`
		} `json:"customfield_14344"`
		Customfield11101 struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Links struct {
				Self string `json:"self"`
			} `json:"_links"`
			CompletedCycles []struct {
				GoalDuration struct {
					Friendly string `json:"friendly"`
					Millis   int    `json:"millis"`
				} `json:"goalDuration"`
				ElapsedTime struct {
					Friendly string `json:"friendly"`
					Millis   int    `json:"millis"`
				} `json:"elapsedTime"`
				RemainingTime struct {
					Friendly string `json:"friendly"`
					Millis   int    `json:"millis"`
				} `json:"remainingTime"`
				StartTime struct {
					Iso8601     string `json:"iso8601"`
					Jira        string `json:"jira"`
					Friendly    string `json:"friendly"`
					EpochMillis int64  `json:"epochMillis"`
				} `json:"startTime"`
				StopTime struct {
					Iso8601     string `json:"iso8601"`
					Jira        string `json:"jira"`
					Friendly    string `json:"friendly"`
					EpochMillis int64  `json:"epochMillis"`
				} `json:"stopTime"`
				Breached bool `json:"breached"`
			} `json:"completedCycles"`
		} `json:"customfield_11101"`
		Customfield14343 struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Links struct {
				Self string `json:"self"`
			} `json:"_links"`
			CompletedCycles []interface{} `json:"completedCycles"`
		} `json:"customfield_14343"`
		Customfield15113 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_15113"`
		Customfield14451 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14451"`
		Customfield15109 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_15109"`
		Customfield14408 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14408"`
		Customfield14321 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14321"`
		Customfield15143 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_15143"`
		Customfield14325 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14325"`
		Customfield15104 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_15104"`
		Customfield15114 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_15114"`
		Customfield14339 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14339"`
		Customfield14423 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14423"`
		Customfield14404 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14404"`
		Customfield14449 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14449"`
		Customfield14425 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14425"`
		Worklog struct {
			Worklogs   []interface{} `json:"worklogs"`
			StartAt    int           `json:"startAt"`
			MaxResults int           `json:"maxResults"`
			Total      int           `json:"total"`
		} `json:"worklog"`
		Versions         []interface{} `json:"versions"`
		Components       []interface{} `json:"components"`
		Labels           []interface{} `json:"labels"`
		Customfield10008 []JIRAUser    `json:"customfield_10008"`
		Subtasks         []struct {
			ID     string `json:"id"`
			Key    string `json:"key"`
			Self   string `json:"self"`
			Fields struct {
				Summary string `json:"summary"`
				Status  struct {
					Self           string `json:"self"`
					Description    string `json:"description"`
					IconURL        string `json:"iconUrl"`
					Name           string `json:"name"`
					ID             string `json:"id"`
					StatusCategory struct {
						Self      string `json:"self"`
						Key       string `json:"key"`
						ColorName string `json:"colorName"`
						Name      string `json:"name"`
						ID        int    `json:"id"`
					} `json:"statusCategory"`
				} `json:"status"`
				Priority struct {
					Self    string `json:"self"`
					IconURL string `json:"iconUrl"`
					Name    string `json:"name"`
					ID      string `json:"id"`
				} `json:"priority"`
				IssueType JIRAIssueType `json:"issuetype"`
			} `json:"fields"`
		} `json:"subtasks"`
		Attachment       []interface{} `json:"attachment"`
		Customfield15216 []struct {
			Active bool `json:"active"`
		} `json:"customfield_15216"`
		Customfield14336 []struct {
			Name string `json:"name"`
			Self string `json:"self"`
		} `json:"customfield_14336"`
		Issuelinks       []interface{} `json:"issuelinks"`
		FixVersions      []interface{} `json:"fixVersions"`
		Customfield15217 []struct {
			Active bool `json:"active"`
		} `json:"customfield_15217"`
		IssueType JIRAIssueType `json:"issuetype"`
		Watches   struct {
			Self       string `json:"self"`
			WatchCount int    `json:"watchCount"`
			IsWatching bool   `json:"isWatching"`
		} `json:"watches"`
		Votes struct {
			Self     string `json:"self"`
			Votes    int    `json:"votes"`
			HasVoted bool   `json:"hasVoted"`
		} `json:"votes"`
		Comment struct {
			Comments   []interface{} `json:"comments"`
			MaxResults int           `json:"maxResults"`
			Total      int           `json:"total"`
			StartAt    int           `json:"startAt"`
		} `json:"comment"`
		Progress struct {
			Progress int `json:"progress"`
			Total    int `json:"total"`
		} `json:"progress"`
		Aggregateprogress struct {
			Progress int `json:"progress"`
			Total    int `json:"total"`
		} `json:"aggregateprogress"`
		Workratio int `json:"workratio"`
	} `json:"fields"`
	Expand string `json:"expand"`
	ID     string `json:"id"`
	Self   string `json:"self"`
	Key    string `json:"key"`
}

type customFields map[string]json.RawMessage

func (issue *JIRAIssue) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &issue.jiraIssue); err != nil {
		return err
	}
	if err := json.Unmarshal(b, &issue.customFields); err != nil {
		return err
	}
	for k := range issue.customFields {
		if !strings.HasPrefix(k, "customfield_") {
			delete(issue.customFields, k)
		}
	}
	return nil
}
func (issue JIRAIssue) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(issue.jiraIssue)
	b = bytes.TrimSpace(b)
	if err != nil || len(b) == 0 || len(issue.customFields) == 0 {
		return b, err
	}
	c, err := json.Marshal(issue.customFields)
	if err != nil {
		return b, err
	}
	c = bytes.TrimSpace(c)
	if b[len(b)-1] == '}' && c[0] == '{' && c[len(c)-1] == '}' {
		return append(b[:len(b)-1], c[1:]...), nil
	}
	return b, fmt.Errorf("b[-1]=%c c=%s", b[len(b)-1], c)
}

func (svc *Jira) IssueGet(ctx context.Context, issueID string, fields []string) (JIRAIssue, error) {
	URL := svc.URLFor("issue", issueID, "")
	if len(fields) != 0 {
		q := URL.Query()
		q["fields"] = fields
		URL.RawQuery = q.Encode()
	}
	var issue JIRAIssue
	req, err := svc.NewRequest(ctx, "GET", URL, nil)
	if err != nil {
		return issue, err
	}
	resp, err := svc.Do(ctx, req)
	logger.Info("IssueGet do", "resp", resp, "error", err)
	if err != nil {
		return issue, err
	}
	err = json.Unmarshal(resp, &issue)
	return issue, err
}
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
func (svc *Jira) Load(tokensFile, jiraUser, jiraPassword string) {
	svc.token.Username, svc.token.Password = jiraUser, jiraPassword
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
	var m map[string]Token
	err = json.NewDecoder(fh).Decode(&m)
	fh.Close()
	if err == nil {
		svc.tokens = make(map[string]Token, len(m))
		for k, v := range m {
			if v.IsValid() {
				svc.tokens[k] = v
			}
		}
		old := svc.token
		svc.token = svc.tokens[redactedURL(svc.URL)]
		if old.Username != "" {
			svc.token.Username, svc.token.Password = old.Username, old.Password
		}
		if svc.token.AuthURL == "" {
			svc.token.AuthURL = old.AuthURL
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
func (svc *Jira) URLFor(typ, id, action string) *url.URL {
	URL := svc.URL.JoinPath("/"+typ, url.PathEscape(id))
	if action != "" {
		URL = URL.JoinPath(action)
	}
	return URL
}
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
	Self         string `json:"self"`
	Name         string `json:"name"`
	Key          string `json:"key"`
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName"`
	TimeZone     string `json:"timeZone"`
	Active       bool   `json:"active"`
}

type JIRAVisibility struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
type JIRAComment struct {
	Self         string         `json:"self"`
	ID           string         `json:"id"`
	Author       JIRAUser       `json:"author"`
	Body         string         `json:"body"`
	UpdateAuthor JIRAUser       `json:"updateAuthor"`
	Created      string         `json:"created"`
	Updated      string         `json:"updated"`
	Visibility   JIRAVisibility `json:"visibility"`
}

type getCommentsResp struct {
	Comments   []JIRAComment `json:"comments"`
	StartAt    int32         `json:"startAt"`
	MaxResults int32         `json:"maxResults"`
	Total      int32         `json:"total"`
}

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
	Body string `json:"body"`
	//Visibility JIRAVisibility `json:"visibility"`
}

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
	Author    JIRAUser `json:"author"`
	Self      string   `json:"self"`
	Filename  string   `json:"filename"`
	Created   string   `json:"created"`
	MimeType  string   `json:"mimeType"`
	Content   string   `json:"content"`
	Thumbnail string   `json:"thumbnail"`
	Size      int      `json:"size"`
}

func (svc *Jira) Do(ctx context.Context, req *http.Request) ([]byte, error) {
	b, changed, err := svc.token.do(ctx, svc.HTTPClient, req)
	if changed {
		if svc.tokens == nil {
			svc.tokens = make(map[string]Token)
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
	JSessionID   string `json:"JSESSIONID"`
	AccessToken  string `json:"access_token"`
	IssuedAt     string `json:"issued_at"`
	ExpiresIn    string `json:"expires_in"`
	RefreshCount string `json:"refresh_count"`
	JIRAError
}

type Token struct {
	till time.Time
	rawToken
	AuthURL            string
	Username, Password string
	mu                 sync.Mutex
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
	t.till = time.Unix(issuedAt/1000, issuedAt%1000).Add(time.Duration(expiresIn) * time.Second)
	logger.Debug("Unmarshal", "issuedAt", issuedAt, "expiresIn", expiresIn, "till", t.till)
	return nil
}
func (t *Token) IsValid() bool {
	return t != nil && t.JSessionID != "" && time.Now().Before(t.till)
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
	Username string `json:"username"`
	Password string `json:"password"`
}

func (je *JIRAError) Error() string {
	var buf strings.Builder
	if je.Code != "" {
		buf.WriteString(je.Code + ": " + je.Message)
	} else if je.Fault.Code != "" {
		buf.WriteString(je.Fault.Code + ": " + je.Fault.Detail.Message)
	}
	for _, m := range je.Messages {
		buf.WriteString("; ")
		buf.WriteString(m)
	}
	return buf.String()
}
func (je *JIRAError) IsValid() bool {
	return je != nil && (je.Code != "" || je.Fault.Code != "" || len(je.Messages) != 0)
}

func (t *Token) do(ctx context.Context, httpClient *http.Client, req *http.Request) ([]byte, bool, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
		if httpClient.Transport == nil {
			httpClient.Transport = http.DefaultTransport
		}
		httpClient.Transport = gzhttp.Transport(httpClient.Transport)
	}
	var buf bytes.Buffer
	t.mu.Lock()
	defer t.mu.Unlock()
	logger.Debug("IsValid", "token", t, "valid", t.IsValid())
	var changed bool
	if !t.IsValid() {
		if t.Username == "" || t.Password == "" || t.AuthURL == "" {
			return nil, changed, fmt.Errorf("empty JIRA username/password/AuthURL")
		}
		/*
		   curl --location --request POST 'https://partnerapi-uat.aegon.hu/partner/v1/ticket/update/auth?grant_type=password' \
		   --header 'Content-Type: application/json' \
		   --header 'Authorization: Basic ...' \
		   --data-raw '{ "username": "svc_unosoft", "password": "5h9RP97@qK6l"}'
		*/

		if err := json.NewEncoder(&buf).Encode(userPass{Username: t.Username, Password: t.Password}); err != nil {
			return nil, changed, err
		}
		req, err := http.NewRequestWithContext(ctx, "POST", t.AuthURL+"?grant_type=password", bytes.NewReader(buf.Bytes()))
		if err != nil {
			return nil, changed, err
		}
		logger.Debug("authenticate", "url", t.AuthURL, "body", buf.String())
		req.Header.Set("Content-Type", "application/json")
		start := time.Now()
		resp, err := httpClient.Do(req.WithContext(ctx))
		logger.Info("authenticate", "dur", time.Since(start).String(), "url", t.AuthURL, "error", err)
		if err != nil {
			return nil, changed, err
		}
		if resp == nil || resp.Body == nil {
			return nil, changed, fmt.Errorf("empty response")
		}
		buf.Reset()
		_, err = io.Copy(&buf, resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, changed, err
		}
		logger.Debug("authenticate", "response", buf.String())
		if err = json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&t); err != nil {
			return nil, changed, fmt.Errorf("decode %q: %w", buf.String(), err)
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
	logEnabled := logger.Enabled(ctx, slog.LevelDebug)
	if logEnabled {
		b, err := httputil.DumpRequestOut(req, true)
		logger.Debug("Do", "request", string(b), "dumpErr", err)
		if err != nil {
			return nil, changed, err
		}
	}
	start := time.Now()
	resp, err := httpClient.Do(req.WithContext(ctx))
	logger.Info("do", "url", req.URL.String(), "method", req.Method, "dur", time.Since(start).String(), "hasBody", resp.Body != nil, "status", resp.Status)
	if err != nil {
		return nil, changed, err
	}
	if resp == nil {
		return nil, changed, fmt.Errorf("empty response")
	}
	if logEnabled {
		b, err := httputil.DumpResponse(resp, true)
		logger.Debug("Do", "response", string(b), "dumpErr", err)
		if err != nil {
			return nil, changed, err
		}
	}
	if resp.Body == nil {
		return nil, changed, nil
	}
	buf.Reset()
	_, err = io.Copy(&buf, resp.Body)
	resp.Body.Close()
	if err != nil {
		logger.Error("read request", "error", err)
	}
	if bytes.Contains(buf.Bytes(), []byte(`"ErrorCode"`)) ||
		bytes.Contains(buf.Bytes(), []byte(`"Error"`)) ||
		bytes.Contains(buf.Bytes(), []byte(`"fault"`)) {
		var jerr JIRAError
		err = json.Unmarshal(buf.Bytes(), &jerr)
		if err != nil {
			logger.Error("Unmarshal JIRAError", "jErr", jerr, "jErrS", fmt.Sprintf("%#v", jerr), "buf", buf.String(), "error", err)
		} else {
			logger.Debug("Unmarshal JIRAError", "jErr", jerr, "jErrS", fmt.Sprintf("%#v", jerr))
		}
		if err == nil && jerr.IsValid() {
			if jerr.Code == "" {
				jerr.Code = resp.Status
			}
			return nil, changed, &jerr
		}
	}
	if resp.StatusCode >= 400 {
		return buf.Bytes(), changed, &JIRAError{Code: resp.Status, Message: buf.String()}
	}
	return buf.Bytes(), changed, nil
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
