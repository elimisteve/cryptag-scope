package main

import (
	"log"
	"path"
	"strings"
	"time"

	"github.com/elimisteve/cryptag"
	"github.com/elimisteve/cryptag/backend"
	"github.com/elimisteve/cryptag/types"
	scopes "launchpad.net/go-unityscopes/v2"
)

const searchCategoryTemplate = `{
  "schema-version": 1,
  "template": {
    "category-layout": "grid",
    "card-size": "small",
    "card-layout": "horizontal"
  },
  "components": {
    "title": "title",
    "art": "art",
    "summary": "short_summary"
  }
}`

// SCOPE ***********************************************************************

type MyScope struct {
	base *scopes.ScopeBase

	dbox *backend.DropboxRemote

	tagCursor string // For efficient diffs when fetching new tags

	cacheDir    string
	rowCacheDir string
	tagCacheDir string
}

func (s *MyScope) Preview(result *scopes.Result, metadata *scopes.ActionMetadata, reply *scopes.PreviewReply, cancelled <-chan bool) error {
	log.Printf("Preview called...\n")

	hints := map[string]interface{}{}
	if err := metadata.Hints(&hints); err != nil {
		log.Printf("Hint unmarshal error: %v\n", err)
	} else {
		log.Printf("metadata hints: %+v\n", hints)
	}

	layout1col := scopes.NewColumnLayout(1)
	// layout2col := scopes.NewColumnLayout(2)

	// Single column layout
	layout1col.AddColumn("image", "header", "summary", "content", "actions")

	// // Two column layout
	// layout2col.AddColumn("image")
	// layout2col.AddColumn("header", "summary", "actions")

	// Register the layouts we just created
	// reply.RegisterLayout(layout1col, layout2col)
	reply.RegisterLayout(layout1col)

	//
	// "image"
	//
	image := scopes.NewPreviewWidget("image", "image")
	image.AddAttributeMapping("source", "art")

	//
	// "header"
	//
	header := scopes.NewPreviewWidget("header", "header")
	header.AddAttributeMapping("title", "title")
	header.AddAttributeMapping("text", "short_summary")

	//
	// "summary"
	//
	summary := scopes.NewPreviewWidget("summary", "text")
	// It has a text property, mapped to the result's description property
	summary.AddAttributeMapping("text", "summary")

	//
	// "content"
	//
	content := scopes.NewPreviewWidget("content", "text")
	content.AddAttributeMapping("text", "text_content")

	//
	// "actions"
	//
	actions := scopes.NewPreviewWidget("actions", "actions")

	open := map[string]interface{}{
		"id":    "open",
		"label": "Open",
	}
	// Eventually create a download link
	// Orig:  "uri": "application:///tmp/non-existent.desktop"
	download := map[string]interface{}{
		"id":    "download",
		"label": "Download",
	}

	actions.AddAttributeValue("actions", []interface{}{open, download})

	var scope_data string

	err := metadata.ScopeData(&scope_data)
	if err != nil {
		return err
	}

	if len(scope_data) > 0 {
		extra := scopes.NewPreviewWidget("extra", "text")
		extra.AddAttributeValue("text", "test Text")
		err = reply.PushWidgets(header, image, summary, content, actions, extra)
	} else {
		err = reply.PushWidgets(header, image, summary, content, actions)
	}

	return err
}

func (s *MyScope) Search(query *scopes.CannedQuery, metadata *scopes.SearchMetadata, reply *scopes.SearchReply, cancelled <-chan bool) error {
	rootDept := s.CreateDepartments(query, metadata, reply)
	reply.RegisterDepartments(rootDept)

	// test incompatible features in RTM version of libunity-scopes
	filter1 := scopes.NewOptionSelectorFilter("f1", "Options", false)
	var filterState scopes.FilterState
	// for RTM version of libunity-scopes we should see a log message
	reply.PushFilters([]scopes.Filter{filter1}, filterState)

	return s.AddQueryResults(query, reply, cancelled)
}

func (s *MyScope) SetScopeBase(base *scopes.ScopeBase) {
	log.Printf("SetScopeBase: changing s.base from `%+v` to `%+v`\n", s.base, base)
	s.base = base

	// Dropbox init

	dbox, err := backend.LoadDropboxRemote(
		path.Join(s.base.ScopeDirectory(), "backends"),
		"dropbox-remote-cryptag-scope.json",
	)
	if err != nil {
		log.Fatalf("LoadDropboxRemote error: %v\n", err)
	}

	s.dbox = dbox

	// More init

	cacheDir := s.base.CacheDirectory()
	cryptag.TrustedBasePath = cacheDir
	cryptag.BackendPath = path.Join(cryptag.TrustedBasePath, "backends")

	s.tagCursor = ""

	s.cacheDir = cacheDir
	s.rowCacheDir = path.Join(cacheDir, "rows")
	s.tagCacheDir = path.Join(cacheDir, "tags")

	log.Printf("*MyScope == `%#v`\n", s)
}

// RESULTS *********************************************************************

const (
	DEPT_ID_ALL       = "" // TODO: Surely this should change?
	DEPT_ID_NOTES     = "Notes"
	DEPT_ID_PASSWORDS = "Passwords"
	DEPT_ID_FILES     = "Files"
)

var deptToPlaintags = map[string][]string{
	// DEPT_ID_ALL:       {"all"}, // not necessary
	DEPT_ID_NOTES:     {"type:text", "type:note"},
	DEPT_ID_PASSWORDS: {"type:text", "type:password"},
	DEPT_ID_FILES:     {"type:file"},
}

