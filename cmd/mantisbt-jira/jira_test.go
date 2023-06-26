// Copyright 2023 Tamás Gulácsi. All rights reserved.

package main

import (
	"encoding/json"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	var issue JIRAIssue
	if err := json.Unmarshal([]byte(`{
		"fields":{
			"customfield_15902":"12345",
			"customfield_num":1, 
			"customfield_string":"s", 
			"customfield_arr":["1",2]
		}
	}`), &issue,
	); err != nil {
		t.Fatal(err)
	}
	t.Log(issue)
	if issue.Fields.MantisID != "12345" {
		t.Error("mantisID:", issue.Fields.MantisID)
	}

}
