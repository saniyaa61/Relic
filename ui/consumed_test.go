package ui

import (
	"testing"
)

// "See all" opens the folder's page, which shows every entry of the group
// and closes itself if the category is deleted.
func TestConsumedSeeAll(t *testing.T) {
	a, _ := libApp(t)
	p := &consumedPage{}
	a.Push(p)
	frame(a)
	cats := a.Lib.ConsumedByCategory(a.timeIndex())
	if len(cats) == 0 {
		t.Fatal("fixture: no time")
	}
	k := folderKey{cats[0].Category.ID, cats[0].Folders[0].Name}
	p.click(k).Click()
	frame(a)
	f, ok := a.top().(*consumedFolderPage)
	if !ok || f.key != k {
		t.Fatalf("See all opened %T", a.top())
	}
	if es, _ := a.Lib.ConsumedFolder(k.cat, k.folder); len(es) < len(cats[0].Folders[0].Entries) {
		t.Errorf("folder page has %d entries, group %d", len(es), len(cats[0].Folders[0].Entries))
	}
	frame(a)

	// Search on the folder page, then on the root.
	f.search.Toggle(a)
	f.search.input.Editor.SetText("entry")
	frame(a)
	a.Back() // closes search
	if a.top() != f {
		t.Fatal("back should close the search first")
	}

	if err := a.Lib.DeleteCategory(k.cat); err != nil {
		t.Fatal(err)
	}
	frame(a)
	if a.top() != p {
		t.Errorf("folder page should close when its category is gone; top is %T", a.top())
	}
}

func TestConsumedEmpty(t *testing.T) {
	a, _ := testApp(t, nil)
	a.Push(&consumedPage{})
	frame(a)
}
