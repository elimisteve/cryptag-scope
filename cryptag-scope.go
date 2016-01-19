package main

import (
	"log"

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

	// Three cokumn layout
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

	return s.AddQueryResults(reply, query.QueryString())
}

func (s *MyScope) SetScopeBase(base *scopes.ScopeBase) {
	s.base = base
}

// RESULTS *********************************************************************

func (s *MyScope) AddQueryResults(reply *scopes.SearchReply, query string) error {
	cat := reply.RegisterCategory("category", "Category", "", searchCategoryTemplate)

	result := scopes.NewCategorisedResult(cat)
	result.SetURI("http://localhost/" + query)
	result.SetDndURI("http://localhost_dnduri" + query)
	result.SetTitle("TEST" + query)
	result.SetArt("https://pbs.twimg.com/profile_images/1117820653/5ttls5.jpg.png")
	result.Set("test_value_bool", true)
	result.Set("test_value_string", "test_value"+query)
	result.Set("test_value_int", 1999)
	result.Set("test_value_float", 1.999)
	if err := reply.Push(result); err != nil {
		return err
	}

	result.SetURI("http://localhost2/" + query)
	result.SetDndURI("http://localhost_dnduri2" + query)
	result.SetTitle("TEST2")
	result.SetArt("https://pbs.twimg.com/profile_images/1117820653/5ttls5.jpg.png")
	result.Set("test_value_bool", false)
	result.Set("test_value_string", "test_value2"+query)
	result.Set("test_value_int", 2000)
	result.Set("test_value_float", 2.100)

	// add a variant map value
	m := make(map[string]interface{})
	m["value1"] = 1
	m["value2"] = "string_value"
	result.Set("test_value_map", m)

	// add a variant array value
	l := make([]interface{}, 0)
	l = append(l, 1999)
	l = append(l, "string_value")
	result.Set("test_value_array", l)
	if err := reply.Push(result); err != nil {
		return err
	}

	return nil
}

// DEPARTMENTS *****************************************************************

func (s *MyScope) CreateDepartments(query *scopes.CannedQuery, metadata *scopes.SearchMetadata, reply *scopes.SearchReply) *scopes.Department {
	root, err := scopes.NewDepartment("", query, "All Encrypted Files and Data")
	if err != nil {
		reply.Error(err)
		return nil
	}
	root.SetAlternateLabel("All Encrypted Files and Data")

	audioDept, err := scopes.NewDepartment("audio", query, "Music & Audio")
	if err != nil {
		reply.Error(err)
	} else {
		root.AddSubdepartment(audioDept)
	}

	notesDept, err := scopes.NewDepartment("notes", query, "Notes")
	if err != nil {
		reply.Error(err)
	} else {
		root.AddSubdepartment(notesDept)
	}

	pwDept, err := scopes.NewDepartment("passwords", query, "Passwords")
	if err != nil {
		reply.Error(err)
	} else {
		root.AddSubdepartment(pwDept)
	}

	return root
}

// MAIN ************************************************************************

func main() {
	if err := scopes.Run(&MyScope{}); err != nil {
		log.Fatalln(err)
	}
}
