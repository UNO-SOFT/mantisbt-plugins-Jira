<?php
# :vim set noet:

if ( !defined( 'MANTIS_DIR' ) ) {
	define(MANTIS_DIR, dirname(__FILE__) . '/../..' );
}
if ( !defined( 'MANTIS_CORE' ) ) {
	define(MANTIS_CORE, MANTIS_DIR . '/core' );
}

require_once( MANTIS_DIR . '/core.php' );
require_once( config_get( 'class_path' ) . 'MantisPlugin.class.php' );
//require_once( dirname(__FILE__).'/core/jira_api.php' );

require_api( 'install_helper_functions_api.php' );
require_api( 'authentication_api.php');
require_api( 'custom_field_api.php' );
require_api( 'bugnote_api.php' );
require_api( 'file_api.php' );
require_api( 'database_api.php' );

class JiraPlugin extends MantisPlugin {
	private $issueid_field_id = 4;
	private $skip_reporter_id = 0;
	private $log_file = null;

	function __destruct() {
		if( $this->log_file ) {
			fclose( $this->log_file );
			$this->log_file = null;
		}
	}

	function register() {
		$this->name = 'Jira';	# Proper name of plugin
		$this->description = 'Jira synchronization';	# Short description of the plugin
		$this->page = '';		   # Default plugin page

		$this->version = '0.0.2';	 # Plugin version string
		$this->requires = array(	# Plugin dependencies, array of basename => version pairs
			'MantisCore' => '2.0.0'
		);

		$this->author = 'Tamás Gulácsi';		 # Author/team name
		$this->contact = 'T.Gulacsi@unosoft.hu';		# Author/team e-mail address
		$this->url = 'http://www.unosoft.hu';			# Support webpage
	}

	function config() {
		return array( 
			'base' => plugin_config_get( 'base', 'https://partnerapi-uat.aegon.hu/partner/v1/ticket/update' ),
			'user' => plugin_config_get( 'user', '' ),
			'password' => plugin_config_get( 'password', '' )
		);
	}

	function hooks() {
		return array(
			'EVENT_BUGNOTE_ADD' => 'bugnote_add',
			//'EVENT_BUGNOTE_EDIT' => 'bugnote_edit',
			'EVENT_MENU_MANAGE' => 'menu_manage',
		);
	}

	function menu_manage( ) {
			if ( access_get_project_level() >= MANAGER) {
					return array( '<a href="' . plugin_page( 'config.php' ) . '">'
							.  plugin_lang_get('config') . '</a>', );
			}
	}

