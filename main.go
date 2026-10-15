// Notepad is a simple plain-text editor in MyGo native UI, built on
// mygo.NewDocuments: a File menu with New, Open, Open Recent, Save, Save As,
// Revert, Encoding, Reopen with Encoding, Close and Quit (Ctrl+N/O/S,
// Ctrl+Shift+S, Ctrl+W), native file dialogs, the file name and a "*" for
// unsaved changes in the title, a Save / Don't Save / Cancel prompt before
// unsaved changes are discarded, and View > Theme (System, Light, Dark).
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// file is a document's content: its text and the encoding it is saved in.
type file struct {
	Text     string
	Encoding string // an id of encodings
}

type note struct {
	doc  *mygo.Document[file]
	win  *mygo.Window
	text string
}

// notes maps windows to their notes. Main thread only.
var notes = map[*mygo.Window]*note{}

func (n *note) encoding() string { return n.doc.Content().Encoding }

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
			OnChange(func() { n.doc.Set(file{Text: n.text, Encoding: n.encoding()}) })
		ui.Row(c).Padding(4, 10).Gap(16).Children(func() {
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
			ui.Text(c, encodingByID(n.encoding()).Label).FontSize(12).TextColor(t.TextMuted)
			ui.Text(c, status).FontSize(12).TextColor(t.TextMuted)
		})
	})
}

// setEncoding saves the document in another encoding from now on, which
// edits it.
func (n *note) setEncoding(id string) {
	n.doc.Set(file{Text: n.text, Encoding: id})
	syncEncodingMenu(n)
	n.win.Update(func() {})
}

// reopenWithEncoding reads the document's file again, decoded with another
// encoding, dropping unsaved changes.
func (n *note) reopenWithEncoding(id string) {
	path := n.doc.State().Path
	if path == "" {
		return
	}
	b, err := os.ReadFile(path)
	if err == nil {
		var text string
		if text, err = decodeText(b, id); err == nil {
			n.text = text
			n.doc.Set(file{Text: text, Encoding: id})
			n.doc.SetEdited(false) // the content is what the file has
		}
	}
	if err != nil {
		log.Println("reopen with encoding:", err)
	}
	syncEncodingMenu(n)
	n.win.Update(func() {})
}

// ---- Theme ----

// themeFile is where the chosen appearance is kept, in PathUserData.
func themeFile() string {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "theme.txt")
}

func loadTheme() mygo.ThemeSource {
	if f := themeFile(); f != "" {
		if b, err := os.ReadFile(f); err == nil {
			switch s := mygo.ThemeSource(strings.TrimSpace(string(b))); s {
			case mygo.ThemeLight, mygo.ThemeDark, mygo.ThemeSystem:
				return s
			}
		}
	}
	return mygo.ThemeSystem
}

func saveTheme(s mygo.ThemeSource) {
	if f := themeFile(); f != "" {
		if err := os.WriteFile(f, []byte(s), 0o644); err != nil {
			log.Println("saving theme:", err)
		}
	}
}

var menu *mygo.Menu

// setTheme applies an appearance to every window, checks its menu item and
// remembers it for the next launch.
func setTheme(s mygo.ThemeSource) {
	mygo.Theme.SetSource(s)
	if it := menu.ItemByID("theme-" + string(s)); it != nil {
		it.SetChecked(true)
	}
	saveTheme(s)
}

func themeItem(label string, s mygo.ThemeSource, current mygo.ThemeSource) *mygo.MenuItem {
	return &mygo.MenuItem{
		ID: "theme-" + string(s), Label: label, Type: mygo.MenuItemRadio, Checked: s == current,
		Click: func(*mygo.MenuItem, *mygo.Window) { setTheme(s) },
	}
}

// ---- Encoding menu ----

// noteOf returns the note of the window a menu item was chosen in, or of
// the focused window.
func noteOf(win *mygo.Window) *note {
	if n := notes[win]; n != nil {
		return n
	}
	for w, n := range notes {
		if w.IsFocused() {
			return n
		}
	}
	return nil
}

