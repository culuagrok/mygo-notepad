// Notepad is a simple plain-text editor in MyGo native UI, built on
// mygo.NewDocuments: File menu (New, Open, Open Recent, Save, Save As,
// Revert, Close, Quit) with Ctrl+N/O/S and Ctrl+Shift+S, native file
// dialogs, the file name and a "*" for unsaved changes in the title, and a
// Save / Don't Save / Cancel prompt before unsaved changes are discarded.
package main

import (
	"fmt"
	"io"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type note struct {
	doc  *mygo.Document[string]
	win  *mygo.Window
	text string
}

func (n *note) view(c *ui.Context) {
	t := c.Theme()
	state := n.doc.State()
	ui.Column(c).Fill().Children(func() {
		if state.Conflict {
			ui.Row(c).Padding(10).Gap(8).Background(t.Warning.Alpha(0.15)).Children(func() {
				ui.Text(c, "Another program changed this file. Save to keep your version, or revert.").Grow(1)
				if ui.Button(c.Key("revert"), "Revert").Clicked() {
					go n.doc.Revert()
				}
				if ui.PrimaryButton(c.Key("save"), "Save").Clicked() {
					go n.doc.SaveAnyway()
				}
			})
		}
		ui.TextArea(c.Key("text"), &n.text).Fill().Font("monospace").Label("Text").AutoFocus().
			OnChange(func() { n.doc.Set(n.text) })
		ui.Row(c).Padding(4, 10).Children(func() {
			status := "Saved"
			switch {
			case state.Path == "":
				status = "Not saved"
			case state.ReadOnly:
				status = "Read-only"
			case state.Edited:
				status = "Edited"
			}
			ui.Text(c, fmt.Sprintf("%d characters", len([]rune(n.text)))).FontSize(12).TextColor(t.TextMuted).Grow(1)
			ui.Text(c, status).FontSize(12).TextColor(t.TextMuted)
		})
	})
}

func main() {
	mygo.NewDocuments(mygo.DocumentsOptions[string]{
		Types: []mygo.DocumentType{{Name: "Text Document", Extensions: []string{"txt", "log", "md", "ini", "csv"}}},
		Read: func(r io.Reader) (string, error) {
			b, err := io.ReadAll(r)
			return string(b), err
		},
		Write: func(w io.Writer, text string) error {
			_, err := io.WriteString(w, text)
			return err
		},
		Equal:     func(a, b string) bool { return a == b },
		Autosave:  mygo.AutosaveOff, // classic Notepad: save only when asked
		AskOnQuit: true,             // ask Save / Don't Save / Cancel on quit
		Window: func(doc *mygo.Document[string]) *mygo.Window {
			n := &note{doc: doc, text: doc.Content()}
			n.win = mygo.NewWindow(mygo.WindowOptions{Width: 800, Height: 600, StateKey: "note", Content: ui.View(n.view)})
			doc.OnReload(func() { n.text = doc.Content(); n.win.Update(func() {}) })
			doc.OnStateChange(func(mygo.DocumentState) { n.win.Update(func() {}) })
			return n.win
		},
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
