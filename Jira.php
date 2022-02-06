<?php
# :vim set noet:

if ( !defined( MANTIS_DIR ) ) {
	define(MANTIS_DIR, dirname(__FILE__) . '/../..' );
}
if ( !defined( MANTIS_CORE ) ) {
	define(MANTIS_CORE, MANTIS_DIR . '/core' );
}

require_once( MANTIS_DIR . '/core.php' );
require_once( config_get( 'class_path' ) . 'MantisPlugin.class.php' );
require_once( dirname(__FILE__).'/core/jira_api.php' );

require_api( 'install_helper_functions_api.php' );
require_api( 'authentication_api.php');

class JiraPlugin extends MantisPlugin {
	function register() {
		$this->name = 'Jira';	# Proper name of plugin
		$this->description = 'Jira syncrhonization';	# Short description of the plugin
		$this->page = '';		   # Default plugin page

		$this->version = '0.0.1';	 # Plugin version string
		$this->requires = array(	# Plugin dependencies, array of basename => version pairs
			'MantisCore' => '2.0.0'
			);

		$this->author = 'Tamás Gulácsi';		 # Author/team name
		$this->contact = 'T.Gulacsi@unosoft.hu';		# Author/team e-mail address
		$this->url = 'http://www.unosoft.hu';			# Support webpage
	}

	function config() {
		return array( 
			'jira_host' => plugin_config_get( 'jira_host', '' ),
			'jira_token' => plugin_config_get( 'jira_token', '' ),
		);
	}

	function hooks() {
		return array(
			'EVENT_UPDATE_BUG' => 'update_bug',
			'EVENT_BUGNOTE_ADD' => 'bugnote_add',
		);
	}

	function update_bug( $p_event_name, $p_params ) {
	}

	function bugnote_add( $p_event_name, $p_params ) {
	}

	function schema() {
		$opts = array(
			'mysql' => 'DEFAULT CHARSET=utf8',
			'pgsql' => 'WITHOUT OIDS'
		);
		return array(
			array( 'CreateTableSQL', array( plugin_table( 'current' ), "
				bug_id		I	NOTNULL UNSIGNED,
				user_id		I	NOTNULL UNSIGNED,
				cents		I	NOTNULL UNSIGNED",
				$opts)
			),
			array( 'CreateIndexSQL', array( 'idx_contributors_bugid', plugin_table( 'current' ), 'bug_id' ) ),

			array( 'CreateTableSQL', array( plugin_table( 'history' ) , "
				id			I	NOTNULL UNSIGNED PRIMARY AUTOINCREMENT,
				modifier_id	I	NOTNULL UNSIGNED,
				modified_at	I	NOTNULL DEFAULT 0,
				bug_id		I	NOTNULL UNSIGNED,
				user_id		I	NOTNULL UNSIGNED,
				cents		I	NOTNULL UNSIGNED",
				$opts)
			),
		);
	}
}

// vim: set noet:
