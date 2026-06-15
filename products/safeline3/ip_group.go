package safeline3

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

type ipGroupListResponse struct {
	Total int           `json:"total"`
	Items []ipGroupItem `json:"items"`
}

type ipGroupItem struct {
	ID       uint64   `json:"id"`
	Name     string   `json:"name"`
	Comment  string   `json:"comment"`
	Original []string `json:"original"`
}

func newIPGroupCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "ip-group", Short: "Manage IP groups"}
	cmd.AddCommand(newIPGroupListCommand())
	cmd.AddCommand(newIPGroupGetCommand())
	cmd.AddCommand(newIPGroupCreateCommand())
	cmd.AddCommand(newIPGroupUpdateCommand())
	cmd.AddCommand(newIPGroupPatchIPsCommand("add-ip", true))
	cmd.AddCommand(newIPGroupPatchIPsCommand("remove-ip", false))
	cmd.AddCommand(newIPGroupDeleteCommand())
	return cmd
}

func newIPGroupListCommand() *cobra.Command {
	var name, nameExact, cidr string
	var page, pageSize int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List IP groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := fetchIPGroups(cmd, page, pageSize)
			if err != nil {
				return err
			}
			resp.Items = filterIPGroups(resp.Items, name, nameExact, cidr)
			resp.Total = len(resp.Items)
			return getRenderer(cmd).Render(resp)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Name filter")
	cmd.Flags().StringVar(&nameExact, "name-exact", "", "Exact name")
	cmd.Flags().StringVar(&cidr, "cidr", "", "CIDR filter")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "Page size")
	return cmd
}

func newIPGroupGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get IP group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			item, err := fetchIPGroupByID(cmd, id)
			if err != nil {
				return err
			}
			return getRenderer(cmd).Render(item)
		},
	}
}

func newIPGroupCreateCommand() *cobra.Command {
	var name, comment string
	var ips []string
	var opts writeOptions
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create IP group",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			values := splitValues(ips)
			if len(values) == 0 {
				return fmt.Errorf("--ip is required")
			}
			body := map[string]any{"name": name, "comment": comment, "original": values}
			return doWrite(cmd, opts, "ip-group.create", http.MethodPost, "/api/v3/detect/ip_group", nil, body, nil)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "IP group name")
	cmd.Flags().StringArrayVar(&ips, "ip", nil, "IP/CIDR; repeatable or comma separated")
	cmd.Flags().StringVar(&comment, "comment", "", "Comment")
	addWriteFlags(cmd, &opts)
	return cmd
}

func newIPGroupUpdateCommand() *cobra.Command {
	var name, comment string
	var ips []string
	var opts writeOptions
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update IP group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			current, err := fetchIPGroupByID(cmd, id)
			if err != nil {
				return err
			}
			if name == "" {
				name = current.Name
			}
			if comment == "" {
				comment = current.Comment
			}
			values := splitValues(ips)
			if len(values) == 0 {
				values = current.Original
			}
			body := map[string]any{"id": id, "name": name, "comment": comment, "original": values}
			return doWrite(cmd, opts, "ip-group.update", http.MethodPut, "/api/v3/detect/ip_group", nil, body, []string{"SafeLine-3 IP group update requires a full name/comment/original payload; omitted fields are read from current state first"})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "IP group name")
	cmd.Flags().StringArrayVar(&ips, "ip", nil, "Replace IP/CIDR list")
	cmd.Flags().StringVar(&comment, "comment", "", "Comment")
	addWriteFlags(cmd, &opts)
	return cmd
}

func newIPGroupPatchIPsCommand(name string, add bool) *cobra.Command {
	var ips []string
	var opts writeOptions
	cmd := &cobra.Command{
		Use:   name + " <id>",
		Short: name + " IP/CIDR entries",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			values := splitValues(ips)
			if len(values) == 0 {
				return fmt.Errorf("--ip is required")
			}
			current, err := fetchIPGroupByID(cmd, id)
			if err != nil {
				return err
			}
			original := patchIPList(current.Original, values, add)
			body := map[string]any{"id": id, "name": current.Name, "comment": current.Comment, "original": original}
			return doWrite(cmd, opts, "ip-group."+name, http.MethodPut, "/api/v3/detect/ip_group", nil, body, []string{"SafeLine-3 has no patch IP API; CLI reads current entries and submits a full update payload"})
		},
	}
	cmd.Flags().StringArrayVar(&ips, "ip", nil, "IP/CIDR; repeatable or comma separated")
	addWriteFlags(cmd, &opts)
	return cmd
}

func fetchIPGroups(cmd *cobra.Command, page, pageSize int) (ipGroupListResponse, error) {
	q := url.Values{}
	addPagination(q, page, pageSize)
	var resp ipGroupListResponse
	if err := getClient(cmd).Do(context.Background(), http.MethodGet, "/api/v3/detect/ip_group", q, nil, &resp); err != nil {
		return ipGroupListResponse{}, err
	}
	return resp, nil
}

func fetchIPGroupByID(cmd *cobra.Command, id uint64) (ipGroupItem, error) {
	page := 1
	pageSize := 100
	for {
		resp, err := fetchIPGroups(cmd, page, pageSize)
		if err != nil {
			return ipGroupItem{}, err
		}
		for _, item := range resp.Items {
			if item.ID == id {
				return item, nil
			}
		}
		if page*pageSize >= resp.Total || len(resp.Items) == 0 {
			break
		}
		page++
	}
	return ipGroupItem{}, fmt.Errorf("IP group %d not found", id)
}

func filterIPGroups(items []ipGroupItem, name, nameExact, cidr string) []ipGroupItem {
	if name == "" && nameExact == "" && cidr == "" {
		return items
	}
	out := make([]ipGroupItem, 0, len(items))
	for _, item := range items {
		if nameExact != "" && item.Name != nameExact {
			continue
		}
		if name != "" && !containsFold(item.Name, name) {
			continue
		}
		if cidr != "" && !ipGroupContains(item.Original, cidr) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func ipGroupContains(values []string, needle string) bool {
	for _, value := range values {
		if strings.EqualFold(value, needle) || containsFold(value, needle) {
			return true
		}
	}
	return false
}

func patchIPList(current, delta []string, add bool) []string {
	seen := map[string]bool{}
	for _, value := range current {
		seen[value] = true
	}
	if add {
		out := append([]string{}, current...)
		for _, value := range delta {
			if !seen[value] {
				out = append(out, value)
				seen[value] = true
			}
		}
		return out
	}
	remove := map[string]bool{}
	for _, value := range delta {
		remove[value] = true
	}
	out := make([]string, 0, len(current))
	for _, value := range current {
		if !remove[value] {
			out = append(out, value)
		}
	}
	return out
}

func newIPGroupDeleteCommand() *cobra.Command {
	var all bool
	var opts writeOptions
	cmd := &cobra.Command{
		Use:   "delete <id...>",
		Short: "Delete IP groups",
		Args: func(cmd *cobra.Command, args []string) error {
			if all {
				return nil
			}
			return cobra.MinimumNArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				return doWrite(cmd, opts, "ip-group.delete-all", http.MethodDelete, "/api/v3/detect/ip_group", nil, map[string]any{"all": true}, nil)
			}
			ids, err := parseIDs(args)
			if err != nil {
				return err
			}
			return doWrite(cmd, opts, "ip-group.delete", http.MethodDelete, "/api/v3/detect/ip_group", nil, map[string]any{"ids": ids}, nil)
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Delete all IP groups")
	addWriteFlags(cmd, &opts)
	return cmd
}
