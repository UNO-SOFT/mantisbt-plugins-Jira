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

class JiraPlugin extends MantisPlugin {
	private $issueid_field_id = 4;
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
			'EVENT_MENU_MANAGE' => 'menu_manage',
                );
        }

        function menu_manage( ) {

                if ( access_get_project_level() >= MANAGER) {
                        return array( '<a href="' . plugin_page( 'config.php' ) . '">'
                                .  plugin_lang_get('config') . '</a>', );
                }
        }

	function bugnote_add( $p_event_name, $p_bug_id ) {
		$this->log( 'bugnote_add(' . $p_event_name . ', ' . $p_bug_id . ')' );
		if ( $this->issueid_field_id === 0 ) {
			$this->issueid_field_id = custom_field_id_from_name( 'nyilvszám' );
		}
		$t_issueid = custom_field_get_value( $this->issueid_field_id, $p_bug_id );
		$this->log( 'nyilvszam(' . $this->issueid_field_id . '): ' . $t_issueid );
		if( !$t_issueid ) {
			return;
		}

		$t_bugnote_id = bugnote_get_latest_id( $p_bug_id );
		$t_bugnote = bugnote_get( $t_bugnote_id );
		$this->log( 'bugnote ' . $t_bugnote->view_state );
		if( VS_PUBLIC != $t_bugnote->view_state ) {
			return;
		}

		if( strlen($t_bugnote->note) !== 0 ) {
			$this->call("comment", $t_issueid, $t_bugnote->note);
		}
		$t_tempdir = sys_get_temp_dir();
		$t_attachments = file_get_visible_attachments( $p_bug_id );
		foreach( $t_attachments as $t_file ) {
			if( $t_file['download_url'] && $t_file['diskfile'] && $t_file['bugnote_id'] == $t_bugnote_id ) {
				$t_bn = basename($t_file['display_name']);
				$t_ext = strrchr($t_bn, '.');
				if( $t_ext ) {
					$t_bn = substr( $t_bn, 0, -strlen($t_ext) );
				} else {
					$t_ext = '';
				}
				$t_tmpfn = secure_named_symlink('', $t_file['diskfile'], $t_file['display_name']);
				$this->call( "attach", $t_issueid, $t_tmpfn );
				if( file_exists($t_tmpfn) && filetype($t_tmpfn) == 'link' ) {
					unlink($t_tmpfn);
				}
			}
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
		$t_process = proc_open( '/usr/local/bin/mantisbt-jira', 
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

function secure_named_symlink($dir, $target, $name) {
	if( !(isset($dir) && is_string($dir) && $dir) ) {
		$dir = sys_get_temp_dir();
	}
	$name = basename( $name );
	$postfix = strrchr($name, '.');
	if( $postfix ) {
		$name = substr($name, 0, -strlen($postfix));
	} else {
		$postfix = '';
	}

    // find a temporary name
    $tries = 1;
    do {
        // get a known, unique temporary file name
        $sysFileName = tempnam($dir, $prefix);
        if ($sysFileName === false) {
            return false;
        }

        // tack on the extension
        $newFileName = $sysFileName . $postfix;
		$ok = symlink( $target, $name . $postfix );
		unlink( $sysFileName );
		if( $ok ) {
			return $newFileName;
        }

        $tries++;
    } while ($tries <= 5);

    return false;
}

// vim: set noet:
