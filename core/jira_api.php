<?php
require dirname(__FILE__).'/../vendor/autoload.php';

use JiraRestApi\Configuration\ArrayConfiguration;
use JiraRestApi\Issue\IssueService;

function create_issue_service(
	string $p_host, 
	string $p_jira_user, string $p_jira_password, 
	string $p_svc_user, string $p_svc_password,
	string $p_api_uri = null
) {
	$config = array(
		'jiraHost' => $p_host,
		'jiraUser' => $p_jira_user,
		'jiraPassword' => $p_jira_password,

		'jiraLogEnabled' => true,
		'jiraLogFile' => 'jira.log',
		'jiraLogLevel' => 'DEBUG',

		'useTokenBasedAuth' => false,
		//'personalAccessToken' => $p_token,
		'useV3RestApi' => false,
		'cookieAuthEnabled' => false,
		//'cookieFile' => './jira-cookie.txt',

		'curlOptVerbose' => true,
	);
	$svc = new IssueService(new ArrayConfiguration($config));
	if( $p_api_uri !== null ) {
		$svc->setAPIURI($p_api_uri);
	}
	$t_result = $svc->exec(
		'/auth?grant_type=password', 
		json_encode(array(
			'username' => $p_svc_user, 
			'password' => $p_svc_password,
		), JSON_UNESCAPED_UNICODE),
		'POST'
	);
	//echo "auth: $t_result";
	$t_result = json_decode($t_result, true);
	//$config['cookieAuthEnabled'] = true;
	$config['useTokenBasedAuth'] = true;
	$config['personalAccessToken'] = $t_result['access_token'];
	$svc = new IssueService(new ArrayConfiguration($config));
	if( $p_api_uri !== null ) {
		$svc->setAPIURI($p_api_uri);
	}

	return $svc;
}
function jira_new_comment($p_body) {
    $comment = new Comment();
    $comment->setBody($p_body);
    return $comment;
}

// vim: set noet shiftwidth=4:
