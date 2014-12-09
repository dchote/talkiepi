package barnard

import (
	"github.com/layeh/barnard/uiterm"
	"github.com/layeh/gumble/gumble"
)

type TreeItem struct {
	User    *gumble.User
	Channel *gumble.Channel
}

func (ti TreeItem) String() string {
	if ti.User != nil {
		return ti.User.Name()
	}
	if ti.Channel != nil {
		return ti.Channel.Name()
	}
	return ""
}

func (ti TreeItem) TreeItemStyle(fg, bg uiterm.Attribute, active bool) (uiterm.Attribute, uiterm.Attribute) {
	if ti.Channel != nil {
		fg |= uiterm.AttrBold
	}
	if active {
		bg |= uiterm.AttrReverse
	}
	return fg, bg
}

func (b *Barnard) TreeItemSelect(ui *uiterm.Ui, tree *uiterm.Tree, item uiterm.TreeItem) {
	treeItem := item.(TreeItem)
	if treeItem.Channel != nil {
		b.Client.Self().Move(treeItem.Channel)
	}
}

func (b *Barnard) TreeItem(item uiterm.TreeItem) []uiterm.TreeItem {
	var treeItem TreeItem
	if ti, ok := item.(TreeItem); !ok {
		root := b.Client.Channels()[0]
		if root == nil {
			return nil
		}
		return []uiterm.TreeItem{
			TreeItem{
				Channel: root,
			},
		}
	} else {
		treeItem = ti
	}

	if treeItem.User != nil {
		return nil
	}

	users := []uiterm.TreeItem{}
	for _, user := range treeItem.Channel.Users() {
		users = append(users, TreeItem{
			User: user,
		})
	}

	channels := []uiterm.TreeItem{}
	for _, subchannel := range treeItem.Channel.Channels() {
		channels = append(channels, TreeItem{
			Channel: subchannel,
		})
	}

	return append(users, channels...)
}
