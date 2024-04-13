package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/pelletier/go-toml/v2"
)

type game_entry struct {
	Name        string
	Path        string
	Libary_name string
	Id          int
}

type app_config struct {
	Game_libary_location string
	App_location         string
	Logs_location        string
}

type state struct {
	SelectedGame game_entry
}

func new_config() app_config {
	return app_config{
		Game_libary_location: os.Getenv("HOME") + "/.config/lwl/libary/",
		App_location:         os.Getenv("HOME") + "/.config/lwl/",
		Logs_location:        os.Getenv("HOME") + "/.config/lwl/logs/",
	}
}

var global_state state

func setup_log(con *app_config) {
	log_file := con.Logs_location + "log"

	if _, err := os.Stat(log_file); err == nil {
		os.Remove(log_file)
	}

	f, err := os.OpenFile(log_file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		log.Fatal(err)
	}

	wrt := io.MultiWriter(os.Stdout, f)
	log.SetOutput(wrt)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Logs setup at: " + log_file)
}

func setup_fs(con *app_config) {
	err := os.MkdirAll(con.Game_libary_location, 0777)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(con.Logs_location, 0777)
	if err != nil {
		log.Fatal(err)
	}

	setup_log(con)

	if _, err := os.Stat(os.Getenv("HOME") + "/.config/lwl/config.toml"); err == nil {
		log.Println("Using config: " + os.Getenv("HOME") + "/.config/lwl/config.toml")

		f, err := os.Open(os.Getenv("HOME") + "/.config/lwl/config.toml")
		if err != nil {
			log.Fatal(err)
		}

		defer f.Close()

		data, err := io.ReadAll(f)
		if err != nil {
			log.Fatal(err)
		}

		err = toml.Unmarshal(data, &con)
		if err != nil {
			log.Fatal(err)
		}

		return
	}

	log.Println("Config not found, creating new config at: " + os.Getenv("HOME") + "/.config/lwl/config.toml")

	config_fs, err := toml.Marshal(con)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(os.Getenv("HOME")+"/.config/lwl/config.toml", config_fs, 0777)
	if err != nil {
		log.Fatal(err)
	}
}

func create_game(game *game_entry, con *app_config) {
	name_fs := game.Name
	name_fs = strings.ToLower(name_fs)
	name_fs = strings.Trim(name_fs, " ")
	name_fs = strings.ReplaceAll(name_fs, " ", "_")
	name_fs = name_fs + ".toml"

	game.Libary_name = name_fs

	game_fs, err := toml.Marshal(game)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(con.Game_libary_location+name_fs, game_fs, 0777)
	if err != nil {
		log.Fatal(err)
	}

	read_libary(con)
}

func delete_game(game *game_entry, w fyne.Window, con *app_config) {
	path := con.Game_libary_location + game.Libary_name

	log.Println("Game:", path, "deleted")

	err := os.Remove(path)
	if err != nil {
		log.Fatal(err)
	}

	main_page(w, con)
}

func read_libary(con *app_config) []game_entry {
	var game_list []game_entry

	l, err := os.Open(con.Game_libary_location)
	if err != nil {
		log.Fatal(err)
	}

	defer l.Close()

	games, err := l.Readdir(0)
	if err != nil {
		log.Fatal(err)
	}

	for _, game := range games {
		var game_entry game_entry

		f, err := os.Open(con.Game_libary_location + game.Name())
		if err != nil {
			log.Fatal(err)
		}

		defer f.Close()

		d, err := io.ReadAll(f)
		if err != nil {
			log.Fatal(err)
		}

		err = toml.Unmarshal(d, &game_entry)
		if err != nil {
			log.Fatal(err)
		}

		game_list = append(game_list, game_entry)
	}

	return game_list
}

