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
	root        Node
	currentNode Node
}

func NewLibrary(dir string) *Library {
	rootPath := filepath.Join(dir, "Music")
	root := tview.NewTreeNode("Music").
		SetTextStyle(tcell.StyleDefault.Foreground(thunder.Colors.Playlist).
			Background(thunder.Colors.Background)).SetSelectable(false)

	rootNode := NewNode()

	tree := tview.NewTreeView().SetRoot(root).SetCurrentNode(root)

	library := &Library{
		TreeView: tree,
		root:     rootNode,
	}

	library.addNode(root, rootNode, rootPath)

	library.SetSelectedFunc(func(treeNode *tview.TreeNode) {
		reference := treeNode.GetReference()
		if reference == nil {
			return
		}
		node := reference.(Node)
		if !node.IsDir() {
			return
		}
		children := node.Children()
		if len(children) == 0 {
			library.addNode(treeNode, node, node.Path())
		} else {
			treeNode.SetExpanded(!treeNode.IsExpanded())
		}
	})

	library.SetChangedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()

		if ref == nil {
			return
		}

		library.currentNode = ref.(Node)
	})

	library.SetBlurFunc(func() {
		currentNode := library.currentNode
		currentTreeNode := library.GetCurrentNode()
		if currentNode != nil {
			if currentNode.IsDir() {
				currentTreeNode.SetSelectedTextStyle(
					tcell.StyleDefault.Foreground(thunder.Colors.Playlist).Background(thunder.Colors.Background),
				)
			} else {
				currentTreeNode.SetSelectedTextStyle(
					tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(thunder.Colors.Background),
				)
			}
		}
	})

	library.SetFocusFunc(func() {
		currentNode := library.currentNode
		currentTreeNode := library.GetCurrentNode()
		if currentNode != nil {
			if currentNode.IsDir() {
				currentTreeNode.SetSelectedTextStyle(
					tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Playlist),
				)
			} else {
				currentTreeNode.SetSelectedTextStyle(
					tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Song),
				)
			}
		}
	})

	library.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Rune() {
		// case ' ':
		// 	return nil
		case 'l':
			currentNode := library.currentNode

			if !currentNode.IsDir() {
				song := nodeToSong(currentNode.Name(), currentNode.Path())
				thunder.PlayPanel.playlist.Insert(song)
				return e
			}

			if library.CheckIsModified() {
				confirmationPopup("You have unsaved changes.\nAre you sure you want to close this playlist?", func(_ int, label string) {
					if label == "yes" {
						playlistInfo := nodeToPlaylistInfo(currentNode.Name(), currentNode.Path(), currentNode.Children())

						thunder.PlayPanel.Update(playlistInfo)
					}
				})
			} else {
				playlistInfo := nodeToPlaylistInfo(currentNode.Name(), currentNode.Path(), currentNode.Children())

				thunder.PlayPanel.Update(playlistInfo)
			}
		case 'n':
			if library.CheckIsModified() {
				confirmationPopup("Are you sure?", func(_ int, label string) {
					if label == "yes" {
						playlistInfo := NewDefaultPlaylistInfo()

						thunder.PlayPanel.Update(playlistInfo)
					}
				})
			} else {
				playlistInfo := NewDefaultPlaylistInfo()

				thunder.PlayPanel.Update(playlistInfo)
			}
		}

		return e
	})

	library.
		SetBorder(true).
		SetTitle("Library").
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(thunder.Colors.Accent).
		SetTitleColor(thunder.Colors.Accent).
		SetBorderPadding(0, 0, 1, 1).
		SetBackgroundColor(thunder.Colors.Background)

	return library
}

func (l *Library) addNode(treeParent *tview.TreeNode, modelParent Node, path string) {
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		exit(err)
	}

	children := make([]Node, 0, len(dirEntries))

	for _, entry := range dirEntries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) != allowedExt {
			continue
		}
		path := filepath.Join(path, entry.Name())
		name := entry.Name()

		childrenNode := NewNode()
		childrenNode.SetIsDir(entry.IsDir())
		childrenNode.SetName(name)
		childrenNode.SetPath(path)
		childrenNode.SetParent(modelParent)

		children = append(children, childrenNode)

		style := tcell.StyleDefault.Foreground(thunder.Colors.Song).Background(thunder.Colors.Background)
		selectedStyle := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Song)

		treeNode := tview.NewTreeNode(entry.Name()).
			SetReference(path)

		if entry.IsDir() {
			style = tcell.StyleDefault.Foreground(thunder.Colors.Playlist).Background(thunder.Colors.Background)
			selectedStyle = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(thunder.Colors.Playlist)
		}

		treeNode.SetTextStyle(style)
		treeNode.SetSelectedTextStyle(selectedStyle)
		treeNode.SetReference(childrenNode)
		treeParent.AddChild(treeNode)
	}

	modelParent.SetChildren(children)
}

func (l *Library) CheckIsModified() bool {
	return thunder.PlayPanel.playlist.isModified
}
