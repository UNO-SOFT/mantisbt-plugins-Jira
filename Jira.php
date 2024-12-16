<?php
# :vim set noet:

if ( !defined( 'MANTIS_DIR' ) ) {
	define( 'MANTIS_DIR', dirname(__FILE__) . '/../..' );
}
if ( !defined( 'MANTIS_CORE' ) ) {
	define( 'MANTIS_CORE', MANTIS_DIR . '/core' );
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
			'password' => plugin_config_get( 'password', '' ),
            'key_regexp' => plugin_config_get( 'key_regexp', '^(INCIDENT|CHANGE|REQUEST|PROBLEM)-[0-9]+$' ),
		);
	}

	function hooks() {
		return array(
			'EVENT_BUGNOTE_ADD' => 'bugnote_add',
			//'EVENT_BUGNOTE_EDIT' => 'bugnote_edit',
			'EVENT_UPDATE_BUG' => 'bug_update',
			'EVENT_MENU_MANAGE' => 'menu_manage',
		);
	}

	function menu_manage( ) {
			if ( access_get_project_level() >= MANAGER) {
					return array( '<a href="' . plugin_page( 'config.php' ) . '">'
							.  plugin_lang_get('config') . '</a>', );
			}
	}

	function bug_update( $p_event, $p_old, &$p_new ) {
		if( $p_old->status == $p_new->status || $p_old->status >= 80 ) {
			return;
		}
		$t_target_status_id = '';
		$t_tran_id = 0;
		// „Folyamatban”  „Átadva” tranzíció  esetében a 21 –es numerikus érték. 
		/*
		Mantis státusz elnevezés	Mantis státuszkód
		új	10
		folyamatban	50
		kérdés	55
		átadva	80
		*/
			// $t_target_status_id = 'NEW';  // NEW
		if( $p_new->status >= 90 ) {
			$t_target_status_id = 'CLOSED';  // CLOSED
		} elseif( $p_new->status >= 80 ) {
			$t_target_status_id = 'RESOLVED';  // RESOLVED
		} elseif( $p_new->status = 50 ) {
			$t_target_status_id = 'IN_PROGRESS';  // INPROGRESS
		} elseif( $p_new->status = 55 ) {
			$t_target_status_id = 'ON_HOLD';  // ONHOLD
		}
		/*
		if( $p_new=>status >= 90 ) {
			$t_tran_id = 21;
			if( $p_old->status > 50 ) {
				$t_tran_id = 61;
			}
		} elseif( $p_new->status >= 80 ) { // átadva
			$t_tran_id = 21;
		// „Új”  „Folyamatban” státuszváltás esetében a 11 –es numerikus érték.
		} elseif( $p_new->status == 50 ) { // folyamatban
			$t_tran_id = 11;
		// „On hold”  „In progress”	51
			if( $p_old->status > 50 ) {
				$t_tran_id = 51;
			}

		}
		*/
		$t_issueid = this->issueid_get( $p_bug_id );
		if( !$t_issueid ) {
			return;
		}

		if( $t_tran_id != 0 ) {
			$this->call("issue", array(
				"transition",
				$t_issueid,
				$t_tran_id ) 
			);
		} elseif ( $t_target_status_id ) {
			$this->call("issue", array(
				"transition-to",
				$t_issueid,
				$t_target_status_id ) 
			);
		}
	}

	function issueid_get( $p_bug_id ) : string {
		$this->log( 'bugnote_add(' . $p_event_name . ', ' . $p_bug_id . ' bugnote_id=' . $p_bugnote_id . ' files=' . var_export( $p_files, TRUE ) . ')' );
		if( $this->issueid_field_id === 0 ) {
			$this->issueid_field_id = custom_field_id_from_name( 'nyilvszám' );
		}
		if( $this->skip_reporter_id === 0 ) {
			$this->skip_reporter_id = user_get_id_by_name( 'jira' );
		}
		$t_issueid = custom_field_get_value( $this->issueid_field_id, $p_bug_id );
        $t_pattern = '/' . plugin_config_get( 'key_regexp', '^(INCIDENT|CHANGE|REQUEST|PROBLEM)-[0-9]+$' ) . '/';
		$this->log( 'nyilvszam(' . $this->issueid_field_id . '): ' . $t_issueid . ", pat=$t_pattern match=" . preg_match( $t_pattern, $t_issueid ) );
		if( !$t_issueid || !preg_match( $t_pattern, $t_issueid ) ) {
			return "";
		}
		return $t_issueid;
	}

	function bugnote_add( $p_event_name, $p_bug_id, $p_bugnote_id, $files ) {
		$p_files = $files;
		$t_issueid = this->issueid_get( $p_bug_id );
		if( !$t_issueid ) {
			return;
		}

		// $t_mantis_id = trim(
		// 	$this->call("issue", array( "mantisID", $t_issueid ) )[1]
		// );
		// if( $t_mantis_id != $p_bug_id ) {
		// 	$this->log("mantisID=$t_mantis_id bugID=$p_bug_id");
		// 	return;
		// }

		$t_bugnote = null;
		if( $p_bugnote_id ) {
			$t_bugnote = bugnote_get( $p_bugnote_id );
		}
		if( $t_bugnote ) {
			$this->log( 'bugnote ' . $t_bugnote->view_state );
			if( VS_PUBLIC != $t_bugnote->view_state ) {
				return;
			}

//TODO: ha kérdésre jön megjegyzés, akkor át kell állítani folyamatban-ra, ÉS ezt elküldeni a Jira-nak is!
			
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
				$this->call("comment", array( 
					"--mantisid=" . $p_bug_id,
					$t_issueid, $t_bugnote->note . "\n\n<<" . user_get_realname( $t_bugnote->reporter_id ) . '>>' ) );
$this->log( 'comment added' );
			}
		}
		if( count( $p_files ) == 0 ) {
			return;
		}

		$t_project_id = (int)(bug_get_field( $p_bug_id, 'project_id' ));
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
			$this->call( "attach", array(
				"--mantisid=" . $p_bug_id,
				"--filename=" . $t_file['name'], 
				$t_issueid, 
				$t_local_disk_file,
			) );
		}
	}

	function call( $p_subcommand, $p_args ) {
		$t_conf = $this->config();
		$t_args = array( '/usr/local/bin/mantisbt-jira' );
		foreach( $t_conf as $k => $v ) {
			if( $v && !strstr( $k, '_' ) ) {
				$t_args[] = escapeshellarg( '--jira-' . $k . '=' . $v );
			}
		}
		$t_args[] = '--queues=/var/local/mantis/jira';
		
		$t_output = array();
		$t_args = implode( ' ', $t_args ) . ' ' . escapeshellarg( $p_subcommand );
		foreach( $p_args as $t_arg ) {
			$t_args .= ' ' . escapeshellarg( $t_arg );
		}
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
		$t_stdout = stream_get_contents( $t_pipes[1] );
		fclose( $t_pipes[1] );
		$t_stderr = stream_get_contents( $t_pipes[2] );
		fclose( $t_pipes[2] );
		$t_rc = proc_close( $t_process );
		$this->log('got ' . $t_rc . ': stderr=' . var_export( $t_stderr, TRUE ) );
		return array( $t_rc, $t_stdout );
	}

	function log( $p_text ) {
		if( !$this->log_file ) {
			$this->log_file = fopen( '/var/log/mantis/jira' . (SYS_FLAVOR == 'dev' ? '-dev' : '') . '.log', 'a' );
		}
		fwrite( $this->log_file, $p_text . "\n" );
		fflush( $this->log_file );
	}
}

// vim: set noet:
