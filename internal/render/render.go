package render

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"mindmap/internal/model"
)

const (
	FormatMermaid = "mermaid"
	FormatCSV     = "csv"
	FormatTSV     = "tsv"
	FormatJSON    = "json"
	FormatTree    = "tree"
)

var separatedHeader = []string{
	"org",
	"workload",
	"environment",
	"src_project",
	"src_interconnect",
	"mapped",
	"src_region",
	"src_state",
	"dst_project",
	"dst_region",
	"dst_vlan_attachment",
	"dst_vlan_attachment_state",
	"dst_vlan_attachment_vlanid",
	"dst_cloud_router",
	"dst_cloud_router_state",
	"dst_cloud_router_interface",
	"dst_cloud_router_interface_ip",
	"remote_bgp_peer",
	"remote_bgp_peer_ip",
	"bgp_peering_status",
}

func Render(report model.Report, format string) ([]byte, string, error) {
	switch format {
	case "", FormatMermaid:
		return renderMermaid(report), "mmd", nil
	case FormatCSV:
		data, err := renderSeparated(report, ',')
		return data, "csv", err
	case FormatTSV:
		data, err := renderSeparated(report, '\t')
		return data, "tsv", err
	case FormatJSON:
		data, err := renderJSON(report)
		return data, "json", err
	case FormatTree:
		return renderTree(report), "tree.txt", nil
	default:
		return nil, "", fmt.Errorf("unsupported output format %q", format)
	}
}

