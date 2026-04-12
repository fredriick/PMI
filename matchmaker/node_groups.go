package matchmaker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type NodeGroups struct {
	mu     sync.RWMutex
	groups map[string]*NodeGroup
	redis  *redis.Client
	ctx    context.Context
}

type NodeGroup struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Tags        map[string]string `json:"tags"`
	Nodes       []string          `json:"nodes"`
	CreatedAt   time.Time         `json:"created_at"`
}

type NodeTags map[string]string

func NewNodeGroups(redisClient *redis.Client) *NodeGroups {
	return &NodeGroups{
		groups: make(map[string]*NodeGroup),
		redis:  redisClient,
		ctx:    context.Background(),
	}
}

func (ng *NodeGroups) CreateGroup(name, description string, tags map[string]string) error {
	ng.mu.Lock()
	defer ng.mu.Unlock()

	if _, exists := ng.groups[name]; exists {
		return fmt.Errorf("group %s already exists", name)
	}

	ng.groups[name] = &NodeGroup{
		Name:        name,
		Description: description,
		Tags:        tags,
		Nodes:       []string{},
		CreatedAt:   time.Now(),
	}

	if ng.redis != nil {
		ng.redis.HSet(ng.ctx, fmt.Sprintf("node_group:%s", name), "description", description)
		ng.redis.Expire(ng.ctx, fmt.Sprintf("node_group:%s", name), 0)
	}

	return nil
}

func (ng *NodeGroups) AddNodeToGroup(groupName, nodeID string) error {
	ng.mu.Lock()
	defer ng.mu.Unlock()

	group, exists := ng.groups[groupName]
	if !exists {
		return fmt.Errorf("group %s not found", groupName)
	}

	for _, n := range group.Nodes {
		if n == nodeID {
			return nil
		}
	}

	group.Nodes = append(group.Nodes, nodeID)

	if ng.redis != nil {
		ng.redis.SAdd(ng.ctx, fmt.Sprintf("group:%s:nodes", groupName), nodeID)
	}

	return nil
}

func (ng *NodeGroups) RemoveNodeFromGroup(groupName, nodeID string) error {
	ng.mu.Lock()
	defer ng.mu.Unlock()

	group, exists := ng.groups[groupName]
	if !exists {
		return fmt.Errorf("group %s not found", groupName)
	}

	for i, n := range group.Nodes {
		if n == nodeID {
			group.Nodes = append(group.Nodes[:i], group.Nodes[i+1:]...)
			break
		}
	}

	if ng.redis != nil {
		ng.redis.SRem(ng.ctx, fmt.Sprintf("group:%s:nodes", groupName), nodeID)
	}

	return nil
}

func (ng *NodeGroups) GetGroupNodes(groupName string) []string {
	ng.mu.RLock()
	defer ng.mu.RUnlock()

	if group, exists := ng.groups[groupName]; exists {
		return group.Nodes
	}

	return []string{}
}

func (ng *NodeGroups) GetGroupsForNode(nodeID string) []string {
	ng.mu.RLock()
	defer ng.mu.RUnlock()

	var result []string
	for name, group := range ng.groups {
		for _, n := range group.Nodes {
			if n == nodeID {
				result = append(result, name)
				break
			}
		}
	}
	return result
}

func (ng *NodeGroups) ListGroups() []*NodeGroup {
	ng.mu.RLock()
	defer ng.mu.RUnlock()

	groups := make([]*NodeGroup, 0, len(ng.groups))
	for _, group := range ng.groups {
		groups = append(groups, group)
	}
	return groups
}

func (ng *NodeGroups) TagNode(nodeID string, tags map[string]string) error {
	if ng.redis != nil {
		for k, v := range tags {
			ng.redis.HSet(ng.ctx, fmt.Sprintf("node_tags:%s", nodeID), k, v)
		}
	}
	return nil
}

func (ng *NodeGroups) GetNodeTags(nodeID string) map[string]string {
	tags := make(map[string]string)

	if ng.redis != nil {
		data, err := ng.redis.HGetAll(ng.ctx, fmt.Sprintf("node_tags:%s", nodeID)).Result()
		if err == nil {
			for k, v := range data {
				tags[k] = v
			}
		}
	}

	return tags
}

func (ng *NodeGroups) SelectByGroup(groupName string, count int) ([]string, error) {
	nodes := ng.GetGroupNodes(groupName)
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes in group %s", groupName)
	}

	if count > len(nodes) {
		count = len(nodes)
	}

	result := make([]string, count)
	for i := 0; i < count; i++ {
		result[i] = nodes[i%len(nodes)]
	}

	return result, nil
}
