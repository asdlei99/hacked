package movies

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/asticode/go-astisub"
	"github.com/inkyblackness/imgui-go"

	"github.com/inkyblackness/hacked/editor/external"
	"github.com/inkyblackness/hacked/ss1/content/audio"
	"github.com/inkyblackness/hacked/ss1/content/movie"
	"github.com/inkyblackness/hacked/ss1/edit/undoable"
	"github.com/inkyblackness/hacked/ss1/edit/undoable/cmd"
	"github.com/inkyblackness/hacked/ss1/resource"
	"github.com/inkyblackness/hacked/ss1/world"
	"github.com/inkyblackness/hacked/ss1/world/ids"
	"github.com/inkyblackness/hacked/ui/gui"
)

type movieInfo struct {
	title     string
	multilang bool
}

var knownMovies = map[resource.ID]movieInfo{
	ids.MovieIntro: {title: "Intro", multilang: true},
	ids.MovieDeath: {title: "Death", multilang: false},
	ids.MovieEnd:   {title: "End", multilang: false},
}

var knownMoviesOrder = []resource.ID{ids.MovieIntro, ids.MovieDeath, ids.MovieEnd}

// View provides edit controls for animations.
type View struct {
	mod *world.Mod

	movieService undoable.MovieService

	modalStateMachine gui.ModalStateMachine
	guiScale          float32
	commander         cmd.Commander

	model viewModel
}

// NewMoviesView returns a new instance.
func NewMoviesView(mod *world.Mod,
	movieService undoable.MovieService,
	modalStateMachine gui.ModalStateMachine, guiScale float32, commander cmd.Commander) *View {
	view := &View{
		mod: mod,

		movieService: movieService,

		modalStateMachine: modalStateMachine,
		guiScale:          guiScale,
		commander:         commander,

		model: freshViewModel(),
	}
	return view
}

// WindowOpen returns the flag address, to be used with the main menu.
func (view *View) WindowOpen() *bool {
	return &view.model.windowOpen
}

// Render renders the view.
func (view *View) Render() {
	if view.model.restoreFocus {
		imgui.SetNextWindowFocus()
		view.model.restoreFocus = false
		view.model.windowOpen = true
	}
	if view.model.windowOpen {
		imgui.SetNextWindowSizeV(imgui.Vec2{X: 800 * view.guiScale, Y: 300 * view.guiScale}, imgui.ConditionOnce)
		if imgui.BeginV("Movies", view.WindowOpen(), imgui.WindowFlagsNoCollapse|imgui.WindowFlagsHorizontalScrollbar) {
			view.renderContent()
		}
		imgui.End()
	}
}

func (view *View) renderContent() {
	if imgui.BeginChildV("Properties", imgui.Vec2{X: 350 * view.guiScale, Y: 0}, false, 0) {
		imgui.PushItemWidth(-150 * view.guiScale)
		if imgui.BeginCombo("Movie", knownMovies[view.model.currentKey.ID].title) {
			for _, id := range knownMoviesOrder {
				if imgui.SelectableV(knownMovies[id].title, id == view.model.currentKey.ID, 0, imgui.Vec2{}) {
					view.model.currentKey.ID = id
					view.model.currentKey.Index = 0
				}
			}
			imgui.EndCombo()
		}

		if knownMovies[view.model.currentKey.ID].multilang {
			if imgui.BeginCombo("Language", view.model.currentKey.Lang.String()) {
				languages := resource.Languages()
				for _, lang := range languages {
					if imgui.SelectableV(lang.String(), lang == view.model.currentKey.Lang, 0, imgui.Vec2{}) {
						view.model.currentKey.Lang = lang
					}
				}
				imgui.EndCombo()
			}
		} else {
			imgui.LabelText("Language", "(not localized)")
		}

		imgui.Separator()

		view.renderProperties()

		imgui.PopItemWidth()
	}
	imgui.EndChild()
	imgui.SameLine()

	if imgui.BeginChildV("Frames", imgui.Vec2{X: -1, Y: 0}, false, 0) {
		scenes := view.movieService.Video(view.model.currentKey)
		if len(scenes) > 0 && len(scenes[0].Frames) > 0 {
			// TODO: download frame & palette to GPU.
		}
	}
	imgui.EndChild()
}

func (view *View) renderProperties() {
	// gui.StepSliderInt("Frame Index", &view.model.currentFrame, 0, lastFrame)

	view.renderAudioProperties()
	view.renderSubtitlesProperties()
}

func (view *View) renderAudioProperties() {
	imgui.PushID("audio")
	imgui.Separator()
	sound := view.currentSound()
	imgui.LabelText("Audio", fmt.Sprintf("%.2f sec", sound.Duration()))
	if imgui.Button("Export") {
		view.requestExportAudio(sound)
	}
	imgui.SameLine()
	if imgui.Button("Import") {
		view.requestImportAudio()
	}
	imgui.PopID()
}

