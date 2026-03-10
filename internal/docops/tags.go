package docops

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TagInfo represents a tag in the workspace.
type TagInfo struct {
	ID    string `json:"id"`
	Value string `json:"value"`
	Color string `json:"color,omitempty"`
	Docs  int    `json:"doc_count,omitempty"`
}

// ListTags returns all tags in the workspace.
func (s *Session) ListTags() ([]TagInfo, error) {
	rootEngID, err := s.LoadWorkspaceRoot()
	if err != nil {
		return nil, err
	}
	defer s.Engine.FreeDoc(rootEngID)

	script := fmt.Sprintf(`
		(function() {
			var doc = globalThis._docs[%d];
			var meta = doc.getMap("meta");
			var properties = meta.get("properties");
			if (!properties || !(properties instanceof Y.Map)) return JSON.stringify([]);
			var tags = properties.get("tags");
			if (!tags || !(tags instanceof Y.Map)) return JSON.stringify([]);
			var options = tags.get("options");
			if (!options || !(options instanceof Y.Array)) return JSON.stringify([]);
			var result = [];
			options.forEach(function(opt) {
				if (opt instanceof Y.Map) {
					result.push({
						id: opt.get("id") || "",
						value: opt.get("value") || "",
						color: opt.get("color") || ""
					});
				}
			});
			return JSON.stringify(result);
		})()
	`, rootEngID)

	val, err := s.Engine.RunScript(script)
	if err != nil {
		return nil, err
	}

	var tags []TagInfo
	if err := json.Unmarshal([]byte(val), &tags); err != nil {
		return nil, err
	}

	// Count docs per tag
	docCountScript := fmt.Sprintf(`
		(function() {
			var doc = globalThis._docs[%d];
			var meta = doc.getMap("meta");
			var pages = meta.get("pages");
			if (!pages) return JSON.stringify({});
			var counts = {};
			pages.forEach(function(p) {
				if (!(p instanceof Y.Map)) return;
				var pageTags = p.get("tags");
				if (pageTags instanceof Y.Array) {
					pageTags.forEach(function(t) {
						counts[t] = (counts[t] || 0) + 1;
					});
				}
			});
			return JSON.stringify(counts);
		})()
	`, rootEngID)

	countsVal, err := s.Engine.RunScript(docCountScript)
	if err == nil {
		var counts map[string]float64
		if json.Unmarshal([]byte(countsVal), &counts) == nil {
			for i := range tags {
				if c, ok := counts[tags[i].ID]; ok {
					tags[i].Docs = int(c)
				}
			}
		}
	}

	return tags, nil
}

// CreateTag creates a new tag in the workspace.
func (s *Session) CreateTag(name string) (string, error) {
	rootEngID, err := s.LoadWorkspaceRoot()
	if err != nil {
		return "", err
	}

	tagID := GenerateDocID()

	err = s.PushDocDelta(rootEngID, s.WorkspaceID, func() error {
		script := fmt.Sprintf(`
			(function() {
				var doc = globalThis._docs[%d];
				var meta = doc.getMap("meta");
				var properties = meta.get("properties");
				if (!properties || !(properties instanceof Y.Map)) {
					properties = new Y.Map();
					meta.set("properties", properties);
				}
				var tags = properties.get("tags");
				if (!tags || !(tags instanceof Y.Map)) {
					tags = new Y.Map();
					properties.set("tags", tags);
				}
				var options = tags.get("options");
				if (!options || !(options instanceof Y.Array)) {
					options = new Y.Array();
					tags.set("options", options);
				}
				// Check for duplicate
				var exists = false;
				options.forEach(function(opt) {
					if (opt instanceof Y.Map && opt.get("value") === %q) exists = true;
				});
				if (exists) return "exists";

				var opt = new Y.Map();
				opt.set("id", %q);
				opt.set("value", %q);
				opt.set("color", "blue");
				options.push([opt]);
				return "created";
			})()
		`, rootEngID, name, tagID, name)
		_, err := s.Engine.RunScript(script)
		return err
	})

	return tagID, err
}

