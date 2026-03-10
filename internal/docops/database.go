package docops

import (
	"encoding/json"
	"fmt"
)

// CreateDatabase creates a new database block in a document with initial columns.
func (s *Session) CreateDatabase(docID, title string, columns []string) (string, []string, error) {
	engDocID, err := s.LoadDoc(docID)
	if err != nil {
		return "", nil, err
	}

	dbBlockID := "db-" + GenerateDocID()
	titleColID := "col-title-" + GenerateDocID()

	var colIDs []string
	colIDs = append(colIDs, titleColID)
	colDefs := ""
	for _, colName := range columns {
		cid := "col-" + GenerateDocID()
		colIDs = append(colIDs, cid)
		colDefs += fmt.Sprintf(`
			var c = new Y.Map();
			c.set("id", %q);
			c.set("name", %q);
			c.set("type", "rich-text");
			c.set("width", 180);
			cols.push([c]);
		`, cid, colName)
	}

	err = s.PushDocDelta(engDocID, docID, func() error {
		// Step 1: Create db block with empty arrays, attach to doc first
		script := fmt.Sprintf(`
			(function() {
				var doc = globalThis._docs[%d];
				var blocks = doc.getMap("blocks");

				var db = new Y.Map();
				db.set("sys:id", %q);
				db.set("sys:flavour", "affine:database");
				db.set("sys:version", 1);
				db.set("sys:parent", null);
				db.set("sys:children", new Y.Array());
				db.set("prop:views", new Y.Array());
				db.set("prop:columns", new Y.Array());
				db.set("prop:cells", new Y.Map());

				// Attach to doc first so shared types work
				blocks.set(%q, db);

				// Now set title (Y.Text needs doc context)
				var titleText = new Y.Text();
				titleText.insert(0, %q, {});
				db.set("prop:title", titleText);

				// Add title column
				var cols = db.get("prop:columns");
				var titleCol = new Y.Map();
				titleCol.set("id", %q);
				titleCol.set("name", "Title");
				titleCol.set("type", "title");
				titleCol.set("width", 260);
				cols.push([titleCol]);

				// Track all column IDs for the view
				var viewColDefs = [];

				%s

				// Build view columns array (each col needs {id, hide, width} in the view)
				var viewCols = new Y.Array();
				cols.forEach(function(c) {
					if (c instanceof Y.Map) {
						var vc = new Y.Map();
						vc.set("id", c.get("id"));
						vc.set("hide", false);
						vc.set("width", c.get("width") || 200);
						viewCols.push([vc]);
					}
				});

				// Create a default table view matching MCP structure
				var views = db.get("prop:views");
				var view = new Y.Map();
				view.set("id", "view-" + Math.random().toString(36).substr(2, 8));
				view.set("name", "Table View");
				view.set("mode", "table");
				view.set("columns", viewCols);
				view.set("filter", {type: "group", op: "and", conditions: []});
				view.set("groupBy", null);
				view.set("sort", null);
				view.set("header", {titleColumn: null, iconColumn: null});
				views.push([view]);

				// Add to note's children
				blocks.forEach(function(b, bid) {
					if (!(b instanceof Y.Map)) return;
					if (b.get("sys:flavour") === "affine:note") {
						var children = b.get("sys:children");
						if (children instanceof Y.Array) {
							children.push([%q]);
						}
					}
				});
				return "ok";
			})()
		`, engDocID, dbBlockID, dbBlockID, title, titleColID, colDefs, dbBlockID)
		_, err := s.Engine.RunScript(script)
		return err
	})

	return dbBlockID, colIDs, err
}

// AddDatabaseColumn adds a column to a database block in a document.
func (s *Session) AddDatabaseColumn(docID, dbBlockID, name, colType string, index int) (string, error) {
	engDocID, err := s.LoadDoc(docID)
	if err != nil {
		return "", err
	}

	colID := "col-" + GenerateDocID()

	err = s.PushDocDelta(engDocID, docID, func() error {
		script := fmt.Sprintf(`
			(function() {
				var doc = globalThis._docs[%d];
				var blocks = doc.getMap("blocks");
				var db = blocks.get(%q);
				if (!db) return "error: database block not found";

				// Get or create columns map
				var columns = db.get("prop:columns");
				if (!columns || !(columns instanceof Y.Array)) {
					columns = new Y.Array();
					db.set("prop:columns", columns);
				}

				var col = new Y.Map();
				col.set("id", %q);
				col.set("name", %q);
				col.set("type", %q);
				col.set("width", 180);

				var idx = %d;
				if (idx >= 0 && idx < columns.length) {
					columns.insert(idx, [col]);
				} else {
					columns.push([col]);
				}
				return %q;
			})()
		`, engDocID, dbBlockID, colID, name, colType, index, colID)
		_, err := s.Engine.RunScript(script)
		return err
	})

	return colID, err
}