func (view *View) renderSubtitlesProperties() {
	imgui.PushID("subtitles")
	imgui.Separator()
	if imgui.BeginCombo("Sub Language", view.model.currentSubtitleLang.String()) {
		languages := resource.Languages()
		for _, lang := range languages {
			if imgui.SelectableV(lang.String(), lang == view.model.currentSubtitleLang, 0, imgui.Vec2{}) {
				view.model.currentSubtitleLang = lang
			}
		}
		imgui.EndCombo()
	}
	if imgui.Button("Export") {
		view.requestExportSubtitles()
	}
	imgui.SameLine()
	if imgui.Button("Import") {
		view.requestImportSubtitles()
	}
	imgui.PopID()
}

func (view *View) restoreFunc() func() {
	oldKey := view.model.currentKey
	oldSubtitlesLang := view.model.currentSubtitleLang

	return func() {
		view.model.restoreFocus = true
		view.model.currentKey = oldKey
		view.model.currentSubtitleLang = oldSubtitlesLang
	}
}

func (view *View) currentSound() audio.L8 {
	return view.movieService.Audio(view.model.currentKey)
}

func (view *View) requestExportAudio(sound audio.L8) {
	filename := fmt.Sprintf("%s_%s.wav", knownMovies[view.model.currentKey.ID].title, view.model.currentKey.Lang.String())

	external.ExportAudio(view.modalStateMachine, filename, sound)
}

func (view *View) requestImportAudio() {
	external.ImportAudio(view.modalStateMachine, func(sound audio.L8) {
		view.movieService.RequestSetAudio(view.model.currentKey, sound, view.restoreFunc())
	})
}

func (view *View) currentSubtitles() movie.Subtitles {
	return view.movieService.Subtitles(view.model.currentKey, view.model.currentSubtitleLang)
}

func (view View) requestExportSubtitles() {
	filename := fmt.Sprintf("%s_%s.srt", knownMovies[view.model.currentKey.ID].title, view.model.currentSubtitleLang.String())
	info := "File to be written: " + filename
	var exportTo func(string)
	currentSubtitles := view.currentSubtitles()

	exportTo = func(dirname string) {
		writer, err := os.Create(filepath.Join(dirname, filename))
		if err != nil {
			external.Export(view.modalStateMachine, "Could not create file.\n"+info, exportTo, true)
			return
		}
		defer func() { _ = writer.Close() }()

		sub := astisub.NewSubtitles()

		var lastItem *astisub.Item
		for _, entry := range currentSubtitles.Entries {
			var item astisub.Item
			var line astisub.Line
			line.Items = append(line.Items, astisub.LineItem{Text: entry.Text})
			item.Lines = []astisub.Line{line}
			item.StartAt = entry.Timestamp.ToDuration()
			item.EndAt = item.StartAt
			if lastItem != nil {
				lastItem.EndAt = item.StartAt
			}
			lastItem = &item
			sub.Items = append(sub.Items, lastItem)
		}

		err = sub.WriteToSRT(writer)
		if err != nil {
			external.Export(view.modalStateMachine, "Could not export subtitles.\n"+info, exportTo, true)
			return
		}
	}

	external.Export(view.modalStateMachine, info, exportTo, false)
}

func (view *View) requestImportSubtitles() {
	info := "File must be an .SRT file."
	types := []external.TypeInfo{{Title: "Subtitle files (*.srt)", Extensions: []string{"srt"}}}
	var fileHandler func(string)

	fileHandler = func(filename string) {
		reader, err := os.Open(filename)
		if err != nil {
			external.Import(view.modalStateMachine, "Could not open file.\n"+info, types, fileHandler, true)
			return
		}
		defer func() { _ = reader.Close() }()

		subtitles, err := astisub.ReadFromSRT(reader)
		if err != nil {
			external.Import(view.modalStateMachine, "File not recognized as SRT.\n"+info, types, fileHandler, true)
			return
		}
		var newSubtitles movie.Subtitles
		for _, item := range subtitles.Items {
			var newEntry movie.SubtitleEntry
			newEntry.Timestamp = movie.TimestampFromSeconds(float32(item.StartAt) / float32(time.Second))
			for _, line := range item.Lines {
				for _, lineItem := range line.Items {
					if len(newEntry.Text) > 0 {
						newEntry.Text += "\n"
					}
					newEntry.Text = newEntry.Text + lineItem.Text
				}
			}
			newSubtitles.Entries = append(newSubtitles.Entries, newEntry)
		}

		view.movieService.RequestSetSubtitles(view.model.currentKey, view.model.currentSubtitleLang,
			newSubtitles, view.restoreFunc())
	}

	external.Import(view.modalStateMachine, info, types, fileHandler, false)
}