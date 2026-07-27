package grafana_test

import (
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const dashboardPath = "braiins-pool-exporter.json"

var allowedMetrics = map[string]struct{}{
	"braiins_pool_account_balance_btc":                 {},
	"braiins_pool_account_hashrate_ghs":                {},
	"braiins_pool_account_workers":                     {},
	"braiins_pool_api_last_success_timestamp_seconds":  {},
	"braiins_pool_api_requests_total":                  {},
	"braiins_pool_active_workers":                      {},
	"braiins_pool_data_age_seconds":                    {},
	"braiins_pool_exporter_build_info":                 {},
	"braiins_pool_exporter_ready":                      {},
	"braiins_pool_hashrate_ghs":                        {},
	"braiins_pool_payout_amount_sats":                  {},
	"braiins_pool_payout_fee_sats":                     {},
	"braiins_pool_reward_daily_btc":                    {},
	"braiins_pool_stats_update_timestamp_seconds":      {},
	"braiins_pool_worker_hashrate_ghs":                 {},
	"braiins_pool_worker_last_share_age_seconds":       {},
	"braiins_pool_worker_last_share_timestamp_seconds": {},
	"braiins_pool_worker_shares":                       {},
	"braiins_pool_worker_state":                        {},
}

var approvedUnits = map[string]struct{}{
	"dateTimeAsIso": {},
	"none":          {},
	"percentunit":   {},
	"reqps":         {},
	"s":             {},
	"short":         {},
	"suffix: BTC":   {},
	"suffix:Gh/s":   {},
	"suffix: sats":  {},
}

type dashboard struct {
	UID        string `json:"uid"`
	Title      string `json:"title"`
	Schema     int    `json:"schemaVersion"`
	Refresh    string `json:"refresh"`
	Panels     []panel
	Templating struct {
		List []variable `json:"list"`
	} `json:"templating"`
}

type panel struct {
	Collapsed   bool       `json:"collapsed"`
	Datasource  datasource `json:"datasource"`
	Description string     `json:"description"`
	FieldConfig struct {
		Defaults struct {
			Unit string `json:"unit"`
		} `json:"defaults"`
	} `json:"fieldConfig"`
	Targets []target  `json:"targets"`
	Panels  []panel   `json:"panels"`
	Title   string    `json:"title"`
	Type    string    `json:"type"`
	Alert   *struct{} `json:"alert"`
}

type target struct {
	Datasource datasource `json:"datasource"`
	Expr       string     `json:"expr"`
	RefID      string     `json:"refId"`
}

type datasource struct {
	Type string `json:"type"`
	UID  string `json:"uid"`
}

type variable struct {
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	Query      any        `json:"query"`
	Definition string     `json:"definition"`
	AllValue   string     `json:"allValue"`
	IncludeAll bool       `json:"includeAll"`
	Multi      bool       `json:"multi"`
	Datasource datasource `json:"datasource"`
}

func TestDashboardIdentityAndVariables(t *testing.T) {
	dash := loadDashboard(t)

	if dash.UID != "braiins-pool-exporter" {
		t.Fatalf("dashboard UID = %q, want braiins-pool-exporter", dash.UID)
	}
	if dash.Title != "Braiins Pool Exporter" {
		t.Fatalf("dashboard title = %q, want Braiins Pool Exporter", dash.Title)
	}
	if dash.Schema != 39 {
		t.Fatalf("schemaVersion = %d, want 39", dash.Schema)
	}
	if dash.Refresh != "1m" {
		t.Fatalf("refresh = %q, want 1m", dash.Refresh)
	}

	vars := map[string]variable{}
	for _, variable := range dash.Templating.List {
		vars[variable.Name] = variable
	}
	for _, name := range []string{"DS_PROMETHEUS", "job", "instance", "worker"} {
		if _, ok := vars[name]; !ok {
			t.Fatalf("missing dashboard variable %q", name)
		}
	}

	if got := vars["DS_PROMETHEUS"].Type; got != "datasource" {
		t.Fatalf("DS_PROMETHEUS type = %q, want datasource", got)
	}
	if got := vars["job"].Definition; got != "label_values(braiins_pool_exporter_ready, job)" {
		t.Fatalf("job variable query = %q", got)
	}
	if got := vars["instance"].Definition; got != "label_values(braiins_pool_exporter_ready{job=~\"$job\"}, instance)" {
		t.Fatalf("instance variable query = %q", got)
	}
	if got := vars["worker"].Definition; got != "label_values(braiins_pool_worker_state{job=~\"$job\", instance=~\"$instance\"}, worker)" {
		t.Fatalf("worker variable query = %q", got)
	}
	for _, name := range []string{"job", "instance", "worker"} {
		if !vars[name].IncludeAll || !vars[name].Multi || vars[name].AllValue != ".*" {
			t.Fatalf("%s variable must be multi-select include-all with allValue .*", name)
		}
	}
}

func TestDashboardQueriesArePortableAndUseKnownMetrics(t *testing.T) {
	dash := loadDashboard(t)
	metricName := regexp.MustCompile(`braiins_pool(?:_exporter)?_[a-z0-9_]+`)

	for _, panel := range allPanels(dash.Panels) {
		if panel.Type == "row" {
			continue
		}
		if panel.Alert != nil {
			t.Fatalf("panel %q contains alert configuration", panel.Title)
		}
		if panel.Datasource.UID != "${DS_PROMETHEUS}" {
			t.Fatalf("panel %q datasource UID = %q, want ${DS_PROMETHEUS}", panel.Title, panel.Datasource.UID)
		}
		if panel.FieldConfig.Defaults.Unit != "" {
			if _, ok := approvedUnits[panel.FieldConfig.Defaults.Unit]; !ok {
				t.Fatalf("panel %q uses unapproved unit %q", panel.Title, panel.FieldConfig.Defaults.Unit)
			}
		}
		if len(panel.Targets) == 0 {
			t.Fatalf("panel %q has no query targets", panel.Title)
		}
		for _, target := range panel.Targets {
			if strings.TrimSpace(target.Expr) == "" {
				t.Fatalf("panel %q target %s has empty PromQL expression", panel.Title, target.RefID)
			}
			if target.Datasource.UID != "${DS_PROMETHEUS}" {
				t.Fatalf("panel %q target %s datasource UID = %q, want ${DS_PROMETHEUS}", panel.Title, target.RefID, target.Datasource.UID)
			}
			for _, name := range metricName.FindAllString(target.Expr, -1) {
				if _, ok := allowedMetrics[name]; !ok {
					t.Fatalf("panel %q target %s references unknown metric %q in %q", panel.Title, target.RefID, name, target.Expr)
				}
			}
			if strings.Contains(target.Expr, "braiins_pool_") && !strings.Contains(target.Expr, `job=~"$job"`) {
				t.Fatalf("panel %q target %s lacks portable job filter in %q", panel.Title, target.RefID, target.Expr)
			}
			if strings.Contains(target.Expr, "braiins_pool_") && !strings.Contains(target.Expr, `instance=~"$instance"`) {
				t.Fatalf("panel %q target %s lacks portable instance filter in %q", panel.Title, target.RefID, target.Expr)
			}
			if strings.Contains(target.Expr, "_total") && !strings.Contains(target.Expr, "rate(") {
				t.Fatalf("counter query in panel %q target %s must use rate(): %q", panel.Title, target.RefID, target.Expr)
			}
		}
	}
}

func TestDashboardContainsNoForbiddenPublicValues(t *testing.T) {
	raw, err := os.ReadFile(dashboardPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, forbidden := range []string{
		"SECRET" + "S.md",
		"Pool-Auth" + "-Token",
		"Author" + "ization",
		"local" + "host",
		"127." + "0.0.1",
		"0.0." + "0.0",
		"192." + "168.",
		"10" + ".",
		"172." + "16.",
		"172." + "17.",
		"172." + "18.",
		"172." + "19.",
		"C:" + "\\",
		"work" + "shop",
		"home" + "lab",
		"Smuts" + " Tech",
		"Smuts" + " Me",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("dashboard contains forbidden public value %q", forbidden)
		}
	}

	addressLike := regexp.MustCompile(`(?i)(bc1|[13])[a-z0-9]{25,}`)
	if addressLike.MatchString(text) {
		t.Fatal("dashboard appears to contain a Bitcoin address-like value")
	}
	longHex := regexp.MustCompile(`(?i)\b[0-9a-f]{64}\b`)
	if longHex.MatchString(text) {
		t.Fatal("dashboard appears to contain a transaction-id-like value")
	}
}

func TestDashboardMetricInventoryStaysExplicit(t *testing.T) {
	dash := loadDashboard(t)
	metricName := regexp.MustCompile(`braiins_pool(?:_exporter)?_[a-z0-9_]+`)
	seen := map[string]struct{}{}
	for _, panel := range allPanels(dash.Panels) {
		for _, target := range panel.Targets {
			for _, name := range metricName.FindAllString(target.Expr, -1) {
				seen[name] = struct{}{}
			}
		}
	}

	var referenced []string
	for name := range seen {
		referenced = append(referenced, name)
	}
	slices.Sort(referenced)

	for _, name := range referenced {
		if _, ok := allowedMetrics[name]; !ok {
			t.Fatalf("unknown metric reference %q", name)
		}
	}
}

func TestDashboardContainsCanonicalPoolStatisticsSection(t *testing.T) {
	dash := loadDashboard(t)
	titles := map[string]struct{}{}
	for _, panel := range dash.Panels {
		if panel.Title != "Pool Statistics" {
			titles[panel.Title] = struct{}{}
			continue
		}
		if panel.Type != "row" {
			t.Fatalf("Pool Statistics type = %q, want row", panel.Type)
		}
	}
	for _, panel := range allPanels(dash.Panels) {
		titles[panel.Title] = struct{}{}
	}
	if _, ok := titles["Pool Statistics"]; !ok {
		t.Fatal("missing Pool Statistics row")
	}
	for _, title := range []string{"Pool hashrate", "Pool active workers", "Pool stats freshness"} {
		if _, ok := titles[title]; !ok {
			t.Fatalf("Pool Statistics section missing panel %q", title)
		}
	}
}

func TestDashboardDoesNotDisplayUnsupportedPoolValues(t *testing.T) {
	dash := loadDashboard(t)
	for _, panel := range allPanels(dash.Panels) {
		title := strings.ToLower(panel.Title)
		for _, unsupported := range []string{"active users", "30m", "30-minute"} {
			if strings.Contains(title, unsupported) {
				t.Fatalf("panel title displays unsupported pool value %q: %q", unsupported, panel.Title)
			}
		}
		for _, target := range panel.Targets {
			expr := strings.ToLower(target.Expr)
			for _, unsupported := range []string{"active_users", "active users", "30m", "30-minute"} {
				if strings.Contains(expr, unsupported) {
					t.Fatalf("panel %q target %s queries unsupported pool value %q in %q", panel.Title, target.RefID, unsupported, target.Expr)
				}
			}
		}
	}
}

func allPanels(panels []panel) []panel {
	var out []panel
	var walk func([]panel)
	walk = func(items []panel) {
		for _, panel := range items {
			out = append(out, panel)
			if len(panel.Panels) > 0 {
				walk(panel.Panels)
			}
		}
	}
	walk(panels)
	return out
}

func loadDashboard(t *testing.T) dashboard {
	t.Helper()
	raw, err := os.ReadFile(dashboardPath)
	if err != nil {
		t.Fatal(err)
	}
	var dash dashboard
	if err := json.Unmarshal(raw, &dash); err != nil {
		t.Fatal(err)
	}
	return dash
}