// DatabaseRow represents a row with column values.
type DatabaseRow struct {
	RowID string         `json:"row_id"`
	Cells map[string]any `json:"cells"`
}

// AddDatabaseRow adds a row to a database block in a document.
// AFFiNE stores cell values in the database block's prop:cells Y.Map:
//
//	cellsMap[rowBlockId] = Y.Map { [colId]: Y.Map { columnId, value } }
//
// Title column value is stored as prop:text on the row paragraph block.
// Rich-text cells store value as Y.Text.
func (s *Session) AddDatabaseRow(docID, dbBlockID string, cells map[string]string) (string, error) {
	engDocID, err := s.LoadDoc(docID)
	if err != nil {
		return "", err
	}

	rowBlockID := "row-" + GenerateDocID()

	cellsJSON, _ := json.Marshal(cells)

	err = s.PushDocDelta(engDocID, docID, func() error {
		script := fmt.Sprintf(`
			(function() {
				var doc = globalThis._docs[%d];
				var blocks = doc.getMap("blocks");
				var db = blocks.get(%q);
				if (!db) return "error: database block not found";

				// Find the title column ID and column types
				var titleColId = "";
				var colTypes = {};
				var cols = db.get("prop:columns");
				if (cols instanceof Y.Array) {
					cols.forEach(function(c) {
						if (c instanceof Y.Map) {
							var cid = c.get("id");
							var ctype = c.get("type");
							colTypes[cid] = ctype;
							if (ctype === "title") titleColId = cid;
						}
					});
				}

				// Create the row block (paragraph)
				var row = new Y.Map();
				row.set("sys:id", %q);
				row.set("sys:flavour", "affine:paragraph");
				row.set("sys:version", 1);
				row.set("sys:parent", %q);
				row.set("sys:children", new Y.Array());
				row.set("prop:type", "text");

				// Set title column value as prop:text on the row block
				var cellData = JSON.parse(%q);
				if (titleColId && cellData[titleColId]) {
					var titleText = new Y.Text();
					titleText.insert(0, cellData[titleColId], {});
					row.set("prop:text", titleText);
				}

				blocks.set(%q, row);

				// Store non-title cell values in db's prop:cells
				// Structure: cellsMap[rowBlockId] = Y.Map { colId: Y.Map { columnId, value } }
				var cellsMap = db.get("prop:cells");
				if (!(cellsMap instanceof Y.Map)) {
					cellsMap = new Y.Map();
					db.set("prop:cells", cellsMap);
				}

				// IMPORTANT: Attach rowCells to doc FIRST, then populate
				// (goja Y.js can't nest shared types in standalone objects)
				var rowCells = new Y.Map();
				cellsMap.set(%q, rowCells);

				for (var colId in cellData) {
					if (colId === titleColId) continue;
					var cellValue = new Y.Map();
					rowCells.set(colId, cellValue);
					cellValue.set("columnId", colId);
					var ctype = colTypes[colId] || "rich-text";
					if (ctype === "rich-text") {
						var yText = new Y.Text();
						yText.insert(0, cellData[colId], {});
						cellValue.set("value", yText);
					} else if (ctype === "number") {
						cellValue.set("value", Number(cellData[colId]));
					} else if (ctype === "checkbox") {
						cellValue.set("value", cellData[colId] === "true");
					} else {
						cellValue.set("value", cellData[colId]);
					}
				}

				// Add row block ID to database's children
				var children = db.get("sys:children");
				if (!(children instanceof Y.Array)) {
					children = new Y.Array();
					db.set("sys:children", children);
				}
				children.push([%q]);

				return "ok";
			})()
		`, engDocID, dbBlockID, rowBlockID, dbBlockID, string(cellsJSON), rowBlockID, rowBlockID, rowBlockID)
		_, err := s.Engine.RunScript(script)
		return err
	})

	return rowBlockID, err
}
