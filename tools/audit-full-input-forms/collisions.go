package main

import "sort"

type collisionKey struct {
	Version   int
	Form, SHA string
}
type collisionIdentity struct {
	Index     int    `json:"input_index"`
	ViewID    string `json:"view_id"`
	SourceSHA string `json:"source_sha256"`
	InputSHA  string `json:"input_sha256"`
	Valid     uint8  `json:"valid_masks_bitset"`
}
type collisionGroup struct {
	Members      []collisionIdentity
	Intersection uint8
}
type collisionIndex map[collisionKey]*collisionGroup
type collisionRecord struct {
	Feature      string            `json:"feature_version"`
	Form         string            `json:"form"`
	SHA          string            `json:"feature_sha256"`
	Size         int               `json:"group_size"`
	Intersection uint8             `json:"group_valid_intersection"`
	Conflict     bool              `json:"incompatible_targets"`
	Member       collisionIdentity `json:"member"`
}
type collisionStats struct {
	Groups    int `json:"groups"`
	Conflicts int `json:"incompatible_groups"`
	Members   int `json:"member_rows"`
}

func newCollisionIndex() collisionIndex { return make(collisionIndex) }
func (c collisionIndex) add(r inputRecord) {
	if r.Declined {
		return
	}
	for version, sha := range r.Features {
		key := collisionKey{version, r.Form, sha}
		group := c[key]
		if group == nil {
			group = &collisionGroup{Intersection: 255}
			c[key] = group
		}
		group.Intersection &= r.Valid
		group.Members = append(group.Members, collisionIdentity{r.Index, r.ViewID, r.SourceSHA, r.InputSHA, r.Valid})
	}
}
func (c collisionIndex) records() ([]collisionRecord, collisionStats) {
	keys := make([]collisionKey, 0, len(c))
	for k, g := range c {
		if len(g.Members) > 1 {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.Version != b.Version {
			return a.Version < b.Version
		}
		if a.Form != b.Form {
			return a.Form < b.Form
		}
		return a.SHA < b.SHA
	})
	var records []collisionRecord
	stats := collisionStats{Groups: len(keys)}
	for _, key := range keys {
		g := c[key]
		if g.Intersection == 0 {
			stats.Conflicts++
		}
		version := "positioned-v3"
		if key.Version == 1 {
			version = "bag-v4"
		}
		for _, member := range g.Members {
			records = append(records, collisionRecord{version, key.Form, key.SHA, len(g.Members), g.Intersection, g.Intersection == 0, member})
			stats.Members++
		}
	}
	return records, stats
}
