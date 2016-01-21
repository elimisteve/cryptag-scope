package main

import (
	"log"
	"strings"
	"time"

	"github.com/elimisteve/cryptag/backend"
	"github.com/elimisteve/cryptag/types"
	scopes "launchpad.net/go-unityscopes/v2"
)

const searchCategoryTemplate = `{
  "schema-version": 1,
  "template": {
    "category-layout": "grid",
    "card-size": "small"
  },
  "components": {
    "title": "title",
    "art": "art",
    "subtitle": "username"
  }
}`

// SCOPE ***********************************************************************

type MyScope struct {
	base *scopes.ScopeBase

	dbox *backend.DropboxRemote
}

func (s *MyScope) Preview(result *scopes.Result, metadata *scopes.ActionMetadata, reply *scopes.PreviewReply, cancelled <-chan bool) error {
	layout1col := scopes.NewColumnLayout(1)
	layout2col := scopes.NewColumnLayout(2)
	layout3col := scopes.NewColumnLayout(3)

	// Single column layout
	layout1col.AddColumn("image", "header", "summary", "actions")

	// Two column layout
	layout2col.AddColumn("image")
	layout2col.AddColumn("header", "summary", "actions")

	// Three column layout
	layout3col.AddColumn("image")
	layout3col.AddColumn("header", "summary", "actions")
	layout3col.AddColumn()

	// Register the layouts we just created
	reply.RegisterLayout(layout1col, layout2col, layout3col)

	header := scopes.NewPreviewWidget("header", "header")

	// It has title and a subtitle properties
	header.AddAttributeMapping("title", "title")
	header.AddAttributeMapping("subtitle", "subtitle")

	// Define the image section
	image := scopes.NewPreviewWidget("image", "image")
	// It has a single source property, mapped to the result's art property
	image.AddAttributeMapping("source", "art")

	// Define the summary section
	description := scopes.NewPreviewWidget("summary", "text")
	// It has a text property, mapped to the result's description property
	description.AddAttributeMapping("text", "description")

	// build variant map.
	tuple1 := map[string]interface{}{
		"id":    "open",
		"label": "Open",
		"uri":   "application:///tmp/non-existent.desktop",
	}

	tuple2 := map[string]interface{}{
		"id":    "download",
		"label": "Download",
	}

	tuple3 := map[string]interface{}{
		"id":    "hide",
		"label": "Hide",
	}

	actions := scopes.NewPreviewWidget("actions", "actions")
	actions.AddAttributeValue("actions", []interface{}{tuple1, tuple2, tuple3})

	var scope_data string

	err := metadata.ScopeData(&scope_data)
	if err != nil {
		return err
	}

	if len(scope_data) > 0 {
		extra := scopes.NewPreviewWidget("extra", "text")
		extra.AddAttributeValue("text", "test Text")
		err = reply.PushWidgets(header, image, description, actions, extra)
	} else {
		err = reply.PushWidgets(header, image, description, actions)
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
	s.base = base
}

// RESULTS *********************************************************************

const (
	DEPT_ID_ALL       = "" // TODO: Surely this should change?
	DEPT_ID_NOTES     = "notes"
	DEPT_ID_PASSWORDS = "passwords"
	DEPT_ID_FILES     = "files"
)

var deptToPlaintags = map[string][]string{
	DEPT_ID_ALL:       {"all"}, // not necessary
	DEPT_ID_NOTES:     {"type:text", "type:note"},
	DEPT_ID_PASSWORDS: {"type:text", "type:password"},
	DEPT_ID_FILES:     {"type:file"},
}

func (s *MyScope) AddQueryResults(query *scopes.CannedQuery, reply *scopes.SearchReply, cancelled <-chan bool) error {
	log.Printf("AddQueryResults called...\n")
	log.Printf("DepartmentID: '%v'\n", query.DepartmentID())

	cat := reply.RegisterCategory("category", "Category", "", searchCategoryTemplate)

	result := scopes.NewCategorisedResult(cat)

	// Fetch search results
	queryStr := strings.TrimSpace(query.QueryString())

	var plaintags []string
	if queryStr != "" {
		plaintags = strings.Split(queryStr, " ")
	}

	if query.DepartmentID() != DEPT_ID_ALL {
		plaintags = append(plaintags, deptToPlaintags[query.DepartmentID()]...)
	}

	rows, err := s.dbox.RowsFromPlainTags(plaintags)
	if err != nil {
		return err
	}

	artURL := "https://pbs.twimg.com/profile_images/1117820653/5ttls5.jpg.png"

	for _, row := range rows {
		result.SetURI(types.RowTagWithPrefix(row, "id:"))
		result.SetTitle(strings.Join(row.PlainTags(), ", "))
		result.SetArt(artURL)
		result.Set("decrypted", row.Decrypted())
		if err = reply.Push(result); err != nil {
			return err
		}

		select {
		case <-cancelled:
			log.Printf("Search cancelled; returning\n")
			return nil
		default:
			// Keep going
		}
	}

	return nil
}

// DEPARTMENTS *****************************************************************

func (s *MyScope) CreateDepartments(query *scopes.CannedQuery, metadata *scopes.SearchMetadata, reply *scopes.SearchReply) *scopes.Department {
	root, err := scopes.NewDepartment("", query, "All CrypTag Data")
	if err != nil {
		reply.Error(err)
		return nil
	}

	// "type:note" and "type:text"
	notesDept, err := scopes.NewDepartment("notes", query, "Notes")
	if err != nil {
		reply.Error(err)
	} else {
		root.AddSubdepartment(notesDept)
	}

	// "type:password" and "type:text"
	pwDept, err := scopes.NewDepartment("passwords", query, "Passwords")
	if err != nil {
		reply.Error(err)
	} else {
		root.AddSubdepartment(pwDept)
	}

	// "type:file"
	fileDept, err := scopes.NewDepartment("files", query, "Files")
	if err != nil {
		reply.Error(err)
	} else {
		root.AddSubdepartment(fileDept)
	}

	return root
}

// MAIN ************************************************************************
func main() {
	key := []byte{
	}
	dboxConf := backend.DropboxConfig{
	}

	dbox, err := backend.NewDropboxRemote(key, "doesntmatter", dboxConf)
	if err != nil {
		log.Fatalf("LoadDropboxRemote error: %v\n", err)
	}

	if err := scopes.Run(&MyScope{dbox: dbox}); err != nil {
		log.Fatalln(err)
	}
}
