package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"os"
	"path/filepath"
)

var allowedExt = ".mp3"

type Library struct {
	*tview.TreeView
	currentNode *tview.TreeNode
}

func NewLibrary(dir string) *Library {
	rootDir := filepath.Join(dir, "Music")
	root := tview.NewTreeNode("Music").SetColor(tcell.ColorBlue).SetSelectable(false)
	tree := tview.NewTreeView().SetRoot(root).SetCurrentNode(root)

	library := &Library{
		TreeView: tree,
	}
	addNode(root, rootDir)

	library.SetSelectedFunc(func(node *tview.TreeNode) {
		reference := node.GetReference()
		if reference == nil {
			return
		}
		children := node.GetChildren()
		if len(children) == 0 {
			path := reference.(string)
			addNode(node, path)
		} else {
			node.SetExpanded(!node.IsExpanded())
		}
	})

	library.SetBorderColor(tcell.ColorBlue).
		SetBorder(true).
		SetTitle("Library").
		SetTitleColor(tcell.ColorBlue).
		SetTitleAlign(tview.AlignLeft).
		SetBorderPadding(0, 0, 1, 1)

	return library
}

func addNode(target *tview.TreeNode, path string) {
	info, err := os.Stat(path)
	if err != nil || (err == nil && !info.IsDir()) {
		return
	}
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		panic(err)
	}

	for _, entry := range dirEntries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) != allowedExt {
			continue
		}
		color := tcell.ColorWhite
		node := tview.NewTreeNode(entry.Name()).
			SetReference(filepath.Join(path, entry.Name()))
		if entry.IsDir() {
			color = tcell.ColorBlue
		} else {
			color = tcell.ColorPurple
		}
		node.SetColor(color)
		target.AddChild(node)
	}
}