// syncEncodingMenu checks the encoding of n in File > Encoding.
func syncEncodingMenu(n *note) {
	if it := menu.ItemByID("enc-" + n.encoding()); it != nil {
		it.SetChecked(true)
	}
}

func encodingItems() (set, reopen []*mygo.MenuItem) {
	for _, e := range encodings {
		id := e.ID
		set = append(set, &mygo.MenuItem{
			ID: "enc-" + id, Label: e.Label, Type: mygo.MenuItemRadio, Checked: id == defaultEncoding,
			Click: func(_ *mygo.MenuItem, win *mygo.Window) {
				if n := noteOf(win); n != nil {
					n.setEncoding(id)
				}
			},
		})
		reopen = append(reopen, &mygo.MenuItem{
			Label: e.Label,
			Click: func(_ *mygo.MenuItem, win *mygo.Window) {
				if n := noteOf(win); n != nil {
					n.reopenWithEncoding(id)
				}
			},
		})
	}
	return set, reopen
}

func main() {
	current := loadTheme()
	mygo.Theme.SetSource(current)
	setEnc, reopenEnc := encodingItems()
	menu = mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "File", Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleNew},
			{Role: mygo.RoleOpen},
			{Role: mygo.RoleOpenRecent},
			mygo.Separator(),
			{Role: mygo.RoleSave},
			{Role: mygo.RoleSaveAs},
			{Role: mygo.RoleRevert},
			mygo.Separator(),
			{Label: "Encoding", Submenu: setEnc},
			{Label: "Reopen with Encoding", Submenu: reopenEnc},
			mygo.Separator(),
			{Role: mygo.RoleClose},
			{Role: mygo.RoleQuit},
		}},
		{Role: mygo.RoleEditMenu},
		{Label: "View", Submenu: []*mygo.MenuItem{
			{Label: "Theme", Submenu: []*mygo.MenuItem{
				themeItem("System", mygo.ThemeSystem, current),
				themeItem("Light", mygo.ThemeLight, current),
				themeItem("Dark", mygo.ThemeDark, current),
			}},
			{Label: "Toggle Light/Dark", Accelerator: "CmdOrCtrl+Shift+D", Click: func(*mygo.MenuItem, *mygo.Window) {
				if mygo.Theme.IsDark() {
					setTheme(mygo.ThemeLight)
				} else {
					setTheme(mygo.ThemeDark)
				}
			}},
		}},
		{Role: mygo.RoleWindowMenu},
	})
	mygo.App.SetMenu(menu)

	mygo.NewDocuments(mygo.DocumentsOptions[file]{
		Types: []mygo.DocumentType{{Name: "Text Document", Extensions: []string{"txt", "log", "md", "ini", "csv"}}},
		New:   func() file { return file{Encoding: defaultEncoding} },
		Read: func(r io.Reader) (file, error) {
			b, err := io.ReadAll(r)
			if err != nil {
				return file{}, err
			}
			id := detectEncoding(b)
			text, err := decodeText(b, id)
			return file{Text: text, Encoding: id}, err
		},
		Write: func(w io.Writer, f file) error {
			b, err := encodeText(f.Text, f.Encoding)
			if err != nil {
				return err
			}
			_, err = w.Write(b)
			return err
		},
		Equal:     func(a, b file) bool { return a == b },
		Autosave:  mygo.AutosaveOff, // classic Notepad: save only when asked
		AskOnQuit: true,             // ask Save / Don't Save / Cancel on quit
		Window: func(doc *mygo.Document[file]) *mygo.Window {
			n := &note{doc: doc, text: doc.Content().Text}
			n.win = mygo.NewWindow(mygo.WindowOptions{Width: 800, Height: 600, StateKey: "note", Content: ui.View(n.view)})
			notes[n.win] = n
			n.win.OnClosed(func() { delete(notes, n.win) })
			n.win.OnFocus(func() { syncEncodingMenu(n) })
			doc.OnReload(func() {
				n.text = doc.Content().Text
				syncEncodingMenu(n)
				n.win.Update(func() {})
			})
			doc.OnStateChange(func(mygo.DocumentState) { n.win.Update(func() {}) })
			syncEncodingMenu(n)
			return n.win
		},
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
