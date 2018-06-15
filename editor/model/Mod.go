package model

import (
	"io/ioutil"

	"github.com/inkyblackness/hacked/ss1/resource"
	"github.com/inkyblackness/hacked/ss1/world"
)

type identifiedResources map[resource.ID]*resource.Resource

// Mod is the central object for a game-mod.
//
// It is based on a "static" world and adds its own changes. The world data itself is not static, it is merely the
// unchangeable background for the mod. Changes to the mod are kept in a separate layer, which can be loaded and saved.
type Mod struct {
	worldManifest    *world.Manifest
	resourcesChanged resource.ModificationCallback

	localizedResources map[resource.Language]identifiedResources
}

// NewMod returns a new instance.
func NewMod(resourcesChanged resource.ModificationCallback) *Mod {
	mod := &Mod{
		resourcesChanged:   resourcesChanged,
		localizedResources: make(map[resource.Language]identifiedResources),
	}
	mod.worldManifest = world.NewManifest(mod.worldChanged)
	for _, lang := range resource.Languages() {
		mod.localizedResources[lang] = make(identifiedResources)
	}
	mod.localizedResources[resource.LangAny] = make(identifiedResources)

	return mod
}

// World returns the static background to the mod. Changes in the returned manifest may cause change callbacks
// being forwarded.
func (mod Mod) World() *world.Manifest {
	return mod.worldManifest
}

// ModifiedResource retrieves the resource of given language and ID.
// There is no fallback lookup, it will return the exact resource stored under the provided identifier.
// Returns nil if the resource does not exist.
func (mod Mod) ModifiedResource(lang resource.Language, id resource.ID) *resource.Resource {
	return mod.localizedResources[lang][id]
}

// ModifiedBlock retrieves the specific block identified by given key.
// Returns nil if the block (or resource) is not modified.
func (mod Mod) ModifiedBlock(key resource.Key) (data []byte) {
	res := mod.ModifiedResource(key.Lang, key.ID)
	if res == nil {
		return
	}
	if key.Index >= res.BlockCount() {
		return
	}
	reader, err := res.Block(key.Index)
	if err != nil {
		return
	}
	data, err = ioutil.ReadAll(reader)
	if err != nil {
		return nil
	}
	return
}

// Filter returns a list of resources that match the given parameters.
func (mod Mod) Filter(lang resource.Language, id resource.ID) resource.List {
	list := mod.worldManifest.Filter(lang, id)
	if res, resExists := mod.localizedResources[resource.LangAny][id]; resExists {
		list = list.With(res)
	}
	for _, worldLang := range resource.Languages() {
		if worldLang.Includes(lang) {
			if res, resExists := mod.localizedResources[lang][id]; resExists {
				list = list.With(res)
			}
		}
	}
	return list
}

// LocalizedResources returns a resource selector for a specific language.
func (mod Mod) LocalizedResources(lang resource.Language) resource.Selector {
	return resource.Selector{
		Lang: lang,
		From: mod,
		As:   world.ResourceViewStrategy(),
	}
}

// Modify requests to change the mod. The provided function will be called to collect all changes.
// After the modifier completes, all the requests will be applied and any changes notified.
func (mod *Mod) Modify(modifier func(*ModTransaction)) {
	notifier := resource.ChangeNotifier{
		Callback:  mod.resourcesChanged,
		Localizer: mod,
	}
	var trans ModTransaction
	trans.modifiedIDs = make(idMarkerMap)
	modifier(&trans)
	notifier.ModifyAndNotify(func() {
		for _, action := range trans.actions {
			action(mod)
		}
	}, trans.modifiedIDs.toList())
}

func (mod Mod) worldChanged(modifiedIDs []resource.ID, failedIDs []resource.ID) {
	// It would be great to also check whether the mod hides any of these changes.
	// Sadly, this is not possible:
	// a) At the point of this callback, we can't do a check on the previous state anymore.
	// b) Even when changing the world only within a modification enclosure of our own notifier, we can't determine
	//    the list of changed IDs before actually changing them. (Specifying ALL IDs is not a good idea due to performance.)
	// As a result, simply forward this list. I don't even expect any big performance gain through such a filter.
	// This would only be relevant to "full conversion" mods AND a change in a big list in the world. Hardly the case.
	mod.resourcesChanged(modifiedIDs, failedIDs)
}

func (mod *Mod) ensureResource(lang resource.Language, id resource.ID) *resource.Resource {
	res, resExists := mod.localizedResources[lang][id]
	if !resExists {
		res = mod.newResource(lang, id)
		mod.localizedResources[lang][id] = res
	}
	return res
}

func (mod *Mod) newResource(lang resource.Language, id resource.ID) *resource.Resource {
	// TODO: if not even existing, create based on defaults
	compound := false
	contentType := resource.Text
	compressed := false

	list := mod.worldManifest.Filter(lang, id)
	if len(list) > 0 {
		existing := list[0]
		compound = existing.Compound
		contentType = existing.ContentType
		compressed = existing.Compressed
	}

	return &resource.Resource{
		Compound:      compound,
		ContentType:   contentType,
		Compressed:    compressed,
		BlockProvider: resource.MemoryBlockProvider(nil),
	}
}

func (mod *Mod) delResource(lang resource.Language, id resource.ID) {
	for _, worldLang := range resource.Languages() {
		if lang.Includes(worldLang) {
			delete(mod.localizedResources[worldLang], id)
		}
	}
	if lang.Includes(resource.LangAny) {
		delete(mod.localizedResources[resource.LangAny], id)
	}
}
