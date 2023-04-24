// Copyright 2022, 2023 Tamás Gulácsi. All rights reserved.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/renameio"
	"github.com/klauspost/compress/gzhttp"

	"golang.org/x/exp/slog"
)

// https://partnerapi-uat.aegon.hu/partner/v1/ticket/update/openapi.json
type Jira struct {
	URL        *url.URL
	tokens     map[string]Token
	HTTPClient *http.Client
	token      Token
	tokensFile string
}

type JIRAIssueType struct {
	Self        string `json:"self"`
	ID          string `json:"id"`
	Description string `json:"description"`
	IconURL     string `json:"iconUrl"`
	Name        string `json:"name"`
	Subtask     bool   `json:"subtask"`
	AvatarID    int    `json:"avatarId"`
}

// https://mholt.github.io/json-to-go/
type JIRAIssue struct {
	Fields struct {
		Timetracking struct {
		} `json:"timetracking"`
		Customfield14512              interface{} `json:"customfield_14512"`
		Customfield10004              interface{} `json:"customfield_10004"`
		Aggregatetimeoriginalestimate interface{} `json:"aggregatetimeoriginalestimate"`
		Customfield15407              interface{} `json:"customfield_15407"`
		Customfield14439              interface{} `json:"customfield_14439"`
		Customfield10600              interface{} `json:"customfield_10600"`
		Customfield14318              interface{} `json:"customfield_14318"`
		Customfield11800              interface{} `json:"customfield_11800"`
		Customfield15406              interface{} `json:"customfield_15406"`
		Customfield14438              interface{} `json:"customfield_14438"`
		Customfield14446              interface{} `json:"customfield_14446"`
		Customfield15150              interface{} `json:"customfield_15150"`
		Customfield15151              interface{} `json:"customfield_15151"`
		Customfield15154              interface{} `json:"customfield_15154"`
		Customfield15152              interface{} `json:"customfield_15152"`
		Customfield15153              interface{} `json:"customfield_15153"`
		Customfield15156              interface{} `json:"customfield_15156"`
		Customfield11901              interface{} `json:"customfield_11901"`
		Customfield15140              interface{} `json:"customfield_15140"`
		Resolution                    interface{} `json:"resolution"`
		Customfield15144              interface{} `json:"customfield_15144"`
		Customfield15141              interface{} `json:"customfield_15141"`
		Customfield15142              interface{} `json:"customfield_15142"`
		Customfield15147              interface{} `json:"customfield_15147"`
		Customfield15148              interface{} `json:"customfield_15148"`
		Customfield15145              interface{} `json:"customfield_15145"`
		Customfield15146              interface{} `json:"customfield_15146"`
		Customfield15149              interface{} `json:"customfield_15149"`
		Customfield10500              interface{} `json:"customfield_10500"`
		Customfield15132              interface{} `json:"customfield_15132"`
		Customfield15133              interface{} `json:"customfield_15133"`
		Customfield15130              interface{} `json:"customfield_15130"`
		Customfield15131              interface{} `json:"customfield_15131"`
		Customfield15136              interface{} `json:"customfield_15136"`
		Customfield15137              interface{} `json:"customfield_15137"`
		Customfield15134              interface{} `json:"customfield_15134"`
		Customfield15135              interface{} `json:"customfield_15135"`
		Customfield15138              interface{} `json:"customfield_15138"`
		Customfield15139              interface{} `json:"customfield_15139"`
		Environment                   interface{} `json:"environment"`
		Duedate                       interface{} `json:"duedate"`
		Customfield10104              interface{} `json:"customfield_10104"`
		Customfield10105              interface{} `json:"customfield_10105"`
		Customfield14823              interface{} `json:"customfield_14823"`
		Customfield12405              interface{} `json:"customfield_12405"`
		Customfield14824              interface{} `json:"customfield_14824"`
		Customfield12407              interface{} `json:"customfield_12407"`
		Customfield10100              interface{} `json:"customfield_10100"`
		Customfield14700              interface{} `json:"customfield_14700"`
		Customfield14821              interface{} `json:"customfield_14821"`
		Customfield10101              interface{} `json:"customfield_10101"`
		Customfield14822              interface{} `json:"customfield_14822"`
		Customfield10102              interface{} `json:"customfield_10102"`
		Customfield12401              interface{} `json:"customfield_12401"`
		Customfield14820              interface{} `json:"customfield_14820"`
		Customfield14814              interface{} `json:"customfield_14814"`
		Customfield14815              interface{} `json:"customfield_14815"`
		Customfield14812              interface{} `json:"customfield_14812"`
		Customfield14813              interface{} `json:"customfield_14813"`
		Customfield14818              interface{} `json:"customfield_14818"`
		Timeestimate                  interface{} `json:"timeestimate"`
		Customfield14819              interface{} `json:"customfield_14819"`
		Customfield14816              interface{} `json:"customfield_14816"`
		Customfield14817              interface{} `json:"customfield_14817"`
		Customfield14324              interface{} `json:"customfield_14324"`
		Customfield14810              interface{} `json:"customfield_14810"`
		Customfield14811              interface{} `json:"customfield_14811"`
		Customfield11301              interface{} `json:"customfield_11301"`
		Customfield11302              interface{} `json:"customfield_11302"`
		Customfield14803              interface{} `json:"customfield_14803"`
		Customfield14804              interface{} `json:"customfield_14804"`
		Customfield14801              interface{} `json:"customfield_14801"`
		Customfield14802              interface{} `json:"customfield_14802"`
		Customfield14807              interface{} `json:"customfield_14807"`
		Aggregatetimeestimate         interface{} `json:"aggregatetimeestimate"`
		Customfield14808              interface{} `json:"customfield_14808"`
		Customfield14805              interface{} `json:"customfield_14805"`
		Customfield14806              interface{} `json:"customfield_14806"`
		Customfield14809              interface{} `json:"customfield_14809"`
		Customfield14448              interface{} `json:"customfield_14448"`
		Customfield14800              interface{} `json:"customfield_14800"`
		Customfield12615              interface{} `json:"customfield_12615"`
		Timespent                     interface{} `json:"timespent"`
		Aggregatetimespent            interface{} `json:"aggregatetimespent"`
		Customfield11401              interface{} `json:"customfield_11401"`
		Customfield11400              interface{} `json:"customfield_11400"`
		Customfield14902              interface{} `json:"customfield_14902"`
		Customfield14903              interface{} `json:"customfield_14903"`
		Customfield14900              interface{} `json:"customfield_14900"`
		Customfield14901              interface{} `json:"customfield_14901"`
		Customfield14904              interface{} `json:"customfield_14904"`
		Customfield14327              interface{} `json:"customfield_14327"`
		Customfield14447              interface{} `json:"customfield_14447"`
		Customfield10301              interface{} `json:"customfield_10301"`
		Customfield12712              interface{} `json:"customfield_12712"`
		Customfield13801              interface{} `json:"customfield_13801"`
		Customfield12711              interface{} `json:"customfield_12711"`
		Customfield13803              interface{} `json:"customfield_13803"`
		Customfield13802              interface{} `json:"customfield_13802"`
		Customfield13804              interface{} `json:"customfield_13804"`
		Customfield11500              interface{} `json:"customfield_11500"`
		Customfield12710              interface{} `json:"customfield_12710"`
		Customfield12705              interface{} `json:"customfield_12705"`
		Customfield12704              interface{} `json:"customfield_12704"`
		Customfield12707              interface{} `json:"customfield_12707"`
		Customfield12706              interface{} `json:"customfield_12706"`
		Customfield12709              interface{} `json:"customfield_12709"`
		Customfield12708              interface{} `json:"customfield_12708"`
		Customfield12811              interface{} `json:"customfield_12811"`
		Customfield12810              interface{} `json:"customfield_12810"`
		Customfield14311              interface{} `json:"customfield_14311"`
		Customfield14432              interface{} `json:"customfield_14432"`
		Customfield14433              interface{} `json:"customfield_14433"`
		Customfield15401              interface{} `json:"customfield_15401"`
		Customfield14430              interface{} `json:"customfield_14430"`
		Customfield14310              interface{} `json:"customfield_14310"`
		Customfield14431              interface{} `json:"customfield_14431"`
		Customfield14315              interface{} `json:"customfield_14315"`
		Customfield14437              interface{} `json:"customfield_14437"`
		Customfield15402              interface{} `json:"customfield_15402"`
		Customfield14435              interface{} `json:"customfield_14435"`
		Customfield14308              interface{} `json:"customfield_14308"`
		Customfield14429              interface{} `json:"customfield_14429"`
		Customfield14309              interface{} `json:"customfield_14309"`
		Customfield14306              interface{} `json:"customfield_14306"`
		Customfield14427              interface{} `json:"customfield_14427"`
		Customfield14307              interface{} `json:"customfield_14307"`
		Customfield14428              interface{} `json:"customfield_14428"`
		Customfield14300              interface{} `json:"customfield_14300"`
		Customfield12000              interface{} `json:"customfield_12000"`
		Customfield14442              interface{} `json:"customfield_14442"`
		Customfield14301              interface{} `json:"customfield_14301"`
		Customfield14422              interface{} `json:"customfield_14422"`
		Customfield14420              interface{} `json:"customfield_14420"`
		Customfield14304              interface{} `json:"customfield_14304"`
		Customfield14200              interface{} `json:"customfield_14200"`
		Customfield14305              interface{} `json:"customfield_14305"`
		Customfield14426              interface{} `json:"customfield_14426"`
		Customfield14302              interface{} `json:"customfield_14302"`
		Customfield14444              interface{} `json:"customfield_14444"`
		Customfield14323              interface{} `json:"customfield_14323"`
		Customfield14303              interface{} `json:"customfield_14303"`
		Customfield14424              interface{} `json:"customfield_14424"`
		Customfield14418              interface{} `json:"customfield_14418"`
		Customfield14419              interface{} `json:"customfield_14419"`
		Customfield14416              interface{} `json:"customfield_14416"`
		Customfield14443              interface{} `json:"customfield_14443"`
		Customfield14410              interface{} `json:"customfield_14410"`
		Customfield14411              interface{} `json:"customfield_14411"`
		Customfield14414              interface{} `json:"customfield_14414"`
		Customfield14415              interface{} `json:"customfield_14415"`
		Customfield14412              interface{} `json:"customfield_14412"`
		Customfield14413              interface{} `json:"customfield_14413"`
		Customfield10049              interface{} `json:"customfield_10049"`
		Customfield14407              interface{} `json:"customfield_14407"`
		Customfield14322              interface{} `json:"customfield_14322"`
		Customfield14405              interface{} `json:"customfield_14405"`
		Customfield14406              interface{} `json:"customfield_14406"`
		Customfield14409              interface{} `json:"customfield_14409"`
		Customfield10040              interface{} `json:"customfield_10040"`
		Customfield10041              interface{} `json:"customfield_10041"`
		Customfield10042              interface{} `json:"customfield_10042"`
		Customfield14400              interface{} `json:"customfield_14400"`
		Customfield14440              interface{} `json:"customfield_14440"`
		Customfield10043              interface{} `json:"customfield_10043"`
		Customfield10044              interface{} `json:"customfield_10044"`
		Customfield14640              interface{} `json:"customfield_14640"`
		Customfield10045              interface{} `json:"customfield_10045"`
		Customfield14403              interface{} `json:"customfield_14403"`
		Customfield10046              interface{} `json:"customfield_10046"`
		Customfield14328              interface{} `json:"customfield_14328"`
		Customfield10047              interface{} `json:"customfield_10047"`
		Customfield14401              interface{} `json:"customfield_14401"`
		Customfield10048              interface{} `json:"customfield_10048"`
		Customfield14402              interface{} `json:"customfield_14402"`
		Customfield10038              interface{} `json:"customfield_10038"`
		Customfield14638              interface{} `json:"customfield_14638"`
		Customfield10039              interface{} `json:"customfield_10039"`
		Customfield14639              interface{} `json:"customfield_14639"`
		Customfield14636              interface{} `json:"customfield_14636"`
		Customfield14637              interface{} `json:"customfield_14637"`
		Customfield14457              interface{} `json:"customfield_14457"`
		Customfield10030              interface{} `json:"customfield_10030"`
		Customfield14630              interface{} `json:"customfield_14630"`
		Customfield10031              interface{} `json:"customfield_10031"`
		Customfield14631              interface{} `json:"customfield_14631"`
		Customfield14510              interface{} `json:"customfield_14510"`
		Customfield15304              interface{} `json:"customfield_15304"`
		Customfield10032              interface{} `json:"customfield_10032"`
		Customfield10033              interface{} `json:"customfield_10033"`
		Customfield10034              interface{} `json:"customfield_10034"`
		Customfield14634              interface{} `json:"customfield_14634"`
		Customfield14513              interface{} `json:"customfield_14513"`
		Customfield10035              interface{} `json:"customfield_10035"`
		Customfield14635              interface{} `json:"customfield_14635"`
		Customfield10036              interface{} `json:"customfield_10036"`
		Customfield14632              interface{} `json:"customfield_14632"`
		Customfield14511              interface{} `json:"customfield_14511"`
		Customfield14464              interface{} `json:"customfield_14464"`
		Customfield14456              interface{} `json:"customfield_14456"`
		Customfield15303              interface{} `json:"customfield_15303"`
		Customfield14627              interface{} `json:"customfield_14627"`
		Customfield14506              interface{} `json:"customfield_14506"`
		Customfield10028              interface{} `json:"customfield_10028"`
		Customfield14507              interface{} `json:"customfield_14507"`
		Customfield14628              interface{} `json:"customfield_14628"`
		Customfield10029              interface{} `json:"customfield_10029"`
		Customfield14504              interface{} `json:"customfield_14504"`
		Customfield14625              interface{} `json:"customfield_14625"`
		Customfield14626              interface{} `json:"customfield_14626"`
		Customfield14505              interface{} `json:"customfield_14505"`
		Customfield14508              interface{} `json:"customfield_14508"`
		Customfield14629              interface{} `json:"customfield_14629"`
		Resolutiondate                interface{} `json:"resolutiondate"`
		Customfield14509              interface{} `json:"customfield_14509"`
		Customfield14459              interface{} `json:"customfield_14459"`
		Customfield14620              interface{} `json:"customfield_14620"`
		Customfield14502              interface{} `json:"customfield_14502"`
		Customfield14624              interface{} `json:"customfield_14624"`
		Customfield14503              interface{} `json:"customfield_14503"`
		Customfield14500              interface{} `json:"customfield_14500"`
		Customfield14621              interface{} `json:"customfield_14621"`
		Customfield14501              interface{} `json:"customfield_14501"`
		Customfield14616              interface{} `json:"customfield_14616"`
		Customfield10017              interface{} `json:"customfield_10017"`
		Customfield14614              interface{} `json:"customfield_14614"`
		Customfield14615              interface{} `json:"customfield_14615"`
		Customfield14618              interface{} `json:"customfield_14618"`
		Customfield14619              interface{} `json:"customfield_14619"`
		Customfield15306              interface{} `json:"customfield_15306"`
		Timeoriginalestimate          interface{} `json:"timeoriginalestimate"`
		Customfield14338              interface{} `json:"customfield_14338"`
		Customfield14458              interface{} `json:"customfield_14458"`
		Customfield10011              interface{} `json:"customfield_10011"`
		Customfield15305              interface{} `json:"customfield_15305"`
		Customfield14337              interface{} `json:"customfield_14337"`
		Customfield10012              interface{} `json:"customfield_10012"`
		Customfield14612              interface{} `json:"customfield_14612"`
		Customfield10013              interface{} `json:"customfield_10013"`
		Customfield11102              interface{} `json:"customfield_11102"`
		Customfield14613              interface{} `json:"customfield_14613"`
		Customfield11103              interface{} `json:"customfield_11103"`
		Customfield10014              interface{} `json:"customfield_10014"`
		Customfield14610              interface{} `json:"customfield_14610"`
		Customfield10015              interface{} `json:"customfield_10015"`
		Customfield14611              interface{} `json:"customfield_14611"`
		Customfield14453              interface{} `json:"customfield_14453"`
		Customfield14605              interface{} `json:"customfield_14605"`
		Customfield14606              interface{} `json:"customfield_14606"`
		Customfield10007              interface{} `json:"customfield_10007"`
		Customfield14603              interface{} `json:"customfield_14603"`
		Customfield14332              interface{} `json:"customfield_14332"`
		Customfield14604              interface{} `json:"customfield_14604"`
		Customfield14452              interface{} `json:"customfield_14452"`
		Customfield14609              interface{} `json:"customfield_14609"`
		Customfield14607              interface{} `json:"customfield_14607"`
		Customfield14608              interface{} `json:"customfield_14608"`
		Customfield14331              interface{} `json:"customfield_14331"`
		Customfield10000              interface{} `json:"customfield_10000"`
		Customfield14601              interface{} `json:"customfield_14601"`
		Customfield14602              interface{} `json:"customfield_14602"`
		Customfield14633              interface{} `json:"customfield_14633"`
		Customfield14600              interface{} `json:"customfield_14600"`
		Customfield13504              interface{} `json:"customfield_13504"`
		Customfield13505              interface{} `json:"customfield_13505"`
		Customfield14455              interface{} `json:"customfield_14455"`
		Customfield15121              interface{} `json:"customfield_15121"`
		Customfield15122              interface{} `json:"customfield_15122"`
		Customfield15120              interface{} `json:"customfield_15120"`
		Customfield15125              interface{} `json:"customfield_15125"`
		Customfield15126              interface{} `json:"customfield_15126"`
		Customfield15123              interface{} `json:"customfield_15123"`
		Customfield15302              interface{} `json:"customfield_15302"`
		Customfield15124              interface{} `json:"customfield_15124"`
		Customfield15127              interface{} `json:"customfield_15127"`
		Customfield15128              interface{} `json:"customfield_15128"`
		Customfield10900              interface{} `json:"customfield_10900"`
		Customfield15110              interface{} `json:"customfield_15110"`
		Customfield15111              interface{} `json:"customfield_15111"`
		Customfield13000              interface{} `json:"customfield_13000"`
		Customfield15115              interface{} `json:"customfield_15115"`
		Customfield15301              interface{} `json:"customfield_15301"`
		Customfield14454              interface{} `json:"customfield_14454"`
		Customfield14330              interface{} `json:"customfield_14330"`
		Customfield15118              interface{} `json:"customfield_15118"`
		Customfield15119              interface{} `json:"customfield_15119"`
		Customfield15116              interface{} `json:"customfield_15116"`
		Customfield15117              interface{} `json:"customfield_15117"`
		Customfield14450              interface{} `json:"customfield_14450"`
		Customfield15309              interface{} `json:"customfield_15309"`
		Customfield14491              interface{} `json:"customfield_14491"`
		Customfield14494              interface{} `json:"customfield_14494"`
		Customfield14495              interface{} `json:"customfield_14495"`
		Customfield15100              interface{} `json:"customfield_15100"`
		Customfield14492              interface{} `json:"customfield_14492"`
		Customfield14493              interface{} `json:"customfield_14493"`
		Customfield15103              interface{} `json:"customfield_15103"`
		Customfield14498              interface{} `json:"customfield_14498"`
		Customfield14499              interface{} `json:"customfield_14499"`
		Customfield14468              interface{} `json:"customfield_14468"`
		Customfield14496              interface{} `json:"customfield_14496"`
		Customfield15102              interface{} `json:"customfield_15102"`
		Customfield14497              interface{} `json:"customfield_14497"`
		Customfield15107              interface{} `json:"customfield_15107"`
		Customfield15108              interface{} `json:"customfield_15108"`
		Customfield15105              interface{} `json:"customfield_15105"`
		Customfield15106              interface{} `json:"customfield_15106"`
		Customfield14483              interface{} `json:"customfield_14483"`
		Customfield14484              interface{} `json:"customfield_14484"`
		Customfield14481              interface{} `json:"customfield_14481"`
		Customfield14482              interface{} `json:"customfield_14482"`
		Customfield15213              interface{} `json:"customfield_15213"`
		Customfield15214              interface{} `json:"customfield_15214"`
		Customfield15211              interface{} `json:"customfield_15211"`
		Customfield15212              interface{} `json:"customfield_15212"`
		Customfield14467              interface{} `json:"customfield_14467"`
		Customfield15314              interface{} `json:"customfield_15314"`
		Customfield15215              interface{} `json:"customfield_15215"`
		Customfield14469              interface{} `json:"customfield_14469"`
		Customfield14472              interface{} `json:"customfield_14472"`
		Customfield14470              interface{} `json:"customfield_14470"`
		Customfield14471              interface{} `json:"customfield_14471"`
		Customfield14476              interface{} `json:"customfield_14476"`
		Customfield14477              interface{} `json:"customfield_14477"`
		Customfield14475              interface{} `json:"customfield_14475"`
		Customfield15318              interface{} `json:"customfield_15318"`
		Customfield15319              interface{} `json:"customfield_15319"`
		Customfield14340              interface{} `json:"customfield_14340"`
		Customfield14461              interface{} `json:"customfield_14461"`
		Customfield14341              interface{} `json:"customfield_14341"`
		Customfield14462              interface{} `json:"customfield_14462"`
		Customfield10037              interface{} `json:"customfield_10037"`
		Customfield14102              interface{} `json:"customfield_14102"`
		Customfield15311              interface{} `json:"customfield_15311"`
		Customfield15312              interface{} `json:"customfield_15312"`
		Customfield14465              interface{} `json:"customfield_14465"`
		Customfield14466              interface{} `json:"customfield_14466"`
		Customfield14100              interface{} `json:"customfield_14100"`
		Customfield14101              interface{} `json:"customfield_14101"`
		Customfield14463              interface{} `json:"customfield_14463"`
		Customfield11100              struct {
			OngoingCycle struct {
				GoalDuration struct {
					Friendly string `json:"friendly"`
					Millis   int    `json:"millis"`
				} `json:"goalDuration"`
				ElapsedTime struct {
					Friendly string `json:"friendly"`
					Millis   int    `json:"millis"`
				} `json:"elapsedTime"`
				RemainingTime struct {
					Friendly string `json:"friendly"`
					Millis   int    `json:"millis"`
				} `json:"remainingTime"`
				StartTime struct {
					Iso8601     string `json:"iso8601"`
					Jira        string `json:"jira"`
					Friendly    string `json:"friendly"`
					EpochMillis int64  `json:"epochMillis"`
				} `json:"startTime"`
				BreachTime struct {
					Iso8601     string `json:"iso8601"`
					Jira        string `json:"jira"`
					Friendly    string `json:"friendly"`
					EpochMillis int64  `json:"epochMillis"`
				} `json:"breachTime"`
				Breached            bool `json:"breached"`
				Paused              bool `json:"paused"`
				WithinCalendarHours bool `json:"withinCalendarHours"`
			} `json:"ongoingCycle"`
			ID    string `json:"id"`
			Name  string `json:"name"`
			Links struct {
				Self string `json:"self"`
			} `json:"_links"`
			CompletedCycles []interface{} `json:"completedCycles"`
		} `json:"customfield_11100"`
		Status struct {
			Self           string `json:"self"`
			Description    string `json:"description"`
			IconURL        string `json:"iconUrl"`
			Name           string `json:"name"`
			ID             string `json:"id"`
			StatusCategory struct {
				Self      string `json:"self"`
				Key       string `json:"key"`
				ColorName string `json:"colorName"`
				Name      string `json:"name"`
				ID        int    `json:"id"`
			} `json:"statusCategory"`
		} `json:"status"`
		Customfield14326 JIRAUser `json:"customfield_14326"`
		Assignee         JIRAUser `json:"assignee"`
		Reporter         JIRAUser `json:"reporter"`
		Creator          JIRAUser `json:"creator"`
		Project          struct {
			Self           string `json:"self"`
			ID             string `json:"id"`
			Key            string `json:"key"`
			Name           string `json:"name"`
			ProjectTypeKey string `json:"projectTypeKey"`
		} `json:"project"`
		Security struct {
			Self        string `json:"self"`
			ID          string `json:"id"`
			Description string `json:"description"`
			Name        string `json:"name"`
		} `json:"security"`
		Priority struct {
			Self    string `json:"self"`
			IconURL string `json:"iconUrl"`
			Name    string `json:"name"`
			ID      string `json:"id"`
		} `json:"priority"`
		Customfield14334 string `json:"customfield_14334"`
		Summary          string `json:"summary"`
		Customfield14421 string `json:"customfield_14421"`
		Description      string `json:"description"`
		LastViewed       string `json:"lastViewed"`
		Customfield15112 string `json:"customfield_15112"`
		Updated          string `json:"updated"`
		Customfield10005 string `json:"customfield_10005"`
		Created          string `json:"created"`
		Customfield14335 string `json:"customfield_14335"`
		Customfield10300 string `json:"customfield_10300"`
		Customfield10009 struct {
			Links struct {
				JiraRest string `json:"jiraRest"`
				Web      string `json:"web"`
				Self     string `json:"self"`
			} `json:"_links"`
			RequestType struct {
				ID    string `json:"id"`
				Links struct {
					Self string `json:"self"`
				} `json:"_links"`
				Name          string   `json:"name"`
				Description   string   `json:"description"`
				HelpText      string   `json:"helpText"`
				ServiceDeskID string   `json:"serviceDeskId"`
				GroupIds      []string `json:"groupIds"`
			} `json:"requestType"`
			CurrentStatus struct {
				Status     string `json:"status"`
				StatusDate struct {
					Iso8601     string `json:"iso8601"`
					Jira        string `json:"jira"`
					Friendly    string `json:"friendly"`
					EpochMillis int64  `json:"epochMillis"`
				} `json:"statusDate"`
			} `json:"currentStatus"`
		} `json:"customfield_10009"`
		Customfield14342 struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Links struct {
				Self string `json:"self"`
			} `json:"_links"`
			CompletedCycles []interface{} `json:"completedCycles"`
		} `json:"customfield_14342"`
		Customfield14344 struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Links struct {
				Self string `json:"self"`
			} `json:"_links"`
			CompletedCycles []interface{} `json:"completedCycles"`
		} `json:"customfield_14344"`
		Customfield11101 struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Links struct {
				Self string `json:"self"`
			} `json:"_links"`
			CompletedCycles []struct {
				GoalDuration struct {
					Friendly string `json:"friendly"`
					Millis   int    `json:"millis"`
				} `json:"goalDuration"`
				ElapsedTime struct {
					Friendly string `json:"friendly"`
					Millis   int    `json:"millis"`
				} `json:"elapsedTime"`
				RemainingTime struct {
					Friendly string `json:"friendly"`
					Millis   int    `json:"millis"`
				} `json:"remainingTime"`
				StartTime struct {
					Iso8601     string `json:"iso8601"`
					Jira        string `json:"jira"`
					Friendly    string `json:"friendly"`
					EpochMillis int64  `json:"epochMillis"`
				} `json:"startTime"`
				StopTime struct {
					Iso8601     string `json:"iso8601"`
					Jira        string `json:"jira"`
					Friendly    string `json:"friendly"`
					EpochMillis int64  `json:"epochMillis"`
				} `json:"stopTime"`
				Breached bool `json:"breached"`
			} `json:"completedCycles"`
		} `json:"customfield_11101"`
		Customfield14343 struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Links struct {
				Self string `json:"self"`
			} `json:"_links"`
			CompletedCycles []interface{} `json:"completedCycles"`
		} `json:"customfield_14343"`
		Customfield15113 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_15113"`
		Customfield14451 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14451"`
		Customfield15109 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_15109"`
		Customfield14408 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14408"`
		Customfield14321 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14321"`
		Customfield15143 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_15143"`
		Customfield14325 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14325"`
		Customfield15104 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_15104"`
		Customfield15114 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_15114"`
		Customfield14339 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14339"`
		Customfield14423 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14423"`
		Customfield14404 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14404"`
		Customfield14449 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14449"`
		Customfield14425 struct {
			Self     string `json:"self"`
			Value    string `json:"value"`
			ID       string `json:"id"`
			Disabled bool   `json:"disabled"`
		} `json:"customfield_14425"`
		Worklog struct {
			Worklogs   []interface{} `json:"worklogs"`
			StartAt    int           `json:"startAt"`
			MaxResults int           `json:"maxResults"`
			Total      int           `json:"total"`
		} `json:"worklog"`
		Versions         []interface{} `json:"versions"`
		Components       []interface{} `json:"components"`
		Labels           []interface{} `json:"labels"`
		Customfield10008 []JIRAUser    `json:"customfield_10008"`
		Subtasks         []struct {
			ID     string `json:"id"`
			Key    string `json:"key"`
			Self   string `json:"self"`
			Fields struct {
				Summary string `json:"summary"`
				Status  struct {
					Self           string `json:"self"`
					Description    string `json:"description"`
					IconURL        string `json:"iconUrl"`
					Name           string `json:"name"`
					ID             string `json:"id"`
					StatusCategory struct {
						Self      string `json:"self"`
						Key       string `json:"key"`
						ColorName string `json:"colorName"`
						Name      string `json:"name"`
						ID        int    `json:"id"`
					} `json:"statusCategory"`
				} `json:"status"`
				Priority struct {
					Self    string `json:"self"`
					IconURL string `json:"iconUrl"`
					Name    string `json:"name"`
					ID      string `json:"id"`
				} `json:"priority"`
				IssueType JIRAIssueType `json:"issuetype"`
			} `json:"fields"`
		} `json:"subtasks"`
		Attachment       []interface{} `json:"attachment"`
		Customfield10010 []interface{} `json:"customfield_10010"`
		Customfield15216 []struct {
			Active bool `json:"active"`
		} `json:"customfield_15216"`
		Customfield14336 []struct {
			Name string `json:"name"`
			Self string `json:"self"`
		} `json:"customfield_14336"`
		Issuelinks       []interface{} `json:"issuelinks"`
		FixVersions      []interface{} `json:"fixVersions"`
		Customfield15217 []struct {
			Active bool `json:"active"`
		} `json:"customfield_15217"`
		IssueType JIRAIssueType `json:"issuetype"`
		Watches   struct {
			Self       string `json:"self"`
			WatchCount int    `json:"watchCount"`
			IsWatching bool   `json:"isWatching"`
		} `json:"watches"`
		Votes struct {
			Self     string `json:"self"`
			Votes    int    `json:"votes"`
			HasVoted bool   `json:"hasVoted"`
		} `json:"votes"`
		Comment struct {
			Comments   []interface{} `json:"comments"`
			MaxResults int           `json:"maxResults"`
			Total      int           `json:"total"`
			StartAt    int           `json:"startAt"`
		} `json:"comment"`
		Progress struct {
			Progress int `json:"progress"`
			Total    int `json:"total"`
		} `json:"progress"`
		Aggregateprogress struct {
			Progress int `json:"progress"`
			Total    int `json:"total"`
		} `json:"aggregateprogress"`
		Workratio int `json:"workratio"`
	} `json:"fields"`
	Expand string `json:"expand"`
	ID     string `json:"id"`
	Self   string `json:"self"`
	Key    string `json:"key"`
}

