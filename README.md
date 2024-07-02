# Folyamatok szinkronizálása
https://docs.atlassian.com/software/jira/docs/api/REST/9.12.2/#api/2/issue-doTransition
GET /rest/api/2/issue/{issueIdOrKey}/transitions

Folyamatban -> (New->)in progress 
Megoldva    -> (In progress->)resolved

(resolved->)in progress -> (Megoldva->)Folyamatban
