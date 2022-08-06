<?php
# MantisBT - a php based bugtracking system
# Copyright (C) 2002 - 2009  MantisBT Team - mantisbt-dev@lists.sourceforge.net
# MantisBT is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 2 of the License, or
# (at your option) any later version.
#
# MantisBT is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with MantisBT.  If not, see <http://www.gnu.org/licenses/>.

require_api( 'database_api.php' );
require_api( 'form_api.php' );
require_api( 'gpc_api.php' );
require_api( 'logging_api.php' );

form_security_validate( 'plugin_jira_edit' );

//auth_reauthenticate( );
access_ensure_global_level( plugin_config_get( 'edit_threshold', MANAGER ) );

$f_bug_id = gpc_get_int( 'bug_id' );

foreach ( $f_users as $i => $t_user_id) {
	contributors_set( $f_bug_id, $t_user_id, string_mul_100($f_hundred_cents[$i]) );
}
$f_new_user_id = gpc_get_int( 'new_user' );
$f_new_cents = string_mul_100(gpc_get_string( 'new_hundred_cents' ));
log_event( LOG_PLUGIN, "new_user=" . var_export( $f_new_user_id, TRUE ) . " new_cents=" . var_export( $f_new_ceents, TRUE ) );
if ( $f_new_user_id != 0 && $f_new_cents > 0 ) {
    contributors_set( $f_bug_id, $f_new_user_id, $f_new_cents );
}

form_security_purge( 'plugin_jira_edit' );

print_successful_redirect( 'view.php?id=' . $f_bug_id . '#jira' );