func (svc *Jira) IssueGet(ctx context.Context, issueID string, fields []string) (JIRAIssue, error) {
	URL := svc.URLFor("issue", issueID, "")
	if len(fields) != 0 {
		q := URL.Query()
		q["fields"] = fields
		URL.RawQuery = q.Encode()
	}
	var issue JIRAIssue
	req, err := svc.NewRequest(ctx, "GET", URL, nil)
	if err != nil {
		return issue, err
	}
	resp, err := svc.Do(ctx, req)
	logger.Info("IssueGet do", "resp", resp, "error", err)
	if err != nil {
		return issue, err
	}
	err = json.Unmarshal(resp, &issue)
	return issue, err
}
func (svc *Jira) IssuePut(ctx context.Context, issue JIRAIssue) error {
	b, err := json.Marshal(issue)
	if err != nil {
		return err
	}
	req, err := svc.NewRequest(ctx, "PUT", svc.URLFor("issue", issue.ID, ""), b)
	if err != nil {
		return err
	}
	resp, err := svc.Do(ctx, req)
	logger.Info("IssuePut", "resp", resp, "error", err)
	return err
}
func (svc *Jira) Load(tokensFile, jiraUser, jiraPassword string) {
	svc.token.Username, svc.token.Password = jiraUser, jiraPassword
	svc.token.AuthURL = svc.URL.JoinPath("auth").String()
	if tokensFile == "" {
		return
	}
	svc.tokensFile = tokensFile
	fh, err := os.Open(tokensFile)
	if err != nil {
		logger.Error("open", "file", tokensFile, "error", err)
		return
	}
	var m map[string]Token
	err = json.NewDecoder(fh).Decode(&m)
	fh.Close()
	if err == nil {
		svc.tokens = make(map[string]Token, len(m))
		for k, v := range m {
			if v.IsValid() {
				svc.tokens[k] = v
			}
		}
		old := svc.token
		svc.token = svc.tokens[redactedURL(svc.URL)]
		if old.Username != "" {
			svc.token.Username, svc.token.Password = old.Username, old.Password
		}
		if svc.token.AuthURL == "" {
			svc.token.AuthURL = old.AuthURL
		}
		return
	}
	if err != nil {
		logger.Error("parse", "file", fh.Name(), "error", err)
	} else {
		logger.Info("not valid", "file", fh.Name())
	}
	_ = os.Remove(fh.Name())
}
func (svc *Jira) URLFor(typ, id, action string) *url.URL {
	URL := svc.URL.JoinPath("/"+typ, url.PathEscape(id))
	if action != "" {
		URL = URL.JoinPath(action)
	}
	return URL
}
func (svc *Jira) NewRequest(ctx context.Context, method string, URL *url.URL, body []byte) (*http.Request, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, URL.String(), r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

type JIRAUser struct {
	Self         string `json:"self"`
	Name         string `json:"name"`
	Key          string `json:"key"`
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName"`
	TimeZone     string `json:"timeZone"`
	Active       bool   `json:"active"`
}

type JIRAVisibility struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
type JIRAComment struct {
	Self         string         `json:"self"`
	ID           string         `json:"id"`
	Author       JIRAUser       `json:"author"`
	Body         string         `json:"body"`
	UpdateAuthor JIRAUser       `json:"updateAuthor"`
	Created      string         `json:"created"`
	Updated      string         `json:"updated"`
	Visibility   JIRAVisibility `json:"visibility"`
}

type getCommentsResp struct {
	Comments   []JIRAComment `json:"comments"`
	StartAt    int32         `json:"startAt"`
	MaxResults int32         `json:"maxResults"`
	Total      int32         `json:"total"`
}

func (svc *Jira) IssueComments(ctx context.Context, issueID string) ([]JIRAComment, error) {
	URL := svc.URLFor("issue", issueID, "comment")
	q := URL.Query()
	q.Set("startAt", "0")
	q.Set("maxResults", "65536")
	URL.RawQuery = q.Encode()
	req, err := svc.NewRequest(ctx, "GET", URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := svc.Do(ctx, req)
	logger.Info("IssueComments", "resp", resp, "error", err)
	if err != nil {
		return nil, err
	}
	var comments getCommentsResp
	err = json.Unmarshal(resp, &comments)
	return comments.Comments, err
}

type JSONCommentBody struct {
	Body string `json:"body"`
	//Visibility JIRAVisibility `json:"visibility"`
}

func (svc *Jira) IssueAddComment(ctx context.Context, issueID, body string) error {
	URL := svc.URLFor("issue", issueID, "comment")
	b, err := json.Marshal(JSONCommentBody{Body: body}) //, Visibility: JIRAVisibility{Type: "role", Value: "Administrators"}})
	if err != nil {
		return err
	}
	req, err := svc.NewRequest(ctx, "POST", URL, b)
	if err != nil {
		return err
	}
	resp, err := svc.Do(ctx, req)
	logger.Info("IssueAddComment", "resp", resp, "error", err)
	if err != nil {
		return err
	}
	var comment JIRAComment
	return json.Unmarshal(resp, &comment)
}

// IssueAddAttachment uploads the attachment to the issue.
func (svc *Jira) IssueAddAttachment(ctx context.Context, issueID, fileName, mimeType string, body io.Reader) error {
	// This resource expects a multipart post. The media-type multipart/form-data is defined in RFC 1867. Most client libraries have classes that make dealing with multipart posts simple. For instance, in Java the Apache HTTP Components library provides a MultiPartEntity that makes it simple to submit a multipart POST.
	//
	// In order to protect against XSRF attacks, because this method accepts multipart/form-data, it has XSRF protection on it. This means you must submit a header of X-Atlassian-Token: no-check with the request, otherwise it will be blocked.
	//
	// The name of the multipart/form-data parameter that contains attachments must be "file"
	//
	// curl -D- -u admin:admin -X POST -H "X-Atlassian-Token: no-check" -F "file=@myfile.txt" http://myhost/rest/api/2/issue/TEST-123/attachments
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	w, err := mw.CreateFormFile("file", fileName)
	if err != nil {
		return err
	}
	if _, err = io.Copy(w, body); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}
	URL := svc.URLFor("issue", issueID, "attachments")
	req, err := http.NewRequestWithContext(ctx, "POST", URL.String(), bytes.NewReader(buf.Bytes()))
	if err != nil {
		return err
	}
	req.Header.Set("X-Atlassian-Token", "no-check")
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := svc.Do(ctx, req)
	logger.Info("IssueAddAttachment", "resp", resp, "error", err)
	if err != nil {
		return err
	}
	attachments := make([]JIRAAttachment, 0, 1)
	return json.Unmarshal(resp, &attachments)
}

type JIRAAttachment struct {
	Author    JIRAUser `json:"author"`
	Self      string   `json:"self"`
	Filename  string   `json:"filename"`
	Created   string   `json:"created"`
	MimeType  string   `json:"mimeType"`
	Content   string   `json:"content"`
	Thumbnail string   `json:"thumbnail"`
	Size      int      `json:"size"`
}

func (svc *Jira) Do(ctx context.Context, req *http.Request) ([]byte, error) {
	b, changed, err := svc.token.do(ctx, svc.HTTPClient, req)
	if changed {
		if svc.tokens == nil {
			svc.tokens = make(map[string]Token)
		}
		svc.tokens[redactedURL(svc.URL)] = svc.token
		if svc.tokensFile != "" {
			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(svc.tokens); err != nil {
				logger.Error("marshal tokens", "error", err)
			} else if err := renameio.WriteFile(svc.tokensFile, buf.Bytes(), 0600); err != nil {
				logger.Error("write token", "file", svc.tokensFile, "error", err)
			}
		}
	}
	if err != nil {
		return b, err
	}
	return b, nil
}

type rawToken struct {
	JSessionID   string `json:"JSESSIONID"`
	AccessToken  string `json:"access_token"`
	IssuedAt     string `json:"issued_at"`
	ExpiresIn    string `json:"expires_in"`
	RefreshCount string `json:"refresh_count"`
	JIRAError
}

type Token struct {
	till time.Time
	rawToken
	AuthURL            string
	Username, Password string
	mu                 sync.Mutex
}

func (t *Token) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &t.rawToken); err != nil {
		return err
	}
	//logger.Debug("UnmarshalJSON", "b", string(b), "raw", fmt.Sprintf("%#v", t.rawToken))
	return t.init()
}
func (t *Token) init() error {
	logger.Debug("init", "raw", t.rawToken)
	if t.rawToken.JIRAError.IsValid() {
		return &t.rawToken.JIRAError
	}
	issuedAt, err := strconv.ParseInt(t.IssuedAt, 10, 64)
	if err != nil {
		return fmt.Errorf("parse issuedAt(%q): %w", t.IssuedAt, err)
	}
	expiresIn, err := strconv.ParseInt(t.ExpiresIn, 10, 64)
	if err != nil {
		return fmt.Errorf("parse expiresIn(%q): %w", t.ExpiresIn, err)
	}
	t.till = time.Unix(issuedAt/1000, issuedAt%1000).Add(time.Duration(expiresIn) * time.Second)
	logger.Debug("Unmarshal", "issuedAt", issuedAt, "expiresIn", expiresIn, "till", t.till)
	return nil
}
func (t *Token) IsValid() bool {
	return t != nil && t.JSessionID != "" && time.Now().Before(t.till)
}

