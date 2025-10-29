package whisker

import (
	"fmt"
	"strings"

	"github.com/aadhilam/mcp-whisker-go/pkg/types"
)

// buildMarkdownTable creates a Markdown table from headers and rows
func buildMarkdownTable(headers []string, rows [][]string) string {
	var sb strings.Builder

	// Write headers
	sb.WriteString("|")
	for _, header := range headers {
		sb.WriteString(header)
		sb.WriteString("|")
	}
	sb.WriteString("\n")

	// Write separator
	sb.WriteString("|")
	for range headers {
		sb.WriteString("---|")
	}
	sb.WriteString("\n")

	// Write rows
	for _, row := range rows {
		sb.WriteString("|")
		for _, cell := range row {
			sb.WriteString(cell)
			sb.WriteString("|")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatAggregateReportAsMarkdown formats the entire aggregate report as Markdown
func (s *Service) FormatAggregateReportAsMarkdown(report *types.FlowAggregateReport) string {
	return formatAggregateReportAsMarkdown(report)
}

// formatAggregateReportAsMarkdown formats the entire aggregate report as Markdown
func formatAggregateReportAsMarkdown(report *types.FlowAggregateReport) string {
	var sb strings.Builder

	// Title and time range
	sb.WriteString("# Flow Logs Aggregate Report\n\n")
	sb.WriteString(fmt.Sprintf("**Time Range:** %s\n\n", report.TimeRange))

	// Traffic Overview
	sb.WriteString("## Traffic Overview\n\n")
	if len(report.TrafficOverview) > 0 {
		headers := []string{"Source", "Source Namespace", "Destination", "Dest Namespace", "Protocol", "Port", "Action", "Packets In/Out", "Bytes In/Out", "Policy"}
		rows := [][]string{}
		for _, entry := range report.TrafficOverview {
			row := []string{
				entry.Source,
				entry.SourceNamespace,
				entry.Destination,
				entry.DestNamespace,
				entry.Protocol,
				fmt.Sprintf("%d", entry.Port),
				entry.Action,
				fmt.Sprintf("%s/%s", entry.PacketsInStr, entry.PacketsOutStr),
				fmt.Sprintf("%s/%s", entry.BytesInStr, entry.BytesOutStr),
				entry.PrimaryPolicy,
			}
			rows = append(rows, row)
		}
		sb.WriteString(buildMarkdownTable(headers, rows))
	} else {
		sb.WriteString("No traffic flows found.\n")
	}
	sb.WriteString("\n")

	// Traffic by Category
	sb.WriteString("## Traffic by Category\n\n")
	if len(report.TrafficByCategory) > 0 {
		headers := []string{"Category", "Count", "Description"}
		rows := [][]string{}
		for _, cat := range report.TrafficByCategory {
			row := []string{
				cat.Category,
				fmt.Sprintf("%d", cat.Count),
				cat.Description,
			}
			rows = append(rows, row)
		}
		sb.WriteString(buildMarkdownTable(headers, rows))
	} else {
		sb.WriteString("No traffic categories identified.\n")
	}
	sb.WriteString("\n")

	// Top Traffic Sources
	sb.WriteString("## Top Traffic Sources\n\n")
	if len(report.TopTrafficSources) > 0 {
		headers := []string{"Source", "Total Flows", "Primary Activity"}
		rows := [][]string{}
		for _, source := range report.TopTrafficSources {
			row := []string{
				source.Name,
				fmt.Sprintf("%d", source.TotalFlows),
				source.PrimaryActivity,
			}
			rows = append(rows, row)
		}
		sb.WriteString(buildMarkdownTable(headers, rows))
	} else {
		sb.WriteString("No traffic sources identified.\n")
	}
	sb.WriteString("\n")

	// Top Traffic Destinations
	sb.WriteString("## Top Traffic Destinations\n\n")
	if len(report.TopTrafficDest) > 0 {
		headers := []string{"Destination", "Total Flows", "Primary Activity"}
		rows := [][]string{}
		for _, dest := range report.TopTrafficDest {
			row := []string{
				dest.Name,
				fmt.Sprintf("%d", dest.TotalFlows),
				dest.PrimaryActivity,
			}
			rows = append(rows, row)
		}
		sb.WriteString(buildMarkdownTable(headers, rows))
	} else {
		sb.WriteString("No traffic destinations identified.\n")
	}
	sb.WriteString("\n")

	// Namespace Activity
	sb.WriteString("## Namespace Activity\n\n")
	if len(report.NamespaceActivity) > 0 {
		headers := []string{"Namespace", "Ingress Flows", "Egress Flows", "Total Traffic Volume"}
		rows := [][]string{}
		for _, ns := range report.NamespaceActivity {
			row := []string{
				ns.Namespace,
				fmt.Sprintf("%d", ns.IngressFlows),
				fmt.Sprintf("%d", ns.EgressFlows),
				ns.TotalTrafficVolume,
			}
			rows = append(rows, row)
		}
		sb.WriteString(buildMarkdownTable(headers, rows))
	} else {
		sb.WriteString("No namespace activity identified.\n")
	}
	sb.WriteString("\n")

	// Security Posture
	sb.WriteString("## Security Posture\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Flows Analyzed**: %d\n", report.SecurityPosture.TotalFlows))
	sb.WriteString(fmt.Sprintf("- **Allowed Flows**: %d (%.1f%%)\n",
		report.SecurityPosture.AllowedFlows,
		report.SecurityPosture.AllowedPercentage))
	sb.WriteString(fmt.Sprintf("- **Denied Flows**: %d (%.1f%%)\n",
		report.SecurityPosture.DeniedFlows,
		report.SecurityPosture.DeniedPercentage))
	sb.WriteString(fmt.Sprintf("- **Active Policies**: %d unique policies",
		report.SecurityPosture.ActivePolicies))

	if len(report.SecurityPosture.UniquePolicyNames) > 0 {
		sb.WriteString(" (")
		sb.WriteString(strings.Join(report.SecurityPosture.UniquePolicyNames, ", "))
		sb.WriteString(")")
	}
	sb.WriteString("\n\n")

	// Additional message based on denied flows
	if report.SecurityPosture.DeniedFlows == 0 {
		sb.WriteString("All traffic is currently allowed - no blocked flows detected in this time window.\n")
	} else {
		sb.WriteString(fmt.Sprintf("⚠️ %d blocked flow(s) detected - review security policies for potential issues.\n",
			report.SecurityPosture.DeniedFlows))
	}

	return sb.String()
}
