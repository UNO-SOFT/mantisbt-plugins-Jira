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
                    'jiraRestApiV3' => true,
              )
       ));
}