type JIRAError struct {
	Code     string   `json:"ErrorCode,omitempty"`
	Message  string   `json:"Error,omitempty"`
	Fault    Fault    `json:"fault,omitempty"`
	Messages []string `json:"errorMessages,omitempty"`
}
type Fault struct {
	Code   string      `json:"faultstring,omitempty"`
	Detail FaultDetail `json:"detail,omitempty"`
}
type FaultDetail struct {
	Message string `json:"errorcode,omitempty"`
}
type userPass struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (je *JIRAError) Error() string {
	var buf strings.Builder
	if je.Code != "" {
		buf.WriteString(je.Code + ": " + je.Message)
	} else if je.Fault.Code != "" {
		buf.WriteString(je.Fault.Code + ": " + je.Fault.Detail.Message)
	}
	for _, m := range je.Messages {
		buf.WriteString("; ")
		buf.WriteString(m)
	}
	return buf.String()
}
func (je *JIRAError) IsValid() bool {
	return je != nil && (je.Code != "" || je.Fault.Code != "" || len(je.Messages) != 0)
}

func (t *Token) do(ctx context.Context, httpClient *http.Client, req *http.Request) ([]byte, bool, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
		if httpClient.Transport == nil {
			httpClient.Transport = http.DefaultTransport
		}
		httpClient.Transport = gzhttp.Transport(httpClient.Transport)
	}
	var buf bytes.Buffer
	t.mu.Lock()
	defer t.mu.Unlock()
	logger.Debug("IsValid", "token", t, "valid", t.IsValid())
	var changed bool
	if !t.IsValid() {
		if t.Username == "" || t.Password == "" || t.AuthURL == "" {
			return nil, changed, fmt.Errorf("empty JIRA username/password/AuthURL")
		}
		/*
		   curl --location --request POST 'https://partnerapi-uat.aegon.hu/partner/v1/ticket/update/auth?grant_type=password' \
		   --header 'Content-Type: application/json' \
		   --header 'Authorization: Basic ...' \
		   --data-raw '{ "username": "svc_unosoft", "password": "5h9RP97@qK6l"}'
		*/

		if err := json.NewEncoder(&buf).Encode(userPass{Username: t.Username, Password: t.Password}); err != nil {
			return nil, changed, err
		}
		req, err := http.NewRequestWithContext(ctx, "POST", t.AuthURL+"?grant_type=password", bytes.NewReader(buf.Bytes()))
		if err != nil {
			return nil, changed, err
		}
		logger.Debug("authenticate", "url", t.AuthURL, "body", buf.String())
		req.Header.Set("Content-Type", "application/json")
		start := time.Now()
		resp, err := httpClient.Do(req.WithContext(ctx))
		logger.Info("authenticate", "dur", time.Since(start).String(), "url", t.AuthURL, "error", err)
		if err != nil {
			return nil, changed, err
		}
		if resp == nil || resp.Body == nil {
			return nil, changed, fmt.Errorf("empty response")
		}
		buf.Reset()
		_, err = io.Copy(&buf, resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, changed, err
		}
		logger.Debug("authenticate", "response", buf.String())
		if err = json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&t); err != nil {
			return nil, changed, fmt.Errorf("decode %q: %w", buf.String(), err)
		}
		changed = true
		/*
		   answer:
		   {
		       "JSESSIONID": "1973D50D4C576BFBAA889B8726A2FF77",
		       "issued_at": "1658754363080",
		       "access_token": "iugVuMjlGng4Lwgdj3LbcE3ehGIB",
		       "expires_in": "7199",
		       "refresh_count": "0"
		   }
		*/
	}
	if req == nil {
		return nil, changed, nil
	}
	/*
	   2.
	   request:
	   curl --location --request POST 'https://partnerapi-uat.aegon.hu/partner/v1/ticket/update/issue' \
	   --header 'Content-Type: application/json' \
	   --header 'Cookie: JSESSIONID=...; TS0126a004=015d4139a83807c002e8dd16d46fa16563299b17c4a228ff33b64e12ada62f8cd7829575e919a595aefcd7736d6717351a163defa1; atlassian.xsrf.token=B0BO-X7QB-KBRG-M4RU_23574bc6e7a2f17160a6128c30ee1a58a7ec4eb5_lin' \
	   --header 'Authorization: Bearer ...' \
	*/
	req.Header.Set("Cookie", "JSESSIONID="+t.JSessionID)
	req.Header.Set("Authorization", "Bearer "+t.AccessToken)
	logEnabled := logger.Enabled(ctx, slog.LevelDebug)
	if logEnabled {
		b, err := httputil.DumpRequestOut(req, true)
		logger.Debug("Do", "request", string(b), "dumpErr", err)
		if err != nil {
			return nil, changed, err
		}
	}
	start := time.Now()
	resp, err := httpClient.Do(req.WithContext(ctx))
	logger.Info("do", "url", req.URL.String(), "method", req.Method, "dur", time.Since(start).String(), "hasBody", resp.Body != nil, "status", resp.Status)
	if err != nil {
		return nil, changed, err
	}
	if resp == nil {
		return nil, changed, fmt.Errorf("empty response")
	}
	if logEnabled {
		b, err := httputil.DumpResponse(resp, true)
		logger.Debug("Do", "response", string(b), "dumpErr", err)
		if err != nil {
			return nil, changed, err
		}
	}
	if resp.Body == nil {
		return nil, changed, nil
	}
	buf.Reset()
	_, err = io.Copy(&buf, resp.Body)
	resp.Body.Close()
	if err != nil {
		logger.Error("read request", "error", err)
	}
	if bytes.Contains(buf.Bytes(), []byte(`"ErrorCode"`)) ||
		bytes.Contains(buf.Bytes(), []byte(`"Error"`)) ||
		bytes.Contains(buf.Bytes(), []byte(`"fault"`)) {
		var jerr JIRAError
		err = json.Unmarshal(buf.Bytes(), &jerr)
		if err != nil {
			logger.Error("Unmarshal JIRAError", "jErr", jerr, "jErrS", fmt.Sprintf("%#v", jerr), "buf", buf.String(), "error", err)
		} else {
			logger.Debug("Unmarshal JIRAError", "jErr", jerr, "jErrS", fmt.Sprintf("%#v", jerr))
		}
		if err == nil && jerr.IsValid() {
			if jerr.Code == "" {
				jerr.Code = resp.Status
			}
			return nil, changed, &jerr
		}
	}
	if resp.StatusCode >= 400 {
		return buf.Bytes(), changed, &JIRAError{Code: resp.Status, Message: buf.String()}
	}
	return buf.Bytes(), changed, nil
}

func redactedURL(u *url.URL) string {
	if u.User == nil {
		return u.String()
	}
	if p, _ := u.User.Password(); p != "" && u.User.Username() == "" {
		return u.String()
	}
	ru := *u
	ru.User = nil
	return ru.String()
}