func renderSeparated(report model.Report, delimiter rune) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Comma = delimiter
	if err := writer.Write(separatedHeader); err != nil {
		return nil, err
	}
	for _, item := range report.Items {
		record := []string{
			report.Selectors.Org,
			report.Selectors.Workload,
			report.Selectors.Environment,
			item.SrcProject,
			item.SrcInterconnect,
			fmt.Sprintf("%t", item.Mapped),
			item.SrcRegion,
			item.SrcState,
			item.DstProject,
			item.DstRegion,
			item.DstVLANAttachment,
			item.DstVLANAttachmentState,
			item.DstVLANAttachmentVLANID,
			item.DstCloudRouter,
			item.DstCloudRouterState,
			item.DstCloudRouterInterface,
			item.DstCloudRouterInterfaceIP,
			item.RemoteBGPPeer,
			item.RemoteBGPPeerIP,
			item.BGPPeeringStatus,
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buf.Bytes(), writer.Error()
}

func renderJSON(report model.Report) ([]byte, error) {
	return json.MarshalIndent(buildJSONReport(report), "", "  ")
}

func renderTree(report model.Report) []byte {
	var b strings.Builder
	workload := valueOrUnknown(report.Selectors.Workload)
	environment := valueOrUnknown(report.Selectors.Environment)
	srcProject := valueOrUnknown(report.SourceProject)
	interconnects := groupInterconnects(report)

	fmt.Fprintf(&b, "org: %s\n", valueOrUnknown(report.Selectors.Org))
	fmt.Fprintf(&b, "`-- workload: %s\n", workload)
	fmt.Fprintf(&b, "    `-- environment: %s\n", environment)
	fmt.Fprintf(&b, "        `-- src_project: %s\n", srcProject)
	for idx, interconnect := range interconnects {
		drawTreeInterconnect(&b, interconnect, "            ", idx == len(interconnects)-1)
	}
	return []byte(b.String())
}

func renderMermaid(report model.Report) []byte {
	var b strings.Builder
	b.WriteString("flowchart LR\n")

	orgID := mermaidID("org-" + report.Selectors.Org)
	workloadID := mermaidID("workload-" + report.Selectors.Org + "-" + report.Selectors.Workload)
	environmentID := mermaidID("environment-" + report.Selectors.Org + "-" + report.Selectors.Workload + "-" + report.Selectors.Environment)
	srcID := mermaidID("src-" + report.SourceProject)

	fmt.Fprintf(&b, "    %s[%q]\n", orgID, "org: "+valueOrUnknown(report.Selectors.Org))
	fmt.Fprintf(&b, "    %s[%q]\n", workloadID, "workload: "+valueOrUnknown(report.Selectors.Workload))
	fmt.Fprintf(&b, "    %s[%q]\n", environmentID, "environment: "+valueOrUnknown(report.Selectors.Environment))
	fmt.Fprintf(&b, "    %s[%q]\n", srcID, "src_project: "+valueOrUnknown(report.SourceProject))
	fmt.Fprintf(&b, "    %s --> %s\n", orgID, workloadID)
	fmt.Fprintf(&b, "    %s --> %s\n", workloadID, environmentID)
	fmt.Fprintf(&b, "    %s --> %s\n", environmentID, srcID)

	seen := make(map[string]struct{})
	for _, item := range report.Items {
		interconnectID := mermaidID("ic-" + item.SrcInterconnect)
		linkIfMissing(&b, seen, srcID, interconnectID, interconnectLabel(item))

		dstProjectID := mermaidID("dst-project-" + item.SrcInterconnect + "-" + item.DstProject)
		linkIfMissing(&b, seen, interconnectID, dstProjectID, destinationProjectLabel(item))
		if !item.Mapped {
			unmappedID := mermaidID("unmapped-" + item.SrcInterconnect + "-" + item.DstProject)
			linkIfMissing(&b, seen, dstProjectID, unmappedID, "unmapped")
			continue
		}

		dstRegionID := mermaidID("dst-region-" + item.SrcInterconnect + "-" + item.DstProject + "-" + item.DstRegion)
		attachmentID := mermaidID("attachment-" + item.SrcInterconnect + "-" + item.DstProject + "-" + item.DstRegion + "-" + item.DstVLANAttachment)
		routerID := mermaidID("router-" + item.SrcInterconnect + "-" + item.DstProject + "-" + item.DstRegion + "-" + item.DstVLANAttachment + "-" + item.DstCloudRouter)
		interfaceID := mermaidID("interface-" + item.SrcInterconnect + "-" + item.DstProject + "-" + item.DstRegion + "-" + item.DstVLANAttachment + "-" + item.DstCloudRouter + "-" + item.DstCloudRouterInterface)
		peerID := mermaidID("peer-" + item.SrcInterconnect + "-" + item.DstProject + "-" + item.DstRegion + "-" + item.DstVLANAttachment + "-" + item.DstCloudRouter + "-" + item.DstCloudRouterInterface + "-" + item.RemoteBGPPeer + "-" + item.RemoteBGPPeerIP)

		linkIfMissing(&b, seen, dstProjectID, dstRegionID, destinationRegionLabel(item))
		linkIfMissing(&b, seen, dstRegionID, attachmentID, attachmentLabel(item))
		linkIfMissing(&b, seen, attachmentID, routerID, routerLabel(item))
		if hasInterface(item) {
			linkIfMissing(&b, seen, routerID, interfaceID, interfaceLabel(item))
		}
		if hasPeer(item) {
			parentID := routerID
			if hasInterface(item) {
				parentID = interfaceID
			}
			linkIfMissing(&b, seen, parentID, peerID, peerLabel(item))
		}
	}
	return []byte(b.String())
}

type jsonReport struct {
	Type string      `json:"type"`
	Org  jsonOrgNode `json:"org"`
}

type jsonOrgNode struct {
	Name      string             `json:"name"`
	Workloads []jsonWorkloadNode `json:"workloads,omitempty"`
}

type jsonWorkloadNode struct {
	Name         string                `json:"name"`
	Environments []jsonEnvironmentNode `json:"environments,omitempty"`
}

type jsonEnvironmentNode struct {
	Name        string           `json:"name"`
	SrcProjects []jsonSourceNode `json:"src_projects,omitempty"`
}

type jsonSourceNode struct {
	SrcProject      string                 `json:"src_project"`
	SrcInterconnect []jsonInterconnectNode `json:"src_interconnects,omitempty"`
}

type jsonInterconnectNode struct {
	SrcInterconnect string                `json:"src_interconnect"`
	Mapped          bool                  `json:"mapped"`
	SrcRegion       string                `json:"src_region"`
	SrcState        string                `json:"src_state"`
	DstProjects     []jsonDestinationNode `json:"dst_projects,omitempty"`
}

type jsonDestinationNode struct {
	DstProject string           `json:"dst_project"`
	Mapped     bool             `json:"mapped"`
	DstRegions []jsonRegionNode `json:"dst_regions,omitempty"`
}

type jsonRegionNode struct {
	DstRegion          string               `json:"dst_region"`
	DstVLANAttachments []jsonAttachmentNode `json:"dst_vlan_attachments,omitempty"`
}

type jsonAttachmentNode struct {
	DstVLANAttachment         string `json:"dst_vlan_attachment"`
	DstVLANAttachmentState    string `json:"dst_vlan_attachment_state"`
	DstVLANAttachmentVLANID   string `json:"dst_vlan_attachment_vlanid"`
	DstCloudRouter            string `json:"dst_cloud_router"`
	DstCloudRouterState       string `json:"dst_cloud_router_state"`
	DstCloudRouterInterface   string `json:"dst_cloud_router_interface"`
	DstCloudRouterInterfaceIP string `json:"dst_cloud_router_interface_ip"`
	RemoteBGPPeer             string `json:"remote_bgp_peer"`
	RemoteBGPPeerIP           string `json:"remote_bgp_peer_ip"`
	BGPPeeringStatus          string `json:"bgp_peering_status"`
}

type interconnectGroup struct {
	SrcInterconnect string
	Mapped          bool
	SrcRegion       string
	SrcState        string
	DstProjects     []destinationGroup
}

type destinationGroup struct {
	DstProject string
	Mapped     bool
	DstRegions []regionGroup
}

type regionGroup struct {
	DstRegion          string
	DstVLANAttachments []attachmentGroup
}

type attachmentGroup struct {
	DstVLANAttachment         string
	DstVLANAttachmentState    string
	DstVLANAttachmentVLANID   string
	DstCloudRouter            string
	DstCloudRouterState       string
	DstCloudRouterInterface   string
	DstCloudRouterInterfaceIP string
	RemoteBGPPeer             string
	RemoteBGPPeerIP           string
	BGPPeeringStatus          string
}

func buildJSONReport(report model.Report) jsonReport {
	interconnects := groupInterconnects(report)
	return jsonReport{
		Type: report.Type,
		Org: jsonOrgNode{
			Name: valueOrUnknown(report.Selectors.Org),
			Workloads: []jsonWorkloadNode{{
				Name: valueOrUnknown(report.Selectors.Workload),
				Environments: []jsonEnvironmentNode{{
					Name: valueOrUnknown(report.Selectors.Environment),
					SrcProjects: []jsonSourceNode{{
						SrcProject:      valueOrUnknown(report.SourceProject),
						SrcInterconnect: buildJSONInterconnects(interconnects),
					}},
				}},
			}},
		},
	}
}

func buildJSONInterconnects(groups []interconnectGroup) []jsonInterconnectNode {
	result := make([]jsonInterconnectNode, 0, len(groups))
	for _, interconnect := range groups {
		node := jsonInterconnectNode{
			SrcInterconnect: valueOrUnknown(interconnect.SrcInterconnect),
			Mapped:          interconnect.Mapped,
			SrcRegion:       valueOrUnknown(interconnect.SrcRegion),
			SrcState:        valueOrUnknown(interconnect.SrcState),
		}
		for _, dst := range interconnect.DstProjects {
			dstNode := jsonDestinationNode{
				DstProject: valueOrUnknown(dst.DstProject),
				Mapped:     dst.Mapped,
			}
			for _, region := range dst.DstRegions {
				regionNode := jsonRegionNode{
					DstRegion: valueOrUnknown(region.DstRegion),
				}
				for _, attachment := range region.DstVLANAttachments {
					regionNode.DstVLANAttachments = append(regionNode.DstVLANAttachments, jsonAttachmentNode{
						DstVLANAttachment:         valueOrUnknown(attachment.DstVLANAttachment),
						DstVLANAttachmentState:    valueOrUnknown(attachment.DstVLANAttachmentState),
						DstVLANAttachmentVLANID:   valueOrUnknown(attachment.DstVLANAttachmentVLANID),
						DstCloudRouter:            valueOrUnknown(attachment.DstCloudRouter),
						DstCloudRouterState:       valueOrUnknown(attachment.DstCloudRouterState),
						DstCloudRouterInterface:   valueOrUnknown(attachment.DstCloudRouterInterface),
						DstCloudRouterInterfaceIP: valueOrUnknown(attachment.DstCloudRouterInterfaceIP),
						RemoteBGPPeer:             valueOrUnknown(attachment.RemoteBGPPeer),
						RemoteBGPPeerIP:           valueOrUnknown(attachment.RemoteBGPPeerIP),
						BGPPeeringStatus:          valueOrUnknown(attachment.BGPPeeringStatus),
					})
				}
				dstNode.DstRegions = append(dstNode.DstRegions, regionNode)
			}
			node.DstProjects = append(node.DstProjects, dstNode)
		}
		result = append(result, node)
	}
	return result
}

func groupInterconnects(report model.Report) []interconnectGroup {
	grouped := make(map[string][]model.MappingItem)
	var names []string
	for _, item := range report.Items {
		if _, ok := grouped[item.SrcInterconnect]; !ok {
			names = append(names, item.SrcInterconnect)
		}
		grouped[item.SrcInterconnect] = append(grouped[item.SrcInterconnect], item)
	}
	sort.Strings(names)

	result := make([]interconnectGroup, 0, len(names))
	for _, name := range names {
		items := grouped[name]
		if len(items) == 0 {
			continue
		}
		group := interconnectGroup{
			SrcInterconnect: name,
			SrcRegion:       items[0].SrcRegion,
			SrcState:        items[0].SrcState,
			DstProjects:     groupDestinations(items),
		}
		for _, item := range items {
			if item.Mapped {
				group.Mapped = true
				break
			}
		}
		result = append(result, group)
	}
	return result
}

func groupDestinations(items []model.MappingItem) []destinationGroup {
	grouped := make(map[string][]model.MappingItem)
	var names []string
	for _, item := range items {
		if _, ok := grouped[item.DstProject]; !ok {
			names = append(names, item.DstProject)
		}
		grouped[item.DstProject] = append(grouped[item.DstProject], item)
	}
	sort.Strings(names)

	result := make([]destinationGroup, 0, len(names))
	for _, name := range names {
		dstItems := grouped[name]
		dst := destinationGroup{DstProject: name}
		for _, item := range dstItems {
			if item.Mapped {
				dst.Mapped = true
				break
			}
		}
		if dst.Mapped {
			dst.DstRegions = groupRegions(dstItems)
		}
		result = append(result, dst)
	}
	return result
}

func groupRegions(items []model.MappingItem) []regionGroup {
	grouped := make(map[string][]model.MappingItem)
	var names []string
	for _, item := range items {
		if !item.Mapped {
			continue
		}
		if _, ok := grouped[item.DstRegion]; !ok {
			names = append(names, item.DstRegion)
		}
		grouped[item.DstRegion] = append(grouped[item.DstRegion], item)
	}
	sort.Strings(names)

	result := make([]regionGroup, 0, len(names))
	for _, name := range names {
		region := regionGroup{DstRegion: name}
		for _, item := range grouped[name] {
			region.DstVLANAttachments = append(region.DstVLANAttachments, attachmentGroup{
				DstVLANAttachment:         item.DstVLANAttachment,
				DstVLANAttachmentState:    item.DstVLANAttachmentState,
				DstVLANAttachmentVLANID:   item.DstVLANAttachmentVLANID,
				DstCloudRouter:            item.DstCloudRouter,
				DstCloudRouterState:       item.DstCloudRouterState,
				DstCloudRouterInterface:   item.DstCloudRouterInterface,
				DstCloudRouterInterfaceIP: item.DstCloudRouterInterfaceIP,
				RemoteBGPPeer:             item.RemoteBGPPeer,
				RemoteBGPPeerIP:           item.RemoteBGPPeerIP,
				BGPPeeringStatus:          item.BGPPeeringStatus,
			})
		}
		result = append(result, region)
	}
	return result
}

func drawTreeInterconnect(b *strings.Builder, interconnect interconnectGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(
		b,
		"%s%s src_interconnect: %s [mapped: %t, src_region: %s, src_state: %s]\n",
		indent,
		prefix,
		valueOrUnknown(interconnect.SrcInterconnect),
		interconnect.Mapped,
		valueOrUnknown(interconnect.SrcRegion),
		valueOrUnknown(interconnect.SrcState),
	)
	for idx, dst := range interconnect.DstProjects {
		drawTreeDestination(b, dst, childIndent, idx == len(interconnect.DstProjects)-1)
	}
}

func drawTreeDestination(b *strings.Builder, dst destinationGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(b, "%s%s dst_project: %s [mapped: %t]\n", indent, prefix, valueOrUnknown(dst.DstProject), dst.Mapped)
	if !dst.Mapped {
		fmt.Fprintf(b, "%s`-- unmapped\n", childIndent)
		return
	}
	for idx, region := range dst.DstRegions {
		drawTreeRegion(b, region, childIndent, idx == len(dst.DstRegions)-1)
	}
}

func drawTreeRegion(b *strings.Builder, region regionGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(b, "%s%s dst_region: %s\n", indent, prefix, valueOrUnknown(region.DstRegion))
	for idx, attachment := range region.DstVLANAttachments {
		drawTreeAttachment(b, attachment, childIndent, idx == len(region.DstVLANAttachments)-1)
	}
}

func drawTreeAttachment(b *strings.Builder, attachment attachmentGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(
		b,
		"%s%s dst_vlan_attachment: %s [dst_vlan_attachment_state: %s, dst_vlan_attachment_vlanid: %s]\n",
		indent,
		prefix,
		valueOrUnknown(attachment.DstVLANAttachment),
		valueOrUnknown(attachment.DstVLANAttachmentState),
		valueOrUnknown(attachment.DstVLANAttachmentVLANID),
	)
	fmt.Fprintf(
		b,
		"%s`-- dst_cloud_router: %s [dst_cloud_router_state: %s]\n",
		childIndent,
		valueOrUnknown(attachment.DstCloudRouter),
		valueOrUnknown(attachment.DstCloudRouterState),
	)
	fmt.Fprintf(
		b,
		"%s    `-- dst_cloud_router_interface: %s [dst_cloud_router_interface_ip: %s]\n",
		childIndent,
		valueOrUnknown(attachment.DstCloudRouterInterface),
		valueOrUnknown(attachment.DstCloudRouterInterfaceIP),
	)
	fmt.Fprintf(
		b,
		"%s        `-- remote_bgp_peer: %s [remote_bgp_peer_ip: %s, bgp_peering_status: %s]\n",
		childIndent,
		valueOrUnknown(attachment.RemoteBGPPeer),
		valueOrUnknown(attachment.RemoteBGPPeerIP),
		valueOrUnknown(attachment.BGPPeeringStatus),
	)
}

func interconnectLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"src_interconnect: %s\\nsrc_region: %s\\nsrc_state: %s",
		valueOrUnknown(item.SrcInterconnect),
		valueOrUnknown(item.SrcRegion),
		valueOrUnknown(item.SrcState),
	)
}

func destinationProjectLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"dst_project: %s\\nmapped: %t",
		valueOrUnknown(item.DstProject),
		item.Mapped,
	)
}

func destinationRegionLabel(item model.MappingItem) string {
	return "dst_region: " + valueOrUnknown(item.DstRegion)
}

func attachmentLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"dst_vlan_attachment: %s\\ndst_vlan_attachment_state: %s\\ndst_vlan_attachment_vlanid: %s",
		valueOrUnknown(item.DstVLANAttachment),
		valueOrUnknown(item.DstVLANAttachmentState),
		valueOrUnknown(item.DstVLANAttachmentVLANID),
	)
}

func routerLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"dst_cloud_router: %s\\ndst_cloud_router_state: %s",
		valueOrUnknown(item.DstCloudRouter),
		valueOrUnknown(item.DstCloudRouterState),
	)
}

func interfaceLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"dst_cloud_router_interface: %s\\ndst_cloud_router_interface_ip: %s",
		valueOrUnknown(item.DstCloudRouterInterface),
		valueOrUnknown(item.DstCloudRouterInterfaceIP),
	)
}

func peerLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"remote_bgp_peer: %s\\nremote_bgp_peer_ip: %s\\nbgp_peering_status: %s",
		valueOrUnknown(item.RemoteBGPPeer),
		valueOrUnknown(item.RemoteBGPPeerIP),
		valueOrUnknown(item.BGPPeeringStatus),
	)
}