	function bugnote_add( $p_event_name, $p_bug_id, $p_bugnote_id, $p_files ) {
		$this->log( 'bugnote_add(' . $p_event_name . ', ' . $p_bug_id . ' bugnote_id=' . $p_bugnote_id . ' files=' . var_export( $p_files, TRUE ) . ')' );
		if( $this->issueid_field_id === 0 ) {
			$this->issueid_field_id = custom_field_id_from_name( 'nyilvszám' );
		}
		if( $this->skip_reporter_id === 0 ) {
			$this->skip_reporter_id = user_get_id_by_name( 'jira' );
		}
		$t_issueid = custom_field_get_value( $this->issueid_field_id, $p_bug_id );
		$this->log( 'nyilvszam(' . $this->issueid_field_id . '): ' . $t_issueid );
		if( !$t_issueid ) {
			return;
		}

		$t_bugnote = null;
		if( $p_bugnote_id ) {
			$t_bugnote = bugnote_get( $p_bugnote_id );
		}
		if( $t_bugnote ) {
			$this->log( 'bugnote ' . $t_bugnote->view_state );
			if( VS_PUBLIC != $t_bugnote->view_state ) {
				return;
			}
			if( $t_bugnote->reporter_id == $this->skip_reporter_id ) {
				// feldolg a végét
				// <<Kiss.Balazs@aegon.hu>>
				$matches = array();
				if( preg_match('/<<([^>@]+@[^>]*)>>/', $t_bugnote->note, $matches) ) {
					$t_uid = user_get_id_by_email( $matches[1] );
					if( !$t_uid ) {
						$t_uid = user_get_id_by_email( strtolower( $matches[1] ) );
					}
$this->log( 'email: ' . var_export( $matches, TRUE ) . ' uid=' . $t_uid );
					if( $t_uid ) {
						$t_bugnote->reporter_id = $t_uid;
						db_param_push();
						$t_query = 'UPDATE {bugnote} SET reporter_id = ' . db_param() . ' WHERE bugnote_text_id = ' . db_param();
						db_query( $t_query, array( $t_uid, $t_bugnote->bugnote_text_id ) );
						db_param_push();

						$t_bugnote->note = str_replace( $matches[0], '', $t_bugnote->note );
						$t_query = 'UPDATE {bugnote_text} SET note = ' . db_param() . ' WHERE id = ' . db_param();
						db_query( $t_query, array( $t_bugnote->note, $t_bugnote->bugnote_text_id ) );
					}
				}
				return;
			}

$this->log( 'note length: ' .strlen( $t_bugnote->note ) );
			if( strlen($t_bugnote->note) !== 0 ) {
				$this->call("comment", $t_issueid, $t_bugnote->note);
$this->log( 'comment added' );
			}
		}
		if( count( $p_files ) == 0 ) {
			return;
		}

		$t_project_id = bug_get_field( $p_bug_id, 'project_id' );
		$this->log( 'project_id=' . $t_project_id);

		foreach( $p_files as $t_file ) {
			$t_diskfile = file_get_field( $t_file['id'], 'diskfile', 'bug' );
			if( !$t_diskfile ) {
				continue;
			}
			$t_local_disk_file = file_normalize_attachment_path( $t_diskfile, $t_project_id );
			$this->log( 'file=' . var_export( $t_file, TRUE ) . ', diskfile=' . $t_diskfile . ' local_disk_file=' . $t_local_disk_file );
			if( !$t_local_disk_file ) {
				continue;
			}
			$this->call( "attach", $t_issueid, $t_local_disk_file );
		}
	}

	function call( $p_subcommand, $p_issueid, $p_arg ) {
		$t_conf = $this->config();
		$t_args = array( '/usr/local/bin/mantisbt-jira' );
		foreach( $t_conf as $k => $v ) {
			if( $v ) {
				$t_args[] = escapeshellarg( '-jira-' . $k . '=' . $v );
			}
		}
		
		$t_output = array();
		$t_args = implode( $t_args, ' ' ) .
			' ' . escapeshellarg( $p_subcommand ) . 
			' ' . escapeshellarg( $p_issueid ) . 
			' ' . escapeshellarg( $p_arg );
		$this->log('calling ' . $t_args );
		// https://stackoverflow.com/questions/2320608/php-stderr-after-exec
		$t_pipes = array();
		// nosemgrep: php.lang.security.exec-use.exec-use
		$t_process = proc_open( $t_args, 
			array(
				1 => array("pipe", "w"),  // stdout
				2 => array("pipe", "w"),  // stderr
			),
			$t_pipes, '/' );
		fclose( $t_pipes[1] );
		$t_stderr = stream_get_contents( $t_pipes[2] );
		fclose( $t_pipes[2] );
		$t_rc = proc_close( $t_process );
		$this->log('got ' . $t_rc . ': stdout=' . var_export( $t_stdout, TRUE ) . ' stderr=' . var_export( $t_stderr, TRUE ) );
		return $t_rc == 0;
	}

	function log( $p_text ) {
		if( !$this->log_file ) {
			$this->log_file = fopen( '/var/log/mantis/jira.log', 'a' );
		}
		fwrite( $this->log_file, $p_text . "\n" );
		fflush( $this->log_file );
	}
}

// vim: set noet:
