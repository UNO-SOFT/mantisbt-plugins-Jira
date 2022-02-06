<?php
require 'vendor/autoload.php';

use JiraRestApi\Configuration\ArrayConfiguration;
use JiraRestApi\Issue\IssueService;

function create_issue_service($p_host, $p_token) {
    return new IssueService(new ArrayConfiguration(
              array(
                   'jiraHost' => $p_host,
                    'tokenBasedAuth' => true,
                    'personalAccessToken' => $p_token,
                    'useV3RestApi' => true,
              )
       ));
}

function jira_new_comment($p_body) {
    $comment = new Comment();
    $comment->setBody($p_body);
    return $comment;
}
