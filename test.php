<?php
require 'vendor/autoload.php';

use JiraRestApi\Configuration\ArrayConfiguration;
use JiraRestApi\Auth\AuthService;
use JiraRestApi\Issue\IssueService;
use JiraRestApi\JiraClient;
use JiraRestApi\JiraException;

function create_issue_service(
	string $p_host, 
	string $p_jira_user, string $p_jira_password, 
	string $p_svc_user, string $p_svc_password,
	string $p_api_uri = null,
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
		'cookieFile' => './jira-cookie.txt',

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
		'POST',
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

$svc = create_issue_service(
	'https://partnerapi-uat.aegon.hu/partner/v1/ticket/update', 
	p_jira_user: getenv('JIRA_USER'),
	p_jira_password: getenv('JIRA_PASSWORD'),
	p_svc_user: getenv('SVC_USER'), 
	p_svc_password: getenv('SVC_PASSWORD'), 
	p_api_uri: '',
);
//echo "search: " . var_dump($svc->search("1=1"));
$svc->get('INCIDENT-6508', array() );

// vim: set noet shiftwidth=4:
