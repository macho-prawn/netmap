package render

import (
	"strings"
	"testing"

	"mindmap/internal/model"
)

func sampleReport() model.Report {
	return model.Report{
		Type:               "interconnect",
		SourceProject:      "src",
		DestinationProject: "",
		Selectors: model.Selectors{
			Org:         "dbc",
			Workload:    "native",
			Environment: "dev",
		},
		Items: []model.MappingItem{
			{
				SrcProject:                "src",
				SrcInterconnect:           "ic-1",
				Mapped:                    true,
				SrcRegion:                 "global",
				SrcState:                  "ACTIVE",
				DstProject:                "dst-a",
				DstRegion:                 "us-central1",
				DstVLANAttachment:         "attachment-1",
				DstVLANAttachmentState:    "ACTIVE",
				DstVLANAttachmentVLANID:   "101",
				DstCloudRouter:            "router-1",
				DstCloudRouterState:       "unknown",
				DstCloudRouterInterface:   "if-1",
				DstCloudRouterInterfaceIP: "169.254.1.1",
				RemoteBGPPeer:             "peer-1",
				RemoteBGPPeerIP:           "169.254.1.2",
				BGPPeeringStatus:          "UP",
			},
			{
				SrcProject:      "src",
				SrcInterconnect: "ic-1",
				Mapped:          false,
				SrcRegion:       "global",
				SrcState:        "ACTIVE",
				DstProject:      "dst-b",
			},
		},
	}
}

func TestRenderCSV(t *testing.T) {
	data, ext, err := Render(sampleReport(), FormatCSV)
	if err != nil {
		t.Fatalf("render csv: %v", err)
	}
	if ext != "csv" {
		t.Fatalf("expected csv extension, got %q", ext)
	}
	content := string(data)
	if !strings.Contains(content, "org,workload,environment,src_project,src_interconnect,mapped,src_region,src_state,dst_project,dst_region") {
		t.Fatalf("unexpected csv header order: %s", content)
	}
	if !strings.Contains(content, "dst_cloud_router_interface,dst_cloud_router_interface_ip,remote_bgp_peer,remote_bgp_peer_ip,bgp_peering_status") {
		t.Fatalf("unexpected csv tail column order: %s", content)
	}
}

func TestRenderJSON(t *testing.T) {
	data, ext, err := Render(sampleReport(), FormatJSON)
	if err != nil {
		t.Fatalf("render json: %v", err)
	}
	if ext != "json" {
		t.Fatalf("expected json extension, got %q", ext)
	}
	content := string(data)
	if !strings.Contains(content, `"org": {`) || !strings.Contains(content, `"workloads"`) || !strings.Contains(content, `"environments"`) {
		t.Fatalf("unexpected json output: %s", content)
	}
	if !strings.Contains(content, `"src_interconnects"`) || !strings.Contains(content, `"dst_projects"`) || !strings.Contains(content, `"dst_regions"`) {
		t.Fatalf("expected hierarchical destination data, got: %s", content)
	}
}

func TestRenderTree(t *testing.T) {
	data, ext, err := Render(sampleReport(), FormatTree)
	if err != nil {
		t.Fatalf("render tree: %v", err)
	}
	if ext != "tree.txt" {
		t.Fatalf("expected tree extension, got %q", ext)
	}
	content := string(data)
	if !strings.Contains(content, "org: dbc\n`-- workload: native\n    `-- environment: dev\n        `-- src_project: src") {
		t.Fatalf("unexpected tree root: %s", content)
	}
	if !strings.Contains(content, "dst_vlan_attachment: attachment-1 [dst_vlan_attachment_state: ACTIVE, dst_vlan_attachment_vlanid: 101]") || !strings.Contains(content, "dst_project: dst-b [mapped: false]") {
		t.Fatalf("unexpected tree output: %s", content)
	}
}

func TestRenderMermaid(t *testing.T) {
	data, ext, err := Render(sampleReport(), FormatMermaid)
	if err != nil {
		t.Fatalf("render mermaid: %v", err)
	}
	if ext != "mmd" {
		t.Fatalf("expected mmd extension, got %q", ext)
	}
	content := string(data)
	if !strings.Contains(content, "flowchart LR") {
		t.Fatalf("expected flowchart output, got %s", content)
	}
	if !strings.Contains(content, "org: dbc") || !strings.Contains(content, "workload: native") || !strings.Contains(content, "environment: dev") {
		t.Fatalf("expected selector hierarchy, got %s", content)
	}
	if !strings.Contains(content, "dst_region: us-central1") || !strings.Contains(content, "dst_vlan_attachment: attachment-1") || !strings.Contains(content, "remote_bgp_peer: peer-1") {
		t.Fatalf("expected destination fanout details, got %s", content)
	}
	if strings.Contains(content, "root((GCP") {
		t.Fatalf("expected minimal mermaid output, got %s", content)
	}
}
