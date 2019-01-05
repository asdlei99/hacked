package texts

import (
	"bytes"
	"fmt"

	"github.com/inkyblackness/imgui-go"

	"github.com/inkyblackness/hacked/editor/cmd"
	"github.com/inkyblackness/hacked/editor/external"
	"github.com/inkyblackness/hacked/editor/model"
	"github.com/inkyblackness/hacked/ss1/content/audio"
	"github.com/inkyblackness/hacked/ss1/content/movie"
	"github.com/inkyblackness/hacked/ss1/content/text"
	"github.com/inkyblackness/hacked/ss1/resource"
	"github.com/inkyblackness/hacked/ss1/world/ids"
	"github.com/inkyblackness/hacked/ui/gui"
)

type textInfo struct {
	id    resource.ID
	title string
}

var knownTextTypes = []textInfo{
	{id: ids.TrapMessageTexts, title: "Trap Messages"},
	{id: ids.WordTexts, title: "Words"},
	{id: ids.LogCategoryTexts, title: "Log Categories"},
	{id: ids.VariousMessageTexts, title: "Various Messages"},
	{id: ids.ScreenMessageTexts, title: "Screen Messages"},
	{id: ids.InfoNodeMessageTexts, title: "Info Node Message Texts (8/5/6)"},
	{id: ids.AccessCardNameTexts, title: "Access Card Names"},
	{id: ids.DataletMessageTexts, title: "Datalet Messages (8/5/8)"},
	{id: ids.PaperTextsStart, title: "Papers"},
	{id: ids.PanelNameTexts, title: "Panel Names"},
}

var textToAudio = map[resource.ID]resource.ID{
	ids.TrapMessageTexts: ids.TrapMessagesAudioStart,
}

// View provides edit controls for texts.
type View struct {
	mod        *model.Mod
	lineCache  *text.Cache
	pageCache  *text.Cache
	cp         text.Codepage
	movieCache *movie.Cache

	modalStateMachine gui.ModalStateMachine
	clipboard         external.Clipboard
	guiScale          float32
	commander         cmd.Commander

	model viewModel

	textTypeTitleByID map[resource.ID]string
}

// NewTextsView returns a new instance.
func NewTextsView(mod *model.Mod,
	lineCache *text.Cache, pageCache *text.Cache, cp text.Codepage, movieCache *movie.Cache,
	modalStateMachine gui.ModalStateMachine, clipboard external.Clipboard,
	guiScale float32, commander cmd.Commander) *View {
	view := &View{
		mod:        mod,
		lineCache:  lineCache,
		pageCache:  pageCache,
		cp:         cp,
		movieCache: movieCache,

		modalStateMachine: modalStateMachine,
		clipboard:         clipboard,
		guiScale:          guiScale,
		commander:         commander,

		model: freshViewModel(),

		textTypeTitleByID: make(map[resource.ID]string),
	}
	for _, info := range knownTextTypes {
		view.textTypeTitleByID[info.id] = info.title
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
		imgui.SetNextWindowSizeV(imgui.Vec2{X: 400 * view.guiScale, Y: 300 * view.guiScale}, imgui.ConditionOnce)
		if imgui.BeginV("Texts", view.WindowOpen(), imgui.WindowFlagsNoCollapse) {
			view.renderContent()
		}
		imgui.End()
	}
}

func (view *View) renderContent() {
	imgui.PushItemWidth(-100 * view.guiScale)
	if imgui.BeginCombo("Text Type", view.textTypeTitleByID[view.model.currentKey.ID]) {
		for _, info := range knownTextTypes {
			if imgui.SelectableV(info.title, info.id == view.model.currentKey.ID, 0, imgui.Vec2{}) {
				view.model.currentKey.ID = info.id
				view.model.currentKey.Index = 0
			}
		}
		imgui.EndCombo()
	}
	info, _ := ids.Info(view.model.currentKey.ID)
	gui.StepSliderInt("Index", &view.model.currentKey.Index, 0, info.MaxCount-1)

	if imgui.BeginCombo("Language", view.model.currentKey.Lang.String()) {
		languages := resource.Languages()
		for _, lang := range languages {
			if imgui.SelectableV(lang.String(), lang == view.model.currentKey.Lang, 0, imgui.Vec2{}) {
				view.model.currentKey.Lang = lang
			}
		}
		imgui.EndCombo()
	}

	if view.hasAudio() {
		imgui.Separator()
		sound := view.currentSound()
		hasSound := len(sound.Samples) > 0
		if hasSound {
			imgui.LabelText("Audio", fmt.Sprintf("%.2f sec", float32(len(sound.Samples))/sound.SampleRate))
			if imgui.Button("Export") {
				view.requestExportAudio(sound)
			}
			imgui.SameLine()
		} else {
			imgui.LabelText("Audio", "(no sound)")
		}
		if imgui.Button("Import") {
			view.requestImportAudio()
		}
	}
	imgui.Separator()

	imgui.PopItemWidth()

	currentText := view.currentText()
	imgui.BeginChildV("Text", imgui.Vec2{X: -100 * view.guiScale, Y: 0}, true, 0)
	imgui.PushTextWrapPos()
	if len(currentText) == 0 {
		imgui.PushStyleColor(imgui.StyleColorText, imgui.Vec4{X: 1.0, Y: 1.0, Z: 1.0, W: 0.5})
		imgui.Text("(empty)")
		imgui.PopStyleColor()
	} else {
		imgui.Text(currentText)
	}
	imgui.PopTextWrapPos()
	imgui.EndChild()

	imgui.SameLine()

	imgui.BeginGroup()
	if imgui.ButtonV("-> Clip", imgui.Vec2{X: -1, Y: 0}) {
		view.copyTextToClipboard(currentText)
	}
	if imgui.ButtonV("<- Clip", imgui.Vec2{X: -1, Y: 0}) {
		view.setTextFromClipboard()
	}
	if imgui.ButtonV("Clear", imgui.Vec2{X: -1, Y: 0}) {
		view.clearText()
	}
	if imgui.ButtonV("Remove", imgui.Vec2{X: -1, Y: 0}) {
		view.removeText()
	}
	imgui.EndGroup()
}

