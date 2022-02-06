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
require_api( 'bugnote_api.php' );
require_api( 'file_api.php' );

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

	function update_bug( $p_event_name, $p_bug_id ) {
		//$conf = $this->config();
		//$iss = create_issue_service($conf['jira_host'], $conf['jira_token']);
	}

	function bugnote_add( $p_event_name, $p_bug_id ) {
		$t_bugnote_id = bugnote_get_latest_id( $p_bug_id );
		$t_bugnote = bugnote_get( $t_bugnote_id );
		if( VS_PUBLIC == $t_bugnote->view_state ) {
			$conf = $this->config();
			$iss = create_issue_service($conf['jira_host'], $conf['jira_token']);
			if( strlen($t_bugnote->note) !== 0 ) {
				$iss->addComment($issueKey, jira_new_comment($t_bugnote->note));
			}
			$t_tempdir = sys_get_temp_dir();
			$t_attachments = file_get_visible_attachments( $p_bug_id );
			$t_files = array();
			foreach( $t_attachments as $t_file ) {
				if( $t_file['download_url'] && $t_file['diskfile'] && $t_file['bugnote_id'] == $t_bugnote_id ) {
					$t_bn = basename($t_file['display_name']);
					$t_ext = strrchr($t_bn, '.');
					if( $t_ext ) {
						$t_bn = substr( $t_bn, 0, -strlen($t_ext) );
					} else {
						$t_ext = '';
					}
					$t_files[] = secure_named_symlink('', $t_file['diskfile'], $t_file['display_name']);
				}
			}
			if( count($t_files) != 0 ) {
				$iss->addAttachment($issueKey, jira_new_attachments($t_files));
				foreach( $t_files as $t_file ) {
					if( file_exists($t_file) && filetype($t_file) == 'link' ) {
						unlink($t_file);
					}
				}
			}
		}
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
