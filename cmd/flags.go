package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"vine/store"
	"vine/utils"
)

// AddJSONFlag adds a --json flag to a command for programmatic output.
func AddJSONFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("json", false, "output in JSON format")
}

// IsJSON returns true if --json was passed.
func IsJSON(cmd *cobra.Command) bool {
	j, _ := cmd.Flags().GetBool("json")
	return j
}

// AddRootFlag adds a --root flag to filter to root-level tasks only.
func AddRootFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("root", false, "only show root-level tasks (no parent)")
}

// IsRoot returns true if --root was passed.
func IsRoot(cmd *cobra.Command) bool {
	r, _ := cmd.Flags().GetBool("root")
	return r
}

// subtaskLabel returns a dim label like "(3 subtasks)" if the task has children.
func subtaskLabel(counts map[string]int, taskID string) string {
	if n, ok := counts[taskID]; ok && n > 0 {
		return " " + utils.Dim(fmt.Sprintf("(%d subtasks)", n))
	}
	return ""
}

// collectChildCounts gets child counts for a slice of tasks.
func collectChildCounts(s *store.Store, tasks []store.Task) map[string]int {
	ids := make([]string, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	counts, _ := s.ChildCounts(ids)
	if counts == nil {
		counts = make(map[string]int)
	}
	return counts
}

// collectParents batch-fetches parent tasks for all tasks that have parents.
// Returns a map from parent ID to the parent Task.
func collectParents(s *store.Store, tasks []store.Task) map[string]*store.Task {
	seen := map[string]bool{}
	for _, t := range tasks {
		if t.ParentID != nil {
			seen[*t.ParentID] = true
		}
	}
	if len(seen) == 0 {
		return nil
	}
	result := make(map[string]*store.Task, len(seen))
	for id := range seen {
		if t, err := s.GetTask(id); err == nil {
			result[id] = t
		}
	}
	return result
}

// parentLabel returns a dim "(sub of vine-abc12 Name)" label, or "" if no parent.
func parentLabel(project string, parents map[string]*store.Task, parentID *string) string {
	if parentID == nil {
		return ""
	}
	ref := utils.FormatTaskID(project, *parentID)
	if p, ok := parents[*parentID]; ok {
		return " " + utils.Dim(fmt.Sprintf("(sub of %s %s)", ref, p.Name))
	}
	return " " + utils.Dim("(sub of "+ref+")")
}

// blockerLabel returns a dim "(blocked by vine-abc12 Name, ...)" string.
func blockerLabel(project string, blockers []*store.Task) string {
	if len(blockers) == 0 {
		return ""
	}
	refs := make([]string, len(blockers))
	for i, t := range blockers {
		refs[i] = utils.FormatTaskID(project, t.ID) + " " + t.Name
	}
	summary := strings.Join(refs, ", ")
	if len(refs) > 3 {
		summary = strings.Join(refs[:3], ", ") + fmt.Sprintf(", +%d more", len(refs)-3)
	}
	return " " + utils.Dim(fmt.Sprintf("(blocked by %s)", summary))
}

// PrintOutput prints text normally, or marshals data as JSON if --json is set.
func PrintOutput(cmd *cobra.Command, text string, data any) {
	if IsJSON(cmd) {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(data); err != nil {
			fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}
	fmt.Println(text)
}
