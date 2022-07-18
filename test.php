<?php
require 'vendor/autoload.php';

use JiraRestApi\Configuration\ArrayConfiguration;
use JiraRestApi\Auth\AuthService;
use JiraRestApi\Issue\IssueService;
use JiraRestApi\JiraClient;
use JiraRestApi\JiraException;

function create_issue_service($p_host, $p_user, $p_password, string $p_api_uri = null) {
	$config = new ArrayConfiguration(
		array(
			'jiraHost' => $p_host,
			'jiraUser' => $p_user,
			'jiraPassword' => $p_password,

			'jiraLogEnabled' => true,
			'jiraLogFile' => 'jira.log',
			'jiraLogLevel' => 'DEBUG',

			'useTokenBasedAuth' => false,
			//'personalAccessToken' => $p_token,
			'useV3RestApi' => false,
			'cookieAuthEnabled' => false,
			//'cookieFile' => './jira-cookie.txt',

			'curlOptVerbose' => true,
	));
	$svc = new IssueService($config);
	if( $p_api_uri !== null ) {
		$svc->setAPIURI($p_api_uri);
	}
	echo "auth:" . $svc->exec(
		'/auth?grant_type=password', 
		json_encode(array('username'=>'svc_unosoft', 'password'=>'?'), JSON_UNESCAPED_UNICODE),
		'POST');
//curl --location --request POST 'https://partnerapi-uat.aegon.hu/partner/v1/ticket/update/auth?grant_type=password'  --header 'Content-Type: application/json' -u :--data-raw '{ "username": "svc_unosoft", "password": "*********"}'
	$auth_svc = new AuthService($config);
// curl --location --request POST 'https://partnerapi-uat.aegon.hu/partner/v1/ticket/update/auth?grant_type=password' \
//   --header 'Content-Type: application/json' \
//   --header 'Authorization: Basic ' \
//   --data-raw '{ "username": "svc_unosoft", "password": "*********"}'
	if( $p_api_uri !== null ) {
		$auth_svc->setAPIURI($p_api_uri);
		$auth_svc->setURI( '' );
	}
	$auth_svc->setAPIURI('/auth');
	$session = $auth_svc->login($p_user, $p_password);
	echo "$session";
	var_dump($session);

	return $svc;
}

function jira_new_comment($p_body) {
	$comment = new Comment();
	$comment->setBody($p_body);
	return $comment;
}

$svc = create_issue_service(
	'https://partnerapi-uat.aegon.hu/partner/v1/ticket/update', 
	'',
	'',
	'',
);
var_dump($svc);
$svc->get('NONDEV-44', array() );

// vim: set noet shiftwidth=4:
