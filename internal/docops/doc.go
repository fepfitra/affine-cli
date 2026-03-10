package docops

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/tomohiro-owada/affine-cli/internal/yjs"
)

// GenerateDocID creates a random short doc ID (10 chars, base62-ish).
func GenerateDocID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)[:10]
}

// CreateDoc creates a new empty document in the workspace.
// It updates the workspace root doc's meta.pages and creates the doc's Y.Doc.
func (s *Session) CreateDoc(title string) (string, error) {
	docID := GenerateDocID()

	// 1. Update workspace root doc to register the new doc
	rootEngID, err := s.LoadWorkspaceRoot()
	if err != nil {
		return "", fmt.Errorf("load workspace root: %w", err)
	}

	err = s.PushDocDelta(rootEngID, s.WorkspaceID, func() error {
		script := fmt.Sprintf(`
			(function() {
				var doc = globalThis._docs[%d];
				var meta = doc.getMap("meta");
				var pages = meta.get("pages");
				if (!pages) {
					pages = new Y.Array();
					meta.set("pages", pages);
				}
				var pageMeta = new Y.Map();
				pageMeta.set("id", %q);
				var titleText = new Y.Text();
				titleText.insert(0, %q, {});
				pageMeta.set("title", titleText);
				pageMeta.set("createDate", Date.now());
				pages.push([pageMeta]);
				return "ok";
			})()
		`, rootEngID, docID, title)
		_, err := s.Engine.RunScript(script)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("register doc in workspace: %w", err)
	}

	// 2. Create the doc's own Y.Doc with basic structure
	newEngID, err := s.Engine.NewDoc()
	if err != nil {
		return "", err
	}

	script := fmt.Sprintf(`
		(function() {
			var doc = globalThis._docs[%d];
			var blocks = doc.getMap("blocks");

			var page = new Y.Map();
			page.set("sys:id", "page-root");
			page.set("sys:flavour", "affine:page");
			page.set("sys:version", 2);
			var titleText = new Y.Text();
			titleText.insert(0, %q, {});
			page.set("prop:title", titleText);
			var pageChildren = new Y.Array();
			pageChildren.push(["surface-1", "note-1"]);
			page.set("sys:children", pageChildren);
			blocks.set("page-root", page);

			var surface = new Y.Map();
			surface.set("sys:id", "surface-1");
			surface.set("sys:flavour", "affine:surface");
			surface.set("sys:version", 5);
			surface.set("sys:children", new Y.Array());
			blocks.set("surface-1", surface);

			var note = new Y.Map();
			note.set("sys:id", "note-1");
			note.set("sys:flavour", "affine:note");
			note.set("sys:version", 1);
			note.set("sys:children", new Y.Array());
			blocks.set("note-1", note);

			return "ok";
		})()
	`, newEngID, title)
	_, err = s.Engine.RunScript(script)
	if err != nil {
		return "", fmt.Errorf("create doc structure: %w", err)
	}

	b64, err := s.Engine.EncodeStateAsUpdate(newEngID)
	if err != nil {
		return "", fmt.Errorf("encode doc: %w", err)
	}

	err = s.Client.PushDocUpdate(s.WorkspaceID, docID, b64)
	if err != nil {
		return "", fmt.Errorf("push doc: %w", err)
	}

	s.Engine.FreeDoc(newEngID)
	return docID, nil
}

// ReadDoc reads a document's blocks and returns structured data.
type BlockInfo struct {
	ID       string `json:"id"`
	Flavour  string `json:"flavour"`
	Type     string `json:"type,omitempty"`
	Text     string `json:"text,omitempty"`
	Language string `json:"language,omitempty"`
}

func (s *Session) ReadDoc(docID string) ([]BlockInfo, string, error) {
	engDocID, err := s.LoadDoc(docID)
	if err != nil {
		return nil, "", err
	}
	defer s.Engine.FreeDoc(engDocID)

	blocks, err := s.Engine.ReadBlocks(engDocID)
	if err != nil {
		return nil, "", err
	}

	var result []BlockInfo
	var textParts []string

	for id, b := range blocks {
		flavour, _ := b["sys:flavour"].(string)
		btype, _ := b["sys:type"].(string)
		text, _ := b["prop:text"].(string)
		lang, _ := b["prop:language"].(string)

		if flavour == "affine:page" || flavour == "affine:surface" || flavour == "affine:note" {
			continue
		}

		info := BlockInfo{
			ID:       id,
			Flavour:  flavour,
			Type:     btype,
			Text:     text,
			Language: lang,
		}
		result = append(result, info)
		if text != "" {
			textParts = append(textParts, text)
		}
	}

	plainText := strings.Join(textParts, "\n")
	return result, plainText, nil
}

// DeleteDoc removes a document from the workspace.
func (s *Session) DeleteDoc(docID string) error {
	// Remove from workspace root doc's meta.pages
	rootEngID, err := s.LoadWorkspaceRoot()
	if err != nil {
		return fmt.Errorf("load workspace root: %w", err)
	}

	err = s.PushDocDelta(rootEngID, s.WorkspaceID, func() error {
		script := fmt.Sprintf(`
			(function() {
				var doc = globalThis._docs[%d];
				var meta = doc.getMap("meta");
				var pages = meta.get("pages");
				if (!pages) return "no pages";
				for (var i = 0; i < pages.length; i++) {
					var p = pages.get(i);
					if (p && p.get && p.get("id") === %q) {
						pages.delete(i, 1);
						return "deleted";
					}
				}
				return "not found";
			})()
		`, rootEngID, docID)
		_, err := s.Engine.RunScript(script)
		return err
	})
	if err != nil {
		return fmt.Errorf("remove doc from meta: %w", err)
	}

	// Send delete event
	s.Client.DeleteDoc(s.WorkspaceID, docID)
	return nil
}

// ExportMarkdown exports a document as markdown.
func (s *Session) ExportMarkdown(docID string) (string, error) {
	engDocID, err := s.LoadDoc(docID)
	if err != nil {
		return "", err
	}
	defer s.Engine.FreeDoc(engDocID)

	blocks, err := s.Engine.ReadBlocks(engDocID)
	if err != nil {
		return "", err
	}

	// Get ordered block IDs from the note's children
	script := fmt.Sprintf(`
		(function() {
			var doc = globalThis._docs[%d];
			var blocks = doc.getMap("blocks");
			var order = [];
			// Find note block and get children order
			blocks.forEach(function(block, id) {
				if (!(block instanceof Y.Map)) return;
				var flavour = block.get("sys:flavour");
				if (flavour === "affine:note") {
					var children = block.get("sys:children");
					if (children instanceof Y.Array) {
						children.forEach(function(childId) {
							order.push(childId);
						});
					}
				}
			});
			return JSON.stringify(order);
		})()
	`, engDocID)
	orderJSON, err := s.Engine.RunScript(script)
	if err != nil {
		return "", err
	}

	// Parse order
	var order []string
	if orderJSON != "" && orderJSON != "[]" {
		// Simple JSON array parse
		orderJSON = strings.Trim(orderJSON, "[]")
		for _, item := range strings.Split(orderJSON, ",") {
			item = strings.Trim(item, `" `)
			if item != "" {
				order = append(order, item)
			}
		}
	}

	var md strings.Builder

	renderBlock := func(id string) {
		b, ok := blocks[id]
		if !ok {
			return
		}
		flavour, _ := b["sys:flavour"].(string)
		btype, _ := b["sys:type"].(string)
		text, _ := b["prop:text"].(string)
		lang, _ := b["prop:language"].(string)

		switch flavour {
		case "affine:paragraph":
			switch btype {
			case "h1":
				md.WriteString("# " + text + "\n\n")
			case "h2":
				md.WriteString("## " + text + "\n\n")
			case "h3":
				md.WriteString("### " + text + "\n\n")
			case "h4":
				md.WriteString("#### " + text + "\n\n")
			case "h5":
				md.WriteString("##### " + text + "\n\n")
			case "h6":
				md.WriteString("###### " + text + "\n\n")
			case "quote":
				md.WriteString("> " + text + "\n\n")
			default:
				if text != "" {
					md.WriteString(text + "\n\n")
				}
			}
		case "affine:list":
			switch btype {
			case "bulleted":
				md.WriteString("- " + text + "\n")
			case "numbered":
				md.WriteString("1. " + text + "\n")
			case "todo":
				checked, _ := b["prop:checked"].(bool)
				if checked {
					md.WriteString("- [x] " + text + "\n")
				} else {
					md.WriteString("- [ ] " + text + "\n")
				}
			default:
				md.WriteString("- " + text + "\n")
			}
		case "affine:code":
			md.WriteString("```" + lang + "\n" + text + "\n```\n\n")
		case "affine:divider":
			md.WriteString("---\n\n")
		}
	}

	if len(order) > 0 {
		for _, id := range order {
			renderBlock(id)
		}
	} else {
		for id := range blocks {
			renderBlock(id)
		}
	}

	return md.String(), nil
}

// AppendParagraph appends a single paragraph block to a document.
func (s *Session) AppendParagraph(docID, text, paragraphType string) error {
	if paragraphType == "" {
		paragraphType = "text"
	}
	engDocID, err := s.LoadDoc(docID)
	if err != nil {
		return err
	}

	blockID := "p-" + GenerateDocID()

	return s.PushDocDelta(engDocID, docID, func() error {
		err := s.Engine.CreateFormattedBlock(engDocID, blockID, "affine:paragraph", paragraphType, text)
		if err != nil {
			return err
		}
		// Add children array and add to note
		script := fmt.Sprintf(`
			(function() {
				var doc = globalThis._docs[%d];
				var blocks = doc.getMap("blocks");
				var block = blocks.get(%q);
				block.set("sys:children", new Y.Array());
				// Find note block and append
				blocks.forEach(function(b, id) {
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
		`, engDocID, blockID, blockID)
		_, err = s.Engine.RunScript(script)
		return err
	})
}

// AppendMarkdown parses markdown and appends multiple blocks.
func (s *Session) AppendMarkdown(docID, markdown string) (int, error) {
	engDocID, err := s.LoadDoc(docID)
	if err != nil {
		return 0, err
	}

	lines := parseMarkdownToLines(markdown)
	if len(lines) == 0 {
		return 0, nil
	}

	count := 0
	err = s.PushDocDelta(engDocID, docID, func() error {
		var blockIDs []string
		for _, line := range lines {
			blockID := "b-" + GenerateDocID()

			// Handle table blocks (affine:table)
			if line.flavour == "affine:database" && len(line.tableHeaders) > 0 {
				if err := createTableBlock(s.Engine, engDocID, blockID, line.tableHeaders, line.tableRows); err != nil {
					return err
				}
				blockIDs = append(blockIDs, blockID)
				count++
				continue
			}

			err := s.Engine.CreateFormattedBlock(engDocID, blockID, line.flavour, line.btype, line.text)
			if err != nil {
				return err
			}
			// Set sys:children, language if code
			extra := ""
			if line.language != "" {
				extra = fmt.Sprintf(`block.set("prop:language", %q);`, line.language)
			}
			if line.checked {
				extra += `block.set("prop:checked", true);`
			}
			script := fmt.Sprintf(`
				(function() {
					var doc = globalThis._docs[%d];
					var block = doc.getMap("blocks").get(%q);
					block.set("sys:children", new Y.Array());
					%s
					return "ok";
				})()
			`, engDocID, blockID, extra)
			_, err = s.Engine.RunScript(script)
			if err != nil {
				return err
			}
			blockIDs = append(blockIDs, blockID)
			count++
		}

		// Add all block IDs to note's children
		idsJS := `[`
		for i, id := range blockIDs {
			if i > 0 {
				idsJS += ","
			}
			idsJS += fmt.Sprintf("%q", id)
		}
		idsJS += `]`

		script := fmt.Sprintf(`
			(function() {
				var doc = globalThis._docs[%d];
				var blocks = doc.getMap("blocks");
				var ids = %s;
				blocks.forEach(function(b, bid) {
					if (!(b instanceof Y.Map)) return;
					if (b.get("sys:flavour") === "affine:note") {
						var children = b.get("sys:children");
						if (children instanceof Y.Array) {
							for (var i = 0; i < ids.length; i++) {
								children.push([ids[i]]);
							}
						}
					}
				});
				return "ok";
			})()
		`, engDocID, idsJS)
		_, err := s.Engine.RunScript(script)
		return err
	})
	return count, err
}

// ReplaceWithMarkdown replaces all content blocks with new markdown content.
func (s *Session) ReplaceWithMarkdown(docID, markdown string) (int, error) {
	engDocID, err := s.LoadDoc(docID)
	if err != nil {
		return 0, err
	}

	lines := parseMarkdownToLines(markdown)

	count := 0
	err = s.PushDocDelta(engDocID, docID, func() error {
		// Remove existing content blocks (keep page, surface, note)
		script := fmt.Sprintf(`
			(function() {
				var doc = globalThis._docs[%d];
				var blocks = doc.getMap("blocks");
				var toDelete = [];
				blocks.forEach(function(b, id) {
					if (!(b instanceof Y.Map)) return;
					var f = b.get("sys:flavour");
					if (f !== "affine:page" && f !== "affine:surface" && f !== "affine:note") {
						toDelete.push(id);
					}
				});
				for (var i = 0; i < toDelete.length; i++) {
					blocks.delete(toDelete[i]);
				}
				// Clear note children
				blocks.forEach(function(b, id) {
					if (!(b instanceof Y.Map)) return;
					if (b.get("sys:flavour") === "affine:note") {
						var children = b.get("sys:children");
						if (children instanceof Y.Array) {
							children.delete(0, children.length);
						}
					}
				});
				return toDelete.length;
			})()
		`, engDocID)
		_, err := s.Engine.RunScript(script)
		if err != nil {
			return err
		}

		// Create new blocks
		var blockIDs []string
		for _, line := range lines {
			blockID := "b-" + GenerateDocID()
			err := s.Engine.CreateFormattedBlock(engDocID, blockID, line.flavour, line.btype, line.text)
			if err != nil {
				return err
			}
			extra := ""
			if line.language != "" {
				extra = fmt.Sprintf(`block.set("prop:language", %q);`, line.language)
			}
			if line.checked {
				extra += `block.set("prop:checked", true);`
			}
			script := fmt.Sprintf(`
				(function() {
					var doc = globalThis._docs[%d];
					var block = doc.getMap("blocks").get(%q);
					block.set("sys:children", new Y.Array());
					%s
					return "ok";
				})()
			`, engDocID, blockID, extra)
			_, err = s.Engine.RunScript(script)
			if err != nil {
				return err
			}
			blockIDs = append(blockIDs, blockID)
			count++
		}

		// Add block IDs to note
		idsJS := `[`
		for i, id := range blockIDs {
			if i > 0 {
				idsJS += ","
			}
			idsJS += fmt.Sprintf("%q", id)
		}
		idsJS += `]`

		script = fmt.Sprintf(`
			(function() {
				var doc = globalThis._docs[%d];
				var blocks = doc.getMap("blocks");
				var ids = %s;
				blocks.forEach(function(b, bid) {
					if (!(b instanceof Y.Map)) return;
					if (b.get("sys:flavour") === "affine:note") {
						var children = b.get("sys:children");
						if (children instanceof Y.Array) {
							for (var i = 0; i < ids.length; i++) {
								children.push([ids[i]]);
							}
						}
					}
				});
				return "ok";
			})()
		`, engDocID, idsJS)
		_, err = s.Engine.RunScript(script)
		return err
	})
	return count, err
}

// CreateDocFromMarkdown creates a new doc and fills it with markdown content.
func (s *Session) CreateDocFromMarkdown(title, markdown string) (string, error) {
	docID, err := s.CreateDoc(title)
	if err != nil {
		return "", err
	}

	_, err = s.AppendMarkdown(docID, markdown)
	if err != nil {
		return docID, fmt.Errorf("doc created (%s) but content failed: %w", docID, err)
	}

	return docID, nil
}

// markdownLine represents a parsed markdown line.
type markdownLine struct {
	flavour  string
	btype    string
	text     string
	language string
	checked  bool
	// For table blocks
	tableHeaders []string
	tableRows    [][]string
}

// parseMarkdownToLines parses markdown into block descriptors.
func parseMarkdownToLines(md string) []markdownLine {
	var result []markdownLine
	lines := strings.Split(md, "\n")
	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			i++
			continue
		}

		// Code block
		if strings.HasPrefix(trimmed, "```") {
			lang := strings.TrimPrefix(trimmed, "```")
			lang = strings.TrimSpace(lang)
			var codeLines []string
			i++
			for i < len(lines) {
				if strings.TrimSpace(lines[i]) == "```" {
					i++
					break
				}
				codeLines = append(codeLines, lines[i])
				i++
			}
			result = append(result, markdownLine{
				flavour:  "affine:code",
				btype:    "",
				text:     strings.Join(codeLines, "\n"),
				language: lang,
			})
			continue
		}

		// Markdown table: detect header row with pipes
		if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
			headers := parseTableRow(trimmed)
			// Check if next line is separator (|---|---|)
			if i+1 < len(lines) {
				sepLine := strings.TrimSpace(lines[i+1])
				if isTableSeparator(sepLine) {
					i += 2 // skip header + separator
					var rows [][]string
					for i < len(lines) {
						rowTrimmed := strings.TrimSpace(lines[i])
						if !strings.HasPrefix(rowTrimmed, "|") {
							break
						}
						rows = append(rows, parseTableRow(rowTrimmed))
						i++
					}
					result = append(result, markdownLine{
						flavour:      "affine:database",
						tableHeaders: headers,
						tableRows:    rows,
					})
					continue
				}
			}
		}

		// Divider
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			result = append(result, markdownLine{flavour: "affine:divider"})
			i++
			continue
		}

		// Headings
		if strings.HasPrefix(trimmed, "# ") {
			result = append(result, markdownLine{flavour: "affine:paragraph", btype: "h1", text: strings.TrimPrefix(trimmed, "# ")})
			i++
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			result = append(result, markdownLine{flavour: "affine:paragraph", btype: "h2", text: strings.TrimPrefix(trimmed, "## ")})
			i++
			continue
		}
		if strings.HasPrefix(trimmed, "### ") {
			result = append(result, markdownLine{flavour: "affine:paragraph", btype: "h3", text: strings.TrimPrefix(trimmed, "### ")})
			i++
			continue
		}
		if strings.HasPrefix(trimmed, "#### ") {
			result = append(result, markdownLine{flavour: "affine:paragraph", btype: "h4", text: strings.TrimPrefix(trimmed, "#### ")})
			i++
			continue
		}

		// Blockquote
		if strings.HasPrefix(trimmed, "> ") {
			result = append(result, markdownLine{flavour: "affine:paragraph", btype: "quote", text: strings.TrimPrefix(trimmed, "> ")})
			i++
			continue
		}

		// Todo list
		if strings.HasPrefix(trimmed, "- [x] ") || strings.HasPrefix(trimmed, "- [X] ") {
			result = append(result, markdownLine{flavour: "affine:list", btype: "todo", text: trimmed[6:], checked: true})
			i++
			continue
		}
		if strings.HasPrefix(trimmed, "- [ ] ") {
			result = append(result, markdownLine{flavour: "affine:list", btype: "todo", text: trimmed[6:]})
			i++
			continue
		}

		// Bulleted list
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			result = append(result, markdownLine{flavour: "affine:list", btype: "bulleted", text: trimmed[2:]})
			i++
			continue
		}

		// Numbered list
		if len(trimmed) > 2 {
			dotIdx := strings.Index(trimmed, ". ")
			if dotIdx > 0 && dotIdx <= 3 {
				allDigits := true
				for _, c := range trimmed[:dotIdx] {
					if c < '0' || c > '9' {
						allDigits = false
						break
					}
				}
				if allDigits {
					result = append(result, markdownLine{flavour: "affine:list", btype: "numbered", text: trimmed[dotIdx+2:]})
					i++
					continue
				}
			}
		}

		// Regular paragraph
		result = append(result, markdownLine{flavour: "affine:paragraph", btype: "text", text: trimmed})
		i++
	}
	return result
}