func hasInterface(item model.MappingItem) bool {
	return strings.TrimSpace(item.DstCloudRouterInterface) != "" || strings.TrimSpace(item.DstCloudRouterInterfaceIP) != ""
}

func hasPeer(item model.MappingItem) bool {
	return strings.TrimSpace(item.RemoteBGPPeer) != "" || strings.TrimSpace(item.RemoteBGPPeerIP) != "" || strings.TrimSpace(item.BGPPeeringStatus) != ""
}

func linkIfMissing(b *strings.Builder, seen map[string]struct{}, parentID, childID, childLabel string) {
	nodeKey := "node:" + childID
	if _, ok := seen[nodeKey]; !ok {
		fmt.Fprintf(b, "    %s[%q]\n", childID, childLabel)
		seen[nodeKey] = struct{}{}
	}
	edgeKey := "edge:" + parentID + "->" + childID
	if _, ok := seen[edgeKey]; ok {
		return
	}
	fmt.Fprintf(b, "    %s --> %s\n", parentID, childID)
	seen[edgeKey] = struct{}{}
}

func mermaidID(value string) string {
	value = strings.ToLower(value)
	replacer := strings.NewReplacer(
		"-", "_",
		".", "_",
		"/", "_",
		":", "_",
		" ", "_",
	)
	return replacer.Replace(value)
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}
