package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// CLI configuration sourced from env vars.
var (
	apiURL string
	token  string
)

func init() {
	apiURL = os.Getenv("PORTWAY_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}
	token = os.Getenv("PORTWAY_TOKEN")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "catalog":
		err = cmdCatalog()
	case "provision":
		err = cmdProvision(os.Args[2:])
	case "resources":
		err = cmdResources(os.Args[2:])
	case "status":
		err = cmdStatus(os.Args[2:])
	case "delete":
		err = cmdDelete(os.Args[2:])
	case "approvals":
		err = cmdApprovals()
	case "approve":
		err = cmdApprove(os.Args[2:])
	case "help", "--help", "-h":
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: portway-cli <command> [options]

Commands:
  catalog                                 List available resource types
  provision <type-id> [options]           Request resource provisioning
  resources [options]                     List resources
  status <resource-id>                    Get resource detail
  delete <resource-id>                    Request resource deletion
  approvals                              List pending approval requests
  approve <approval-id> [--comment ".."] Approve a request

Provision options:
  --name <name>             Resource name (required)
  --project <project-id>    Project ID (required)
  --spec key=value          Spec fields (repeatable)

Resources options:
  --project <project-id>    Filter by project
  --status <status>         Filter by status

Environment:
  PORTWAY_API_URL    API server URL (default: http://localhost:8080)
  PORTWAY_TOKEN      Authentication token`)
}

// --- HTTP client helpers ---

func doRequest(method, path string, body any) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, apiURL+path, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(data, &errResp) == nil && errResp.Error != "" {
			return data, resp.StatusCode, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error)
		}
		return data, resp.StatusCode, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(data))
	}

	return data, resp.StatusCode, nil
}

// --- Output helpers ---

func statusColor(status string) string {
	switch status {
	case "ready":
		return "\033[32m" + status + "\033[0m" // green
	case "provisioning", "updating", "deleting":
		return "\033[33m" + status + "\033[0m" // yellow
	case "failed":
		return "\033[31m" + status + "\033[0m" // red
	case "requested":
		return "\033[36m" + status + "\033[0m" // cyan
	case "deleted":
		return "\033[90m" + status + "\033[0m" // gray
	case "pending":
		return "\033[33m" + status + "\033[0m" // yellow
	case "approved":
		return "\033[32m" + status + "\033[0m" // green
	case "denied":
		return "\033[31m" + status + "\033[0m" // red
	default:
		return status
	}
}

func newTable() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

func formatTime(raw any) string {
	if raw == nil {
		return "-"
	}
	// Handle the pgtype.Timestamptz JSON format (could be string or object).
	switch v := raw.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.Local().Format("2006-01-02 15:04")
		}
		return v
	case map[string]any:
		if ts, ok := v["Time"].(string); ok {
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				return t.Local().Format("2006-01-02 15:04")
			}
			return ts
		}
	}
	return fmt.Sprintf("%v", raw)
}

func parseArgs(args []string) map[string][]string {
	result := map[string][]string{}
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			key := strings.TrimPrefix(args[i], "--")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				result[key] = append(result[key], args[i+1])
				i++
			} else {
				result[key] = append(result[key], "true")
			}
		}
	}
	return result
}

func getArg(parsed map[string][]string, key string) string {
	vals := parsed[key]
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// --- Commands ---

func cmdCatalog() error {
	data, _, err := doRequest("GET", "/api/v1/resource-types", nil)
	if err != nil {
		return err
	}

	var types []map[string]any
	if err := json.Unmarshal(data, &types); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if len(types) == 0 {
		fmt.Println("No resource types found.")
		return nil
	}

	w := newTable()
	fmt.Fprintln(w, "ID\tNAME\tSLUG\tCATEGORY\tENABLED")
	for _, t := range types {
		enabled := "yes"
		if e, ok := t["enabled"].(bool); ok && !e {
			enabled = "no"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			t["id"], t["name"], t["slug"], t["category"], enabled)
	}
	return w.Flush()
}

func cmdProvision(args []string) error {
	if len(args) < 1 || strings.HasPrefix(args[0], "--") {
		return fmt.Errorf("usage: portway-cli provision <resource-type-id> --name <name> --project <project-id> [--spec key=value ...]")
	}

	resourceTypeID := args[0]
	parsed := parseArgs(args[1:])

	name := getArg(parsed, "name")
	projectID := getArg(parsed, "project")

	if name == "" || projectID == "" {
		return fmt.Errorf("--name and --project are required")
	}

	spec := map[string]any{}
	for _, kv := range parsed["spec"] {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			spec[parts[0]] = parts[1]
		}
	}

	body := map[string]any{
		"resource_type_id": resourceTypeID,
		"name":             name,
		"project_id":       projectID,
	}
	if len(spec) > 0 {
		body["spec"] = spec
	}

	data, _, err := doRequest("POST", "/api/v1/resources", body)
	if err != nil {
		return err
	}

	var resource map[string]any
	if err := json.Unmarshal(data, &resource); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Resource provisioning requested.\n")
	fmt.Printf("  ID:     %s\n", resource["id"])
	fmt.Printf("  Name:   %s\n", resource["name"])
	fmt.Printf("  Status: %s\n", statusColor(fmt.Sprintf("%v", resource["status"])))
	return nil
}

func cmdResources(args []string) error {
	parsed := parseArgs(args)

	params := url.Values{}
	if v := getArg(parsed, "project"); v != "" {
		params.Set("project_id", v)
	}
	if v := getArg(parsed, "status"); v != "" {
		params.Set("status", v)
	}

	if params.Get("project_id") == "" && params.Get("status") == "" {
		return fmt.Errorf("--project or --status is required")
	}

	path := "/api/v1/resources"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	data, _, err := doRequest("GET", path, nil)
	if err != nil {
		return err
	}

	var resources []map[string]any
	if err := json.Unmarshal(data, &resources); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if len(resources) == 0 {
		fmt.Println("No resources found.")
		return nil
	}

	w := newTable()
	fmt.Fprintln(w, "ID\tNAME\tSTATUS\tPROVIDER REF\tCREATED")
	for _, r := range resources {
		provRef := fmt.Sprintf("%v", r["provider_ref"])
		if provRef == "" || provRef == "<nil>" {
			provRef = "-"
		}
		// Truncate long provider refs.
		if len(provRef) > 40 {
			provRef = provRef[:37] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			r["id"],
			r["name"],
			statusColor(fmt.Sprintf("%v", r["status"])),
			provRef,
			formatTime(r["created_at"]),
		)
	}
	return w.Flush()
}

func cmdStatus(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: portway-cli status <resource-id>")
	}

	resourceID := args[0]

	data, _, err := doRequest("GET", "/api/v1/resources/"+resourceID, nil)
	if err != nil {
		return err
	}

	var resource map[string]any
	if err := json.Unmarshal(data, &resource); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Resource: %s\n", resource["name"])
	fmt.Printf("  ID:             %s\n", resource["id"])
	fmt.Printf("  Project:        %s\n", resource["project_id"])
	fmt.Printf("  Resource Type:  %s\n", resource["resource_type_id"])
	fmt.Printf("  Status:         %s\n", statusColor(fmt.Sprintf("%v", resource["status"])))
	if msg, ok := resource["status_message"].(string); ok && msg != "" {
		fmt.Printf("  Message:        %s\n", msg)
	}
	if ref, ok := resource["provider_ref"].(string); ok && ref != "" {
		fmt.Printf("  Provider Ref:   %s\n", ref)
	}
	fmt.Printf("  Requested By:   %s\n", resource["requested_by"])
	fmt.Printf("  Created:        %s\n", formatTime(resource["created_at"]))
	fmt.Printf("  Updated:        %s\n", formatTime(resource["updated_at"]))

	if spec, ok := resource["spec"]; ok && spec != nil {
		specJSON, _ := json.MarshalIndent(spec, "  ", "  ")
		fmt.Printf("  Spec:\n  %s\n", string(specJSON))
	}

	return nil
}

func cmdDelete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: portway-cli delete <resource-id>")
	}

	resourceID := args[0]

	data, _, err := doRequest("DELETE", "/api/v1/resources/"+resourceID, nil)
	if err != nil {
		return err
	}

	var resource map[string]any
	if err := json.Unmarshal(data, &resource); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Resource deletion requested.\n")
	fmt.Printf("  ID:     %s\n", resource["id"])
	fmt.Printf("  Status: %s\n", statusColor(fmt.Sprintf("%v", resource["status"])))
	return nil
}

func cmdApprovals() error {
	data, _, err := doRequest("GET", "/api/v1/approvals?status=pending", nil)
	if err != nil {
		return err
	}

	var approvals []map[string]any
	if err := json.Unmarshal(data, &approvals); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if len(approvals) == 0 {
		fmt.Println("No pending approvals.")
		return nil
	}

	w := newTable()
	fmt.Fprintln(w, "ID\tPROJECT\tRESOURCE TYPE\tREQUESTED BY\tSTATUS\tCREATED")
	for _, a := range approvals {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			a["id"],
			a["project_id"],
			a["resource_type"],
			a["requested_by"],
			statusColor(fmt.Sprintf("%v", a["status"])),
			formatTime(a["created_at"]),
		)
	}
	return w.Flush()
}

func cmdApprove(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: portway-cli approve <approval-id> [--comment \"...\"]")
	}

	approvalID := args[0]
	parsed := parseArgs(args[1:])
	comment := getArg(parsed, "comment")

	body := map[string]any{
		"status": "approved",
	}
	if comment != "" {
		body["comment"] = comment
	}

	_, _, err := doRequest("POST", "/api/v1/approvals/"+approvalID+"/review", body)
	if err != nil {
		return err
	}

	fmt.Printf("Approval %s approved.\n", approvalID)
	if comment != "" {
		fmt.Printf("  Comment: %s\n", comment)
	}
	return nil
}
