package grafana_test

import (
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const avalonDashboardPath = "dashboards/avalon-dashboard.json"

var allowedAvalonMetrics = map[string]struct{}{
	"avalon_best_share":                    {},
	"avalon_chip_count":                    {},
	"avalon_chip_matching_work":            {},
	"avalon_chip_matching_work_avg":        {},
	"avalon_chip_matching_work_max":        {},
	"avalon_chip_matching_work_min":        {},
	"avalon_chip_matching_work_sum":        {},
	"avalon_chip_temp_avg_celsius":         {},
	"avalon_chip_temp_celsius":             {},
	"avalon_chip_temp_max_celsius":         {},
	"avalon_chip_temp_min_celsius":         {},
	"avalon_chip_voltage_avg_volts":        {},
	"avalon_chip_voltage_max_volts":        {},
	"avalon_chip_voltage_min_volts":        {},
	"avalon_chip_voltage_volts":            {},
	"avalon_fan1_rpm":                      {},
	"avalon_fan2_rpm":                      {},
	"avalon_fan_duty_percent":              {},
	"avalon_hashrate_avg_ghs":              {},
	"avalon_hashrate_ghs":                  {},
	"avalon_hashrate_moving_ghs":           {},
	"avalon_hashboard_hashrate_ghs":        {},
	"avalon_hashboard_temp_avg_celsius":    {},
	"avalon_hashboard_temp_max_celsius":    {},
	"avalon_hw_error_rate_percent":         {},
	"avalon_hw_errors_total":               {},
	"avalon_info":                          {},
	"avalon_last_scrape_timestamp_seconds": {},
	"avalon_mpo_target":                    {},
	"avalon_pool_shares_accepted_total":    {},
	"avalon_pool_shares_rejected_total":    {},
	"avalon_pool_stale_total":              {},
	"avalon_pool_up":                       {},
	"avalon_power_err":                     {},
	"avalon_power_iout":                    {},
	"avalon_power_pout_wall":               {},
	"avalon_power_vout":                    {},
	"avalon_power_vout_cmd":                {},
	"avalon_ps_slot_0":                     {},
	"avalon_ps_slot_1":                     {},
	"avalon_ps_slot_2":                     {},
	"avalon_ps_slot_3":                     {},
	"avalon_ps_slot_4":                     {},
	"avalon_ps_slot_5":                     {},
	"avalon_scrape_duration_seconds":       {},
	"avalon_scrape_errors_parse_total":     {},
	"avalon_scrape_errors_timeout_total":   {},
	"avalon_scrape_errors_total":           {},
	"avalon_shares_accepted_total":         {},
	"avalon_shares_rejected_total":         {},
	"avalon_shares_stale_total":            {},
	"avalon_temp_avg_celsius":              {},
	"avalon_temp_current_celsius":          {},
	"avalon_temp_max_celsius":              {},
	"avalon_temp_target_celsius":           {},
	"avalon_up":                            {},
	"avalon_work_utility":                  {},
	"avalon_work_utility_summary":          {},
}

var approvedAvalonUnits = map[string]struct{}{
	"celsius":       {},
	"dateTimeAsIso": {},
	"none":          {},
	"ops":           {},
	"percent":       {},
	"rpm":           {},
	"s":             {},
	"short":         {},
	"suffix: GH/s":  {},
}

func TestAvalonDashboardIdentityAndVariables(t *testing.T) {
	dash := loadAvalonDashboard(t)

	if dash.UID != "avalon-miner-exporter" {
		t.Fatalf("dashboard UID = %q, want avalon-miner-exporter", dash.UID)
	}
	if dash.Title != "Avalon Miner Exporter" {
		t.Fatalf("dashboard title = %q, want Avalon Miner Exporter", dash.Title)
	}

	vars := map[string]variable{}
	for _, variable := range dash.Templating.List {
		vars[variable.Name] = variable
	}
	for _, name := range []string{"DS_PROMETHEUS", "job", "instance"} {
		if _, ok := vars[name]; !ok {
			t.Fatalf("missing dashboard variable %q", name)
		}
	}
	if got := vars["DS_PROMETHEUS"].Type; got != "datasource" {
		t.Fatalf("DS_PROMETHEUS type = %q, want datasource", got)
	}
	if got := vars["job"].Definition; got != "label_values(avalon_up, job)" {
		t.Fatalf("job variable query = %q", got)
	}
	if got := vars["instance"].Definition; got != `label_values(avalon_up{job=~"$job"}, instance)` {
		t.Fatalf("instance variable query = %q", got)
	}
	for _, name := range []string{"job", "instance"} {
		if !vars[name].IncludeAll || !vars[name].Multi || vars[name].AllValue != ".*" {
			t.Fatalf("%s variable must be multi-select include-all with allValue .*", name)
		}
	}
}

func TestAvalonDashboardQueriesArePortableAndUseKnownMetrics(t *testing.T) {
	dash := loadAvalonDashboard(t)
	metricName := regexp.MustCompile(`avalon_[a-z0-9_]+`)

	for _, panel := range dash.Panels {
		if panel.Alert != nil {
			t.Fatalf("panel %q contains alert configuration", panel.Title)
		}
		if len(panel.Targets) == 0 {
			continue
		}
		if panel.Datasource.UID != "${DS_PROMETHEUS}" {
			t.Fatalf("panel %q datasource UID = %q, want ${DS_PROMETHEUS}", panel.Title, panel.Datasource.UID)
		}
		if panel.FieldConfig.Defaults.Unit != "" {
			if _, ok := approvedAvalonUnits[panel.FieldConfig.Defaults.Unit]; !ok {
				t.Fatalf("panel %q uses unapproved unit %q", panel.Title, panel.FieldConfig.Defaults.Unit)
			}
		}
		for _, target := range panel.Targets {
			if strings.TrimSpace(target.Expr) == "" {
				t.Fatalf("panel %q target %s has empty PromQL expression", panel.Title, target.RefID)
			}
			if target.Datasource.UID != "${DS_PROMETHEUS}" {
				t.Fatalf("panel %q target %s datasource UID = %q, want ${DS_PROMETHEUS}", panel.Title, target.RefID, target.Datasource.UID)
			}
			for _, name := range metricName.FindAllString(target.Expr, -1) {
				if _, ok := allowedAvalonMetrics[name]; !ok {
					t.Fatalf("panel %q target %s references unknown metric %q in %q", panel.Title, target.RefID, name, target.Expr)
				}
			}
			if strings.Contains(target.Expr, "avalon_") && !strings.Contains(target.Expr, `job=~"$job"`) {
				t.Fatalf("panel %q target %s lacks portable job filter in %q", panel.Title, target.RefID, target.Expr)
			}
			if strings.Contains(target.Expr, "avalon_") && !strings.Contains(target.Expr, `instance=~"$instance"`) {
				t.Fatalf("panel %q target %s lacks portable instance filter in %q", panel.Title, target.RefID, target.Expr)
			}
		}
	}
}

func TestAvalonDashboardContainsNoForbiddenPublicValues(t *testing.T) {
	raw, err := os.ReadFile(avalonDashboardPath)
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
}

func TestAvalonDashboardMetricInventoryStaysExplicit(t *testing.T) {
	dash := loadAvalonDashboard(t)
	metricName := regexp.MustCompile(`avalon_[a-z0-9_]+`)
	seen := map[string]struct{}{}
	for _, panel := range dash.Panels {
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
		if _, ok := allowedAvalonMetrics[name]; !ok {
			t.Fatalf("unknown metric reference %q", name)
		}
	}
}

func loadAvalonDashboard(t *testing.T) dashboard {
	t.Helper()
	raw, err := os.ReadFile(avalonDashboardPath)
	if err != nil {
		t.Fatal(err)
	}
	var dash dashboard
	if err := json.Unmarshal(raw, &dash); err != nil {
		t.Fatal(err)
	}
	return dash
}
