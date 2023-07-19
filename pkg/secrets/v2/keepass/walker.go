package keepass

import (
	"path"

	keepass "github.com/tobischo/gokeepasslib/v3"
	"github.com/zostay/go-std/slices"
)

// KeepassWalker represents a tool for walking Keepass records.
type KeepassWalker struct {
	groups  []*keepass.Group
	entries []*keepass.Entry

	dirs []string

	currentGroup *keepass.Group
	currentEntry *keepass.Entry

	walkEntries bool
}

// Walker creates an iterator for walking through the Keepass database records.
func (k *Keepass) Walker(walkEntries bool) *KeepassWalker {
	w := &KeepassWalker{
		groups:      make([]*keepass.Group, 0, len(k.db.Content.Root.Groups)),
		entries:     []*keepass.Entry{},
		dirs:        []string{},
		walkEntries: walkEntries,
	}

	w.pushGroups(k.db.Content.Root.Groups)

	return w
}

// pushGroups pushes a pointer to each group onto the open list in reverse
// order.
func (w *KeepassWalker) pushGroups(groups []keepass.Group) {
	for i := len(groups) - 1; i >= 0; i-- {
		w.groups = slices.Push(w.groups, &groups[i])
	}
}

// pushEntries pushes a pointer to each entry onto the open list in reverse
// order.
func (w *KeepassWalker) pushEntries(entries []keepass.Entry) {
	for i := len(entries) - 1; i >= 0; i-- {
		w.entries = slices.Push(w.entries, &entries[i])
	}
}

// Next returns the next record for iteration. If walkEntries was set to true,
// this will return true if another entry is found in the tree. Otherwise, this
// will return false if another group is found in the tree. Returns false if no
// records are left for iteration.
func (w *KeepassWalker) Next() bool {
	if w.walkEntries {
		return w.nextEntry()
	}
	return w.nextGroup()
}

// nextEntry sets the cursor on the next available entry. If no such entry is
// found, this returns false, otherwise returns true.
func (w *KeepassWalker) nextEntry() bool {
	for len(w.entries) == 0 {
		if len(w.groups) == 0 {
			return false
		}

		w.currentGroup, w.groups = slices.Pop(w.groups)

		if len(w.currentGroup.Entries) > 0 {
			w.pushGroups(w.currentGroup.Groups)
			w.pushEntries(w.currentGroup.Entries)
			break
		}
	}

	w.currentEntry, w.entries = slices.Pop(w.entries)
	return true
}

// nextGroup sets the cursor on the next available group. If no such group is
// found, this returns false, otherwise returns true.
func (w *KeepassWalker) nextGroup() bool {
	if len(w.groups) == 0 {
		return false
	}

	w.currentGroup, w.groups = slices.Pop(w.groups)
	w.pushGroups(w.currentGroup.Groups)
	return true
}

// Entry retrieves the current entry to inspect during iteration.
func (w *KeepassWalker) Entry() *keepass.Entry {
	return w.currentEntry
}

// Group retrieves the current group to inspect during iteration.
func (w *KeepassWalker) Group() *keepass.Group {
	return w.currentGroup
}

// Dir retrieves the name of the location of the current group as a path.
func (w *KeepassWalker) Dir() string {
	currentDir := w.dirs[len(w.dirs)-1]
	return path.Join(currentDir, w.currentGroup.Name)
}