func (s *MyScope) AddQueryResults(query *scopes.CannedQuery, reply *scopes.SearchReply, cancelled <-chan bool) error {
	log.Printf("DepartmentID: '%v'\n", query.DepartmentID())

	// TODO: Create different category for each type of results.
	// E.g., when DepartmentID == "files".
	cat := reply.RegisterCategory("category", query.DepartmentID(), "",
		searchCategoryTemplate)

	result := scopes.NewCategorisedResult(cat)

	// Split on whitespace
	plaintags := strings.Fields(query.QueryString())

	if query.DepartmentID() != DEPT_ID_ALL {
		plaintags = append(plaintags, deptToPlaintags[query.DepartmentID()]...)
	}

	rows, err := s.dbox.RowsFromPlainTags(plaintags)
	if err != nil {
		return err
	}

	for _, row := range rows {
		rowID := types.RowTagWithPrefix(row, "id:")

		filepath, err := types.SaveRowAsFile(row, s.rowCacheDir)
		if err != nil {
			log.Printf("Error saving row %v: %v\n", rowID, err)
		} else {
			log.Printf("Successfully saved %v to %v\n", rowID, filepath)
		}

		result.SetURI(filepath)
		result.SetDndURI(rowID) // I don't know what this does...
		result.SetTitle(rowTitle(row, query.DepartmentID()))
		result.SetArt(rowArt(row))
		result.Set("summary", rowSummary(row))
		result.Set("short_summary", rowShortSummary(row, query.DepartmentID()))

		text := rowTextContent(row, query.DepartmentID())
		result.Set("text_content", text)

		if err = reply.Push(result); err != nil {
			return err
		}

		select {
		case <-cancelled:
			log.Println("Search cancelled; returning")
			return nil
		default:
			// Keep going
		}
	}

	return nil
}

func rowTitle(row *types.Row, deptID string) string {
	if t := types.RowTagWithPrefix(row, "filename:", "title:", "site:"); t != "" {
		return t
	}
	if deptID == DEPT_ID_PASSWORDS {
		return strings.Join(humanReadableTags(row.PlainTags()), ", ")
	}
	return "(No Title)"
}

func rowSummary(row *types.Row) string {
	tags := row.PlainTags()
	return bold("All Tags: ") + strings.Join(tags, ", ")
}

func rowShortSummary(row *types.Row, deptID string) string {
	if deptID == DEPT_ID_PASSWORDS {
		return ""
	}
	return strings.Join(humanReadableTags(row.PlainTags()), ", ")
}

func rowArt(row *types.Row) string {
	var rowTypes []string
	for _, t := range row.PlainTags() {
		if strings.HasPrefix(t, "type:") && t != "type:text" && t != "type:file" {
			rowTypes = append(rowTypes, strings.TrimPrefix(t, "type:"))
		}
	}
	return "https://placeholdit.imgix.net/~text?txtsize=100&txt=" + strings.Join(rowTypes, ", ") + "&w=300&h=300"

}

func rowTextContent(row *types.Row, deptID string) string {
	// Only attach row content if it's text
	if !row.HasPlainTag("type:text") {
		return ""
	}

	text := string(row.Decrypted())

	switch deptID {
	case DEPT_ID_PASSWORDS:
		return text
	case DEPT_ID_NOTES:
		return bold("Note: ") + text
	}
	return bold("Content: ") + text
}

func bold(text string) string {
	return `<b>` + text + `</b>`
}

func humanReadableTags(tags []string) []string {
	var good []string
	for _, t := range tags {
		if t == "all" || hasAnyPrefix(t, "type:", "id:", "app:", "filename:") {
			continue
		}
		good = append(good, t)
	}
	return good
}

func hasAnyPrefix(s string, strs ...string) bool {
	for i := range strs {
		if strings.HasPrefix(s, strs[i]) {
			return true
		}
	}
	return false
}

// DEPARTMENTS *****************************************************************

func (s *MyScope) CreateDepartments(query *scopes.CannedQuery, metadata *scopes.SearchMetadata, reply *scopes.SearchReply) *scopes.Department {
	// TODO: Check metadata.IsAggregated(). Also use
	// metadata.SetAggregatedKeywords(keywords []string) at some point
	// so this Scope's content can be found by other apps.
	//
	// Paraphrased from
	// https://developer.ubuntu.com/api/scopes/cpp/development/index/
	// -- "You can use the IsAggregated() method... in order to ensure
	// that an appropriate set of results are returned when queried by
	// an aggregator."

	root, err := scopes.NewDepartment(DEPT_ID_ALL, query, "All CrypTag Data")
	if err != nil {
		reply.Error(err)
		return nil
	}

	// "type:note" and "type:text"
	notesDept, err := scopes.NewDepartment(DEPT_ID_NOTES, query, "Notes")
	if err != nil {
		reply.Error(err)
	} else {
		root.AddSubdepartment(notesDept)
	}

	// "type:password" and "type:text"
	pwDept, err := scopes.NewDepartment(DEPT_ID_PASSWORDS, query, "Passwords")
	if err != nil {
		reply.Error(err)
	} else {
		root.AddSubdepartment(pwDept)
	}

	// "type:file"
	fileDept, err := scopes.NewDepartment(DEPT_ID_FILES, query, "Files")
	if err != nil {
		reply.Error(err)
	} else {
		root.AddSubdepartment(fileDept)
	}

	return root
}

// MAIN ************************************************************************

func main() {
	go ticker()

	if err := scopes.Run(&MyScope{}); err != nil {
		log.Fatalln(err)
	}
}

func ticker() {
	for t := range time.Tick(time.Second) {
		log.Printf("%v\n", t)
	}
}