// createTableBlock creates an affine:table block from parsed markdown table data.
// Cell text is stored as Y.Text with inline markdown formatting support.
func createTableBlock(engine *yjs.Engine, engDocID int, blockID string, headers []string, rows [][]string) error {
	// Include header as first row
	allRows := append([][]string{headers}, rows...)
	numRows := len(allRows)
	numCols := len(headers)

	// Generate row/column IDs
	var rowIDs, colIDs []string
	for i := 0; i < numRows; i++ {
		rowIDs = append(rowIDs, "r-"+GenerateDocID())
	}
	for i := 0; i < numCols; i++ {
		colIDs = append(colIDs, "c-"+GenerateDocID())
	}

	// Build flat-key set statements for rows, columns, and cells
	var flatKeys []string
	for i, rid := range rowIDs {
		flatKeys = append(flatKeys,
			fmt.Sprintf(`block.set("prop:rows.%s.rowId", %q);`, rid, rid),
			fmt.Sprintf(`block.set("prop:rows.%s.order", "r%04d");`, rid, i),
		)
	}
	for i, cid := range colIDs {
		flatKeys = append(flatKeys,
			fmt.Sprintf(`block.set("prop:columns.%s.columnId", %q);`, cid, cid),
			fmt.Sprintf(`block.set("prop:columns.%s.order", "c%04d");`, cid, i),
		)
	}

	// Build cell data as JS array for inline markdown parsing
	var cellEntries []string
	for ri, row := range allRows {
		for ci := 0; ci < numCols; ci++ {
			text := ""
			if ci < len(row) {
				text = row[ci]
			}
			cellKey := fmt.Sprintf("prop:cells.%s:%s.text", rowIDs[ri], colIDs[ci])
			cellEntries = append(cellEntries, fmt.Sprintf(`{key:%q,text:%q}`, cellKey, text))
		}
	}

	script := fmt.Sprintf(`
		(function() {
			var doc = globalThis._docs[%d];
			var blocks = doc.getMap("blocks");
			var block = new Y.Map();
			block.set("sys:id", %q);
			block.set("sys:flavour", "affine:table");
			block.set("sys:version", 1);
			block.set("sys:parent", null);
			block.set("sys:children", new Y.Array());

			// Attach block to doc first (goja Y.js needs doc context for shared types)
			blocks.set(%q, block);

			// Set rows/columns as flat keys
			%s

			// Set cells as flat keys with Y.Text values + inline markdown
			var cellData = [%s];
			for (var i = 0; i < cellData.length; i++) {
				var cd = cellData[i];
				var segments = parseInlineMarkdown(cd.text);
				var yText = new Y.Text();
				var pos = 0;
				for (var j = 0; j < segments.length; j++) {
					var seg = segments[j];
					yText.insert(pos, seg.text, seg.attrs || {});
					pos += seg.text.length;
				}
				block.set(cd.key, yText);
			}

			block.set("prop:comments", undefined);
			block.set("prop:textAlign", undefined);
			return "ok";
		})()
	`, engDocID, blockID, blockID,
		strings.Join(flatKeys, "\n\t\t\t"),
		strings.Join(cellEntries, ","))

	_, err := engine.RunScript(script)
	return err
}

