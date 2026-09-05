package matchmaker

import (
	"testing"
)

func TestNodeGroups_CreateGroup(t *testing.T) {
	ng := NewNodeGroups(nil)

	if err := ng.CreateGroup("web-servers", "Production web servers", nil); err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}

	err := ng.CreateGroup("web-servers", "Duplicate", nil)
	if err == nil {
		t.Error("expected error for duplicate group")
	}

	tags := map[string]string{"env": "prod", "region": "us-east"}
	if err := ng.CreateGroup("api-servers", "API servers", tags); err != nil {
		t.Fatalf("CreateGroup failed: %v", err)
	}
}

func TestNodeGroups_AddAndRemoveNode(t *testing.T) {
	ng := NewNodeGroups(nil)

	if err := ng.CreateGroup("group1", "Test group", nil); err != nil {
		t.Fatal(err)
	}

	if err := ng.AddNodeToGroup("group1", "node1"); err != nil {
		t.Fatalf("AddNodeToGroup failed: %v", err)
	}
	if err := ng.AddNodeToGroup("group1", "node2"); err != nil {
		t.Fatalf("AddNodeToGroup failed: %v", err)
	}

	nodes := ng.GetGroupNodes("group1")
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes in group, got %d", len(nodes))
	}

	err := ng.AddNodeToGroup("nonexistent", "node1")
	if err == nil {
		t.Error("expected error for nonexistent group")
	}

	if err := ng.RemoveNodeFromGroup("group1", "node1"); err != nil {
		t.Fatalf("RemoveNodeFromGroup failed: %v", err)
	}

	nodes = ng.GetGroupNodes("group1")
	if len(nodes) != 1 {
		t.Errorf("expected 1 node after removal, got %d", len(nodes))
	}

	err = ng.RemoveNodeFromGroup("nonexistent", "node1")
	if err == nil {
		t.Error("expected error for removing from nonexistent group")
	}
}

func TestNodeGroups_GetGroupsForNode(t *testing.T) {
	ng := NewNodeGroups(nil)

	ng.CreateGroup("group1", "Test", nil)
	ng.CreateGroup("group2", "Test", nil)

	ng.AddNodeToGroup("group1", "node1")
	ng.AddNodeToGroup("group2", "node1")
	ng.AddNodeToGroup("group1", "node2")

	groups := ng.GetGroupsForNode("node1")
	if len(groups) != 2 {
		t.Errorf("expected 2 groups for node1, got %d", len(groups))
	}

	groups = ng.GetGroupsForNode("node2")
	if len(groups) != 1 {
		t.Errorf("expected 1 group for node2, got %d", len(groups))
	}

	groups = ng.GetGroupsForNode("nonexistent")
	if len(groups) != 0 {
		t.Errorf("expected 0 groups for nonexistent node, got %d", len(groups))
	}
}

func TestNodeGroups_ListGroups(t *testing.T) {
	ng := NewNodeGroups(nil)

	ng.CreateGroup("group1", "Test 1", nil)
	ng.CreateGroup("group2", "Test 2", nil)

	groups := ng.ListGroups()
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}

func TestNodeGroups_AddDuplicateNode(t *testing.T) {
	ng := NewNodeGroups(nil)

	ng.CreateGroup("group1", "Test", nil)

	ng.AddNodeToGroup("group1", "node1")
	err := ng.AddNodeToGroup("group1", "node1")

	if err != nil {
		t.Errorf("expected no error adding duplicate node, got %v", err)
	}

	nodes := ng.GetGroupNodes("group1")
	if len(nodes) != 1 {
		t.Errorf("expected 1 node (no duplicates), got %d", len(nodes))
	}
}

func TestNodeGroups_SelectByGroup(t *testing.T) {
	ng := NewNodeGroups(nil)

	ng.CreateGroup("group1", "Test", nil)
	ng.AddNodeToGroup("group1", "node1")
	ng.AddNodeToGroup("group1", "node2")
	ng.AddNodeToGroup("group1", "node3")

	result, err := ng.SelectByGroup("group1", 2)
	if err != nil {
		t.Fatalf("SelectByGroup failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(result))
	}

	result, err = ng.SelectByGroup("group1", 0)
	if err != nil {
		t.Fatalf("SelectByGroup with count 0 failed: %v", err)
	}

	_, err = ng.SelectByGroup("nonexistent", 2)
	if err == nil {
		t.Error("expected error for nonexistent group")
	}
}

func TestNodeGroups_RemoveNodeFromGroup_NonExistentNode(t *testing.T) {
	ng := NewNodeGroups(nil)

	ng.CreateGroup("group1", "Test", nil)
	ng.AddNodeToGroup("group1", "node1")

	err := ng.RemoveNodeFromGroup("group1", "nonexistent-node")
	if err != nil {
		t.Errorf("expected no error removing nonexistent node, got %v", err)
	}
}

func TestNodeGroups_GetGroupNodes_Nonexistent(t *testing.T) {
	ng := NewNodeGroups(nil)
	nodes := ng.GetGroupNodes("nonexistent")
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes for nonexistent group, got %d", len(nodes))
	}
}
