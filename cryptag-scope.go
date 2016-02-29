package main

import (
	"errors"
	"log"
	"os"
	"path"
	"strings"
	"sync"

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

var (
	ErrBadCmdCreate = errors.New("Bad create command")
)

// SCOPE ***********************************************************************

type MyScope struct {
	base *scopes.ScopeBase

	dbox    *backend.WebserverBackend
	fsCache *backend.FileSystem

	cachedTagPairs types.TagPairs

	// Locks accesses to local fs cache (fsCache and tagCursor)
	cacheLock sync.RWMutex

	cacheDir    string
	rowCacheDir string
	tagCacheDir string

	initSuccess bool
}

func (s *MyScope) serverInfo() (baseURL, authToken string) {
	m := map[string]interface{}{}
	s.base.Settings(&m)

	info := strings.SplitN(m["serverInfo"].(string), "#", 2)
	if len(info) < 2 {
		return
	}

	baseURL, authToken = info[0], info[1]

	return
}

func (s *MyScope) cacheTagPairs(pairs types.TagPairs) error {
	var finalErr error
	for _, p := range pairs {
		if _, err := s.fsCache.SaveTagPair(p); err != nil {
			log.Printf("Error caching tag pair `%#v`: %v\n", p, err)
			finalErr = err
		}
	}

	return finalErr
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

	copyClipboard := map[string]interface{}{
		"id":    "copy",
		"label": "Copy",
	}

	// Eventually create a download link
	// Orig:  "uri": "application:///tmp/non-existent.desktop"
	download := map[string]interface{}{
		"id":    "download",
		"label": "Download",
	}

	actions.AddAttributeValue("actions", []interface{}{copyClipboard, download})

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
	if base != nil {
		log.Printf("SetScopeBase: changing s.base from `%+v` to `%+v`\n", s.base, base)
		s.base = base
	} else if s.base == nil {
		log.Printf("FATAL: s.base == nil and base is, too! Returning...\n")
		return
	}

	if s.initSuccess {
		log.Printf("*MyScope already initialized; returning ~early from SetScopeBase...\n")
		return
	}

	// Backend init

	backendPath := path.Join(s.base.ScopeDirectory(), "backends")
	cryptag.BackendPath = backendPath
	log.Printf("cryptag.BackendPath set to `%v`\n", cryptag.BackendPath)

	dbox, err := backend.LoadWebserverBackend(backendPath, "webserver-scope")
	if err != nil { // Assume backend couldn't be found
		log.Printf("Error from LoadWebserverBackend (`%v`), creating new backend\n", err)

		serverBaseURL, authToken := s.serverInfo()

		dbox, err = backend.NewWebserverBackend(nil, "webserver-scope", serverBaseURL, authToken)
		if err != nil {
			log.Fatalf("NewWebserverBackend error: %v\n", err)
		}

		cfg, err := dbox.Config()
		if err != nil {
			log.Fatalf("Error getting backend config: %v\n", err)
		}

		err = cfg.Save(cryptag.BackendPath)
		if err != nil && err != backend.ErrConfigExists {
			log.Fatalf("Error saving backend config to disk: %v\n", err)
		}
	}

	s.dbox = dbox

	// More init

	s.cacheDir = s.base.CacheDirectory()
	cryptag.TrustedBasePath = s.cacheDir
	cryptag.LocalDataPath = s.cacheDir
	log.Printf("cacheDir set to `%v`\n", s.cacheDir)

	s.rowCacheDir = path.Join(s.cacheDir, "rows")
	os.MkdirAll(s.rowCacheDir, 0700)

	s.tagCacheDir = path.Join(s.cacheDir, "tags")
	os.MkdirAll(s.tagCacheDir, 0700)

	log.Printf("*MyScope before loading cached data == `%#v`\n", s)

	//
	// Grab cached data
	//

	s.cacheLock.RLock()
	defer s.cacheLock.RUnlock()

	// Cached tags
	cacheCfg := &backend.Config{
		Name:     "local-cache",
		Key:      s.dbox.Key(),
		Local:    true,
		DataPath: s.cacheDir,
	}
	fsCache, err := backend.NewFileSystem(cacheCfg)
	if err != nil {
		log.Fatalf("LoadOrCreateFileSystem: uh oh: %v\n", err)
	}
	s.fsCache = fsCache

	tagPairs, err := fsCache.AllTagPairs()
	if err != nil {
		log.Printf("Error from fsCache.AllTagPairs: %v\n", err)
	} else {
		s.cachedTagPairs = tagPairs
	}

	s.initSuccess = true
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
	q := query.QueryString()

	log.Printf("DepartmentID: '%v'\n", query.DepartmentID())

	// Split on whitespace
	plaintags := strings.Fields(q)

	if query.DepartmentID() != DEPT_ID_ALL {
		plaintags = append(plaintags, deptToPlaintags[query.DepartmentID()]...)
	}

	go func() {
		// log.Printf("s.dbox.AllTagPairs starting\n")
		// start := time.Now()

		pairs, err := s.dbox.AllTagPairs()
		// log.Printf("s.dbox.AllTagPairs finished in %s\n", time.Since(start))
		if err != nil {
			if strings.Contains(err.Error(), "unexpected HTTP status code 304") {
				// 304 == No new tags; no problem!
				return
			}
			log.Printf("Error getting all pairs: %v", err)
			return // Nothing new to cache
		}

		s.cacheLock.Lock()
		defer s.cacheLock.Unlock()

		if err = s.cacheTagPairs(pairs); err != nil {
			log.Printf("Error from cacheTagPairs: %v\n", err)
		} else {
			log.Printf("Successfully cached all tag pairs to fsCache\n")
		}
	}()

	// If you want passwords, download them, otherwise just get file
	// (Row) metadata
	var rows []*types.Row
	var err error
	if query.DepartmentID() == DEPT_ID_PASSWORDS || query.DepartmentID() == DEPT_ID_NOTES {
		// log.Printf("s.dbox.RowsFromPlainTags(%#v) starting\n", plaintags)
		// start := time.Now()
		rows, err = s.dbox.RowsFromPlainTags(plaintags)
		// log.Printf("s.dbox.RowsFromPlainTags finished in %s\n", time.Since(start))
	} else {
		// log.Printf("s.dbox.ListRows starting\n")
		// start := time.Now()
		rows, err = s.dbox.ListRows(plaintags)
		// log.Printf("s.dbox.ListRows finished in %s\n", time.Since(start))
	}
	if err != nil {
		return err
	}

	if err = addRowsToReply(rows, query, reply, cancelled); err != nil {
		return err
	}

	return nil
}

func addRowsToReply(rows types.Rows, query *scopes.CannedQuery, reply *scopes.SearchReply, cancelled <-chan bool) error {
	// TODO: Create different category for each type of results.
	// E.g., when DepartmentID == "files".
	cat := reply.RegisterCategory("category", query.DepartmentID(), "",
		searchCategoryTemplate)

	result := scopes.NewCategorisedResult(cat)

	for _, row := range rows {
		// rowID := types.RowTagWithPrefix(row, "id:")

		// filepath, err := types.SaveRowAsFile(row, s.rowCacheDir)
		// if err != nil {
		// 	log.Printf("Error saving row %v: %v\n", rowID, err)
		// } else {
		// 	log.Printf("Successfully saved %v to %v\n", rowID, filepath)
		// }

		// "scope://com.canonical.scopes.clickstore?q=" + string(row.Decrypted())
		if query.DepartmentID() == DEPT_ID_PASSWORDS {
			result.SetURI("")
			// Send to self for copy-and-paste'ing!
			result.SetURI("scope://cryptag-scope.elimisteve_cryptag-scope?q=" + string(row.Decrypted()))
		} else {
			result.SetURI("")
		}

		// result.SetURI(filepath)
		// result.SetDndURI(rowID) // I don't know what this does...
		result.SetTitle(rowTitle(row, query.DepartmentID()))
		result.SetArt(rowArt(row))
		result.Set("summary", rowSummary(row))
		result.Set("short_summary", rowShortSummary(row, query.DepartmentID()))
		result.Set("text_content", rowTextContent(row, query.DepartmentID()))

		log.Printf("Pushing result...\n")
		if err := reply.Push(result); err != nil {
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

	// // "type:file"
	// fileDept, err := scopes.NewDepartment(DEPT_ID_FILES, query, "Files")
	// if err != nil {
	// 	reply.Error(err)
	// } else {
	// 	root.AddSubdepartment(fileDept)
	// }

	return root
}

// MAIN ************************************************************************

func main() {
	if err := scopes.Run(&MyScope{}); err != nil {
		log.Fatalln(err)
	}
}
