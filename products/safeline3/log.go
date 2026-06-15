package safeline3

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type logCommonOptions struct {
	Start    string
	End      string
	Page     int
	PageSize int
}

type attackLogOptions struct {
	logCommonOptions
	SrcIP      string
	Host       string
	URLPath    string
	EventID    string
	AttackType string
	Action     string
	RiskLevel  string
	Method     string
	RuleID     string
}

type accessLogOptions struct {
	logCommonOptions
	SrcIP      string
	Host       string
	URLPath    string
	EventID    string
	StatusCode string
	Method     string
}

type botLogOptions struct {
	logCommonOptions
	SrcIP   string
	DstIP   string
	IsBot   string
	BotType string
	Country string
}

func newLogCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Query SafeLine-3 logs",
		Long: `Query SafeLine-3 protected logs.

List commands call POST log APIs and default to the last 24 hours because
SafeLine-3 requires start_time and end_time. Detail and decode commands call
GET APIs with event_id plus the same time range.`,
	}
	attack := &cobra.Command{Use: "attack", Short: "Query attack detection logs"}
	attack.AddCommand(newAttackLogListCommand())
	attack.AddCommand(newLogDetailCommand("get", "attack", "/api/v3/protected-logger/DetectLogDetail", true))
	attack.AddCommand(newLogDetailCommand("decode", "attack", "/api/v3/protected-logger/DetectLogDecode", true))

	access := &cobra.Command{Use: "access", Short: "Query access logs"}
	access.AddCommand(newAccessLogListCommand())
	access.AddCommand(newLogDetailCommand("get", "access", "/api/v3/protected-logger/AccessLogDetail", true))
	access.AddCommand(newLogDetailCommand("decode", "access", "/api/v3/protected-logger/AccessLogDecode", true))

	bot := &cobra.Command{Use: "bot", Short: "Query bot defense logs"}
	bot.AddCommand(newBotLogListCommand())
	bot.AddCommand(newLogDetailCommand("get", "bot", "/api/v3/protected-logger/BotLogDetail", false))

	cmd.AddCommand(attack, access, bot)
	return cmd
}

func newAttackLogListCommand() *cobra.Command {
	var opts attackLogOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List attack logs",
		Long: `List attack detection logs.

API:
  POST /api/v3/protected-logger/DetectLogList

Required:
  None. CLI defaults --start -24h and --end now.

Filters:
  --src-ip IP/CIDR, --host string, --url-path string, --event-id string,
  --attack-type int, --action int|string, --risk-level low|medium|high|0|1|2|3,
  --method HTTP_METHOD, --rule-id string|int.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := buildAttackLogBody(opts)
			if err != nil {
				return err
			}
			return doRequest(cmd, http.MethodPost, "/api/v3/protected-logger/DetectLogList", nil, body)
		},
	}
	addLogCommonFlags(cmd, &opts.logCommonOptions)
	cmd.Flags().StringVar(&opts.SrcIP, "src-ip", "", "Source IP/CIDR filter")
	cmd.Flags().StringVar(&opts.Host, "host", "", "Host filter")
	cmd.Flags().StringVar(&opts.URLPath, "url-path", "", "URL path filter")
	cmd.Flags().StringVar(&opts.EventID, "event-id", "", "Event ID filter")
	cmd.Flags().StringVar(&opts.AttackType, "attack-type", "", "Attack type integer")
	cmd.Flags().StringVar(&opts.Action, "action", "", "Action integer or name")
	cmd.Flags().StringVar(&opts.RiskLevel, "risk-level", "", "Risk level low|medium|high|0|1|2|3")
	cmd.Flags().StringVar(&opts.Method, "method", "", "HTTP method")
	cmd.Flags().StringVar(&opts.RuleID, "rule-id", "", "Rule ID")
	return cmd
}

func newAccessLogListCommand() *cobra.Command {
	var opts accessLogOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List access logs",
		Long: `List access logs.

API:
  POST /api/v3/protected-logger/AccessLogList

Required:
  None. CLI defaults --start -24h and --end now.

Filters:
  --src-ip IP/CIDR, --host string, --url-path string, --status-code int,
  --method HTTP_METHOD, --event-id string.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := buildAccessLogBody(opts)
			if err != nil {
				return err
			}
			return doRequest(cmd, http.MethodPost, "/api/v3/protected-logger/AccessLogList", nil, body)
		},
	}
	addLogCommonFlags(cmd, &opts.logCommonOptions)
	cmd.Flags().StringVar(&opts.SrcIP, "src-ip", "", "Source IP/CIDR filter")
	cmd.Flags().StringVar(&opts.Host, "host", "", "Host filter")
	cmd.Flags().StringVar(&opts.URLPath, "url-path", "", "URL path filter")
	cmd.Flags().StringVar(&opts.EventID, "event-id", "", "Event ID filter")
	cmd.Flags().StringVar(&opts.StatusCode, "status-code", "", "HTTP status code")
	cmd.Flags().StringVar(&opts.Method, "method", "", "HTTP method")
	return cmd
}