func (view View) currentText() string {
	var cache *text.Cache
	resourceInfo, existing := ids.Info(view.model.currentKey.ID)
	if !existing || resourceInfo.List {
		cache = view.lineCache
	} else {
		cache = view.pageCache
	}
	currentValue, cacheErr := cache.Text(view.model.currentKey)
	if cacheErr != nil {
		currentValue = ""
	}
	return currentValue
}

func (view View) hasAudio() bool {
	return view.model.currentKey.ID == ids.TrapMessageTexts
}

func (view *View) currentSoundKey() (key resource.Key, hasAudio bool) {
	if !view.hasAudio() {
		return key, false
	}
	key = view.model.currentKey
	key.ID = textToAudio[key.ID].Plus(key.Index)
	key.Index = 0
	return key, true

}

func (view *View) currentSound() (sound audio.L8) {
	key, hasAudio := view.currentSoundKey()
	if !hasAudio {
		return
	}
	sound, _ = view.movieCache.Audio(key)
	return
}

func (view View) currentModification() (data [][]byte, isList bool) {
	info, _ := ids.Info(view.model.currentKey.ID)
	if info.List {
		data = [][]byte{view.mod.ModifiedBlock(view.model.currentKey.Lang, view.model.currentKey.ID, view.model.currentKey.Index)}
	} else {
		data = view.mod.ModifiedBlocks(view.model.currentKey.Lang, view.model.currentKey.ID.Plus(view.model.currentKey.Index))
	}
	return data, info.List
}

func (view View) copyTextToClipboard(text string) {
	if len(text) > 0 {
		view.clipboard.SetString(text)
	}
}

func (view *View) setTextFromClipboard() {
	value, err := view.clipboard.String()
	if err != nil {
		return
	}

	blockedValue := text.Blocked(value)
	oldData, isList := view.currentModification()
	if isList {
		newData := view.cp.Encode(blockedValue[0])
		view.requestSetTextLine(oldData[0], newData, false, nil)
	} else {
		newData := make([][]byte, len(blockedValue))
		for index, blockLine := range blockedValue {
			newData[index] = view.cp.Encode(blockLine)
		}
		view.requestSetTextPage(oldData, newData)
	}
}

func (view *View) clearText() {
	emptyLine := view.cp.Encode("")
	currentModification, isList := view.currentModification()
	if isList && !bytes.Equal(currentModification[0], emptyLine) {
		view.requestSetTextLine(currentModification[0], emptyLine, true, &audio.L8{SampleRate: 22050, Samples: []byte{0x80}})
	} else if !isList && ((len(currentModification) != 1) || !bytes.Equal(currentModification[0], []byte{0x00})) {
		view.requestSetTextPage(currentModification, [][]byte{emptyLine})
	}
}

func (view *View) removeText() {
	currentModification, isList := view.currentModification()
	if isList && (len(currentModification[0]) > 0) {
		view.requestSetTextLine(currentModification[0], nil, true, nil)
	} else if !isList && (len(currentModification) > 0) {
		view.requestSetTextPage(currentModification, nil)
	}
}

func (view *View) requestExportAudio(sound audio.L8) {
	key, hasAudio := view.currentSoundKey()
	if !hasAudio {
		return
	}
	filename := fmt.Sprintf("%05d_%s.wav", key.ID.Value(), view.model.currentKey.Lang.String())

	external.ExportAudio(view.modalStateMachine, filename, sound)
}

func (view *View) requestImportAudio() {
	external.ImportAudio(view.modalStateMachine, func(sound audio.L8) {
		view.requestSetAudio(sound)
	})
}

func (view *View) setAudioCommand(sound *audio.L8) cmd.Command {
	dataKey, _ := view.currentSoundKey()
	command := setAudioCommand{
		restoreKey: view.model.currentKey,
		model:      &view.model,

		dataKey: dataKey,
		oldData: view.mod.ModifiedBlocks(dataKey.Lang, dataKey.ID),
	}
	if sound != nil {
		movieData := movie.ContainSoundData(*sound)
		command.newData = [][]byte{movieData}
	}
	return command
}

func (view *View) requestSetAudio(sound audio.L8) {
	view.commander.Queue(view.setAudioCommand(&sound))
}

func (view *View) requestSetTextLine(oldData []byte, newData []byte, withAudio bool, sound *audio.L8) {
	commands := cmd.List{
		setTextLineCommand{
			key:     view.model.currentKey,
			model:   &view.model,
			oldData: oldData,
			newData: newData,
		},
	}
	if withAudio && view.hasAudio() {
		commands = append(commands, view.setAudioCommand(sound))
	}
	view.commander.Queue(commands)
}

func (view *View) requestSetTextPage(oldData [][]byte, newData [][]byte) {
	command := setTextPageCommand{
		key:     view.model.currentKey,
		model:   &view.model,
		oldData: oldData,
		newData: newData,
	}
	view.commander.Queue(command)
}