func lunch_game(game *game_entry, w fyne.Window) {
	game_path := game.Path

	cmd := exec.Command(game_path)
	cmd.Dir = path.Dir(game_path)

	err := cmd.Start()
	if err != nil {
		log.Println("Error while lunching game: " + game.Name + ", " + err.Error())
		dialog.ShowInformation("Error", "Error while lunching game: "+game.Name+", "+err.Error(), w)
		return
	}

	log.Println("Lunched game: \"" + game.Name + "\" exec: \"" + game.Path + "\" work dir: \"" + path.Dir(game_path) + "\"")
}

func main_page(w fyne.Window, con *app_config) {
	w.Content().Refresh()

	game_list := read_libary(con)

	for game := range game_list {
		log.Println("Found game: \"" + game_list[game].Name + "\", file:" + con.Game_libary_location + game_list[game].Libary_name)
	}

	list := new_game_list(game_list, w, con)

	play_game_button := widget.NewButton("Play", func() {
		if global_state.SelectedGame.Path == "" {
			return
		}

		lunch_game(&global_state.SelectedGame, w)
	})

	add_game_button := widget.NewButton("Add Game", func() { new_game_page(w, con, game_list) })

	button_box := container.New(layout.NewVBoxLayout(), play_game_button, add_game_button)

	content := container.NewBorder(nil, nil, nil, button_box, list)

	list.OnSelected = func(id widget.ListItemID) {
		global_state.SelectedGame = game_list[id]
		log.Println("Selected game: \""+game_list[id].Name+"\", list id:", id, "game id:", game_list[id].Id)
	}

	w.SetContent(content)
}

func new_game_page(w fyne.Window, con *app_config, game_list []game_entry) {
	var new_game game_entry

	game_name_entry := widget.NewEntry()
	path_game_entry := widget.NewEntry()
	path_game_dialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			log.Fatal(err)
		}

		if reader == nil {
			return
		}

		path_game_entry.SetText(reader.URI().Path())
	}, w)
	path_game_dialog.Resize(fyne.NewSize(1200, 600))
	path_dialog_button := widget.NewButton("File...", func() {
		path_game_dialog.Show()
	})

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Game Name", Widget: game_name_entry},
			{Text: "Game Path", Widget: path_game_entry},
			{Widget: path_dialog_button},
		},
		OnSubmit: func() {
			if game_name_entry.Text == "" {
				return
			}

			if path_game_entry.Text == "" {
				return
			}

			new_game.Name = game_name_entry.Text
			new_game.Path = path_game_entry.Text
			new_game.Id = len(game_list)

			create_game(&new_game, con)
			main_page(w, con)
		},
		OnCancel: func() {
			main_page(w, con)
		},
	}

	w.SetContent(form)
}

func new_game_list(game_list []game_entry, w fyne.Window, con *app_config) *widget.List {
	return widget.NewList(
		func() int {
			return len(game_list)
		},
		func() fyne.CanvasObject {
			button_edit := widget.NewButton("Edit", nil)
			button_edit.Importance = widget.LowImportance

			delete_button := widget.NewButton("Delete", nil)
			delete_button.Importance = widget.DangerImportance

			return container.NewBorder(
				nil,
				nil,
				widget.NewLabel("template"),
				container.NewHBox(
					button_edit,
					delete_button,
				),
				nil,
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			label := o.(*fyne.Container).Objects[0].(*widget.Label)
			label.SetText(game_list[i].Name)

			button_edit := o.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*widget.Button)

			button_edit.OnTapped = func() {
				log.Println("Edit game:" + game_list[i].Name)
			}

			button_delete := o.(*fyne.Container).Objects[1].(*fyne.Container).Objects[1].(*widget.Button)

			button_delete.OnTapped = func() {
				delete_game(&game_list[i], w, con)
			}
		})
}

func main() {
	a := app.New()
	w := a.NewWindow("LWL")

	config := new_config()

	setup_fs(&config)

	w.Resize(fyne.NewSize(1600, 800))

	main_page(w, &config)

	w.ShowAndRun()
}