func newBotLogListCommand() *cobra.Command {
	var opts botLogOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List bot logs",
		Long: `List bot defense logs.

API:
  POST /api/v3/protected-logger/BotLogList

Required:
  None. CLI defaults --start -24h and --end now.

Filters:
  --src-ip IP/CIDR, --dst-ip IP/CIDR, --is-bot true|false|0|1|2,
  --bot-type int, --country string.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := buildBotLogBody(opts)
			if err != nil {
				return err
			}
			return doRequest(cmd, http.MethodPost, "/api/v3/protected-logger/BotLogList", nil, body)
		},
	}
	addLogCommonFlags(cmd, &opts.logCommonOptions)
	cmd.Flags().StringVar(&opts.SrcIP, "src-ip", "", "Source IP/CIDR filter")
	cmd.Flags().StringVar(&opts.DstIP, "dst-ip", "", "Destination IP/CIDR filter")
	cmd.Flags().StringVar(&opts.IsBot, "is-bot", "", "Bot verification state true|false|0|1|2")
	cmd.Flags().StringVar(&opts.BotType, "bot-type", "", "Bot type integer")
	cmd.Flags().StringVar(&opts.Country, "country", "", "Country/region filter")
	return cmd
}

func newLogDetailCommand(name, logType, path string, includeTime bool) *cobra.Command {
	var common logCommonOptions
	cmd := &cobra.Command{
		Use:   name + " <event-id>",
		Short: name + " " + logType + " log detail",
		Long: fmt.Sprintf(`%s %s log detail.

API:
  GET %s

Required:
  <event-id> string

Time:
  --start and --end default to the last 24 hours where the API requires them.`, strings.Title(name), logType, path),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q := url.Values{"event_id": {args[0]}}
			if includeTime {
				start, end, err := parseLogRange(common)
				if err != nil {
					return err
				}
				q.Set("start_time", strconv.FormatInt(start, 10))
				q.Set("end_time", strconv.FormatInt(end, 10))
			}
			return doRequest(cmd, http.MethodGet, path, q, nil)
		},
	}
	if includeTime {
		cmd.Flags().StringVar(&common.Start, "start", "-24h", "Start time (RFC3339, local time, unix seconds/ms, relative like -24h)")
		cmd.Flags().StringVar(&common.End, "end", "now", "End time (RFC3339, local time, unix seconds/ms, relative like -24h, now)")
	}
	return cmd
}

func addLogCommonFlags(cmd *cobra.Command, opts *logCommonOptions) {
	cmd.Flags().StringVar(&opts.Start, "start", "-24h", "Start time (RFC3339, local time, unix seconds/ms, relative like -24h)")
	cmd.Flags().StringVar(&opts.End, "end", "now", "End time (RFC3339, local time, unix seconds/ms, relative like -24h, now)")
	cmd.Flags().IntVar(&opts.Page, "page", 1, "Page number")
	cmd.Flags().IntVar(&opts.PageSize, "page-size", 20, "Page size")
}

func buildAttackLogBody(opts attackLogOptions) (map[string]any, error) {
	body, err := baseLogBody(opts.logCommonOptions)
	if err != nil {
		return nil, err
	}
	addStringFilter(body, "src_ip", opts.SrcIP)
	addStringFilter(body, "host", opts.Host)
	addStringFilter(body, "url_path", opts.URLPath)
	addStringFilter(body, "event_id", opts.EventID)
	addStringFilter(body, "method", strings.ToUpper(opts.Method))
	addStringFilter(body, "rule_id", opts.RuleID)
	if err := addIntFilter(body, "attack_type", opts.AttackType); err != nil {
		return nil, err
	}
	if opts.Action != "" {
		action, err := parseActionValue(opts.Action)
		if err != nil {
			return nil, err
		}
		body["action"] = intFilter(action)
	}
	if opts.RiskLevel != "" {
		risk, err := parseRiskLevel(opts.RiskLevel)
		if err != nil {
			return nil, err
		}
		body["risk_level"] = intFilter(risk)
	}
	return body, nil
}

func buildAccessLogBody(opts accessLogOptions) (map[string]any, error) {
	body, err := baseLogBody(opts.logCommonOptions)
	if err != nil {
		return nil, err
	}
	addStringFilter(body, "src_ip", opts.SrcIP)
	addStringFilter(body, "host", opts.Host)
	addStringFilter(body, "url_path", opts.URLPath)
	addStringFilter(body, "event_id", opts.EventID)
	addStringFilter(body, "method", strings.ToUpper(opts.Method))
	if err := addUIntFilter(body, "status_code", opts.StatusCode); err != nil {
		return nil, err
	}
	return body, nil
}

func buildBotLogBody(opts botLogOptions) (map[string]any, error) {
	body, err := baseLogBody(opts.logCommonOptions)
	if err != nil {
		return nil, err
	}
	addStringFilter(body, "src_ip", opts.SrcIP)
	addStringFilter(body, "dst_ip", opts.DstIP)
	addStringFilter(body, "country", opts.Country)
	if err := addIntFilter(body, "bot_type_match", opts.BotType); err != nil {
		return nil, err
	}
	if opts.IsBot != "" {
		value, err := parseBotState(opts.IsBot)
		if err != nil {
			return nil, err
		}
		body["is_bot"] = intFilter(value)
	}
	return body, nil
}

func baseLogBody(opts logCommonOptions) (map[string]any, error) {
	start, end, err := parseLogRange(opts)
	if err != nil {
		return nil, err
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 {
		opts.PageSize = 20
	}
	return map[string]any{
		"start_time": start,
		"end_time":   end,
		"offset":     (opts.Page - 1) * opts.PageSize,
		"count":      opts.PageSize,
	}, nil
}

func parseLogRange(opts logCommonOptions) (int64, int64, error) {
	start, err := parseTimeValue(defaultString(opts.Start, "-24h"))
	if err != nil {
		return 0, 0, err
	}
	end, err := parseTimeValue(defaultString(opts.End, "now"))
	if err != nil {
		return 0, 0, err
	}
	if start >= end {
		return 0, 0, fmt.Errorf("--start must be before --end")
	}
	return start, end, nil
}

func addStringFilter(body map[string]any, key, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	body[key] = map[string]any{"operator": "=", "value": splitValues([]string{value})}
}

func addIntFilter(body map[string]any, key, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return fmt.Errorf("--%s must be an integer", strings.ReplaceAll(key, "_", "-"))
	}
	body[key] = intFilter(int32(parsed))
	return nil
}

func addUIntFilter(body map[string]any, key, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return fmt.Errorf("--%s must be an unsigned integer", strings.ReplaceAll(key, "_", "-"))
	}
	body[key] = map[string]any{"operator": "=", "value": []uint32{uint32(parsed)}}
	return nil
}

func intFilter(value int32) map[string]any {
	return map[string]any{"operator": "=", "value": []int32{value}}
}

func parseRiskLevel(value string) (int32, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low":
		return 1, nil
	case "medium", "middle":
		return 2, nil
	case "high":
		return 3, nil
	default:
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid risk level %q", value)
		}
		return int32(parsed), nil
	}
}

func parseActionValue(value string) (int32, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "allow":
		return 0, nil
	case "deny", "block":
		return 1, nil
	case "forward":
		return 4, nil
	default:
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid action %q, expected allow|deny|forward or integer", value)
		}
		return int32(parsed), nil
	}
}

func parseBotState(value string) (int32, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "bot":
		return 1, nil
	case "false", "human":
		return 0, nil
	default:
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("--is-bot must be true|false|0|1|2")
		}
		return int32(parsed), nil
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