// AddTableRow adds a row to an existing affine:table block using flat-key format.
// It reads existing column IDs and row count, then appends a new row.
func (s *Session) AddTableRow(docID, tableBlockID string, cells []string) (string, error) {
	engDocID, err := s.LoadDoc(docID)
	if err != nil {
		return "", err
	}

	rowID := "r-" + GenerateDocID()

	// Build cell texts as JSON array
	var cellJSON []string
	for i, text := range cells {
		cellJSON = append(cellJSON, fmt.Sprintf(`{idx:%d,text:%q}`, i, text))
	}

	err = s.PushDocDelta(engDocID, docID, func() error {
		script := fmt.Sprintf(`
			(function() {
				var doc = globalThis._docs[%d];
				var blocks = doc.getMap("blocks");
				var block = blocks.get(%q);
				if (!block) return "error: table block not found";

				// Collect existing column IDs in order
				var colEntries = [];
				var maxRowOrder = -1;
				for (var key of block.keys()) {
					var cm = key.match(/^prop:columns\.([^.]+)\.order$/);
					if (cm) {
						colEntries.push({id: cm[1], order: block.get(key)});
					}
					var rm = key.match(/^prop:rows\.([^.]+)\.order$/);
					if (rm) {
						var ord = block.get(key);
						var num = parseInt(ord.replace("r",""), 10);
						if (num > maxRowOrder) maxRowOrder = num;
					}
				}
				colEntries.sort(function(a,b) { return a.order.localeCompare(b.order); });
				var newOrder = "r" + String(maxRowOrder + 1).padStart(4, "0");

				// Add new row
				block.set("prop:rows.%s.rowId", %q);
				block.set("prop:rows.%s.order", newOrder);

				// Add cells
				var cellData = [%s];
				for (var i = 0; i < cellData.length; i++) {
					if (i >= colEntries.length) break;
					var colId = colEntries[i].id;
					var segments = parseInlineMarkdown(cellData[i].text);
					var yText = new Y.Text();
					var pos = 0;
					for (var j = 0; j < segments.length; j++) {
						var seg = segments[j];
						yText.insert(pos, seg.text, seg.attrs || {});
						pos += seg.text.length;
					}
					block.set("prop:cells." + %q + ":" + colId + ".text", yText);
				}
				return "ok";
			})()
		`, engDocID, tableBlockID, rowID, rowID, rowID, strings.Join(cellJSON, ","), rowID)
		_, err := s.Engine.RunScript(script)
		return err
	})

	return rowID, err
}

// parseTableRow extracts cell values from a markdown table row like "| a | b | c |"
func parseTableRow(row string) []string {
	row = strings.TrimSpace(row)
	row = strings.Trim(row, "|")
	parts := strings.Split(row, "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

// isTableSeparator checks if a line is a markdown table separator like "|---|---|"
func isTableSeparator(line string) bool {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") {
		return false
	}
	cleaned := strings.ReplaceAll(line, "|", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, ":", "")
	cleaned = strings.TrimSpace(cleaned)
	return cleaned == ""
}