// AddTagToDoc adds a tag to a document.
func (s *Session) AddTagToDoc(docID, tagName string) error {
	rootEngID, err := s.LoadWorkspaceRoot()
	if err != nil {
		return err
	}

	// First find the tag ID by name
	script := fmt.Sprintf(`
		(function() {
			var doc = globalThis._docs[%d];
			var meta = doc.getMap("meta");
			var properties = meta.get("properties");
			if (!properties) return "";
			var tags = properties.get("tags");
			if (!tags) return "";
			var options = tags.get("options");
			if (!options) return "";
			var tagId = "";
			options.forEach(function(opt) {
				if (opt instanceof Y.Map && opt.get("value") === %q) tagId = opt.get("id");
			});
			return tagId;
		})()
	`, rootEngID, tagName)

	tagID, err := s.Engine.RunScript(script)
	if err != nil {
		return err
	}
	if tagID == "" {
		return fmt.Errorf("tag %q not found", tagName)
	}

	return s.PushDocDelta(rootEngID, s.WorkspaceID, func() error {
		script := fmt.Sprintf(`
			(function() {
				var doc = globalThis._docs[%d];
				var meta = doc.getMap("meta");
				var pages = meta.get("pages");
				if (!pages) return "no pages";
				for (var i = 0; i < pages.length; i++) {
					var p = pages.get(i);
					if (!(p instanceof Y.Map)) continue;
					if (p.get("id") === %q) {
						var tags = p.get("tags");
						if (!(tags instanceof Y.Array)) {
							tags = new Y.Array();
							p.set("tags", tags);
						}
						// Check if already tagged
						var found = false;
						tags.forEach(function(t) { if (t === %q) found = true; });
						if (!found) tags.push([%q]);
						return "added";
					}
				}
				return "doc not found in meta";
			})()
		`, rootEngID, docID, tagID, tagID)
		_, err := s.Engine.RunScript(script)
		return err
	})
}

// RemoveTagFromDoc removes a tag from a document.
func (s *Session) RemoveTagFromDoc(docID, tagName string) error {
	rootEngID, err := s.LoadWorkspaceRoot()
	if err != nil {
		return err
	}

	// Find tag ID
	script := fmt.Sprintf(`
		(function() {
			var doc = globalThis._docs[%d];
			var meta = doc.getMap("meta");
			var properties = meta.get("properties");
			if (!properties) return "";
			var tags = properties.get("tags");
			if (!tags) return "";
			var options = tags.get("options");
			if (!options) return "";
			var tagId = "";
			options.forEach(function(opt) {
				if (opt instanceof Y.Map && opt.get("value") === %q) tagId = opt.get("id");
			});
			return tagId;
		})()
	`, rootEngID, tagName)

	tagID, err := s.Engine.RunScript(script)
	if err != nil {
		return err
	}
	if tagID == "" {
		return fmt.Errorf("tag %q not found", tagName)
	}

	return s.PushDocDelta(rootEngID, s.WorkspaceID, func() error {
		script := fmt.Sprintf(`
			(function() {
				var doc = globalThis._docs[%d];
				var meta = doc.getMap("meta");
				var pages = meta.get("pages");
				if (!pages) return "no pages";
				for (var i = 0; i < pages.length; i++) {
					var p = pages.get(i);
					if (!(p instanceof Y.Map)) continue;
					if (p.get("id") === %q) {
						var tags = p.get("tags");
						if (!(tags instanceof Y.Array)) return "no tags";
						for (var j = 0; j < tags.length; j++) {
							if (tags.get(j) === %q) {
								tags.delete(j, 1);
								return "removed";
							}
						}
						return "tag not on doc";
					}
				}
				return "doc not found";
			})()
		`, rootEngID, docID, tagID)
		_, err := s.Engine.RunScript(script)
		return err
	})
}

// ListDocsByTag returns doc IDs that have a given tag.
func (s *Session) ListDocsByTag(tagName string) ([]string, error) {
	rootEngID, err := s.LoadWorkspaceRoot()
	if err != nil {
		return nil, err
	}
	defer s.Engine.FreeDoc(rootEngID)

	script := fmt.Sprintf(`
		(function() {
			var doc = globalThis._docs[%d];
			var meta = doc.getMap("meta");
			// Find tag ID
			var tagId = "";
			var properties = meta.get("properties");
			if (properties instanceof Y.Map) {
				var tags = properties.get("tags");
				if (tags instanceof Y.Map) {
					var options = tags.get("options");
					if (options instanceof Y.Array) {
						options.forEach(function(opt) {
							if (opt instanceof Y.Map && opt.get("value") === %q) tagId = opt.get("id");
						});
					}
				}
			}
			if (!tagId) return JSON.stringify([]);

			var pages = meta.get("pages");
			if (!pages) return JSON.stringify([]);
			var result = [];
			pages.forEach(function(p) {
				if (!(p instanceof Y.Map)) return;
				var pageTags = p.get("tags");
				if (pageTags instanceof Y.Array) {
					var found = false;
					pageTags.forEach(function(t) { if (t === tagId) found = true; });
					if (found) result.push(p.get("id"));
				}
			});
			return JSON.stringify(result);
		})()
	`, rootEngID, tagName)

	val, err := s.Engine.RunScript(script)
	if err != nil {
		return nil, err
	}

	val = strings.Trim(val, "[]")
	if val == "" {
		return nil, nil
	}

	var docIDs []string
	for _, item := range strings.Split(val, ",") {
		item = strings.Trim(item, `" `)
		if item != "" {
			docIDs = append(docIDs, item)
		}
	}
	return docIDs, nil
}
