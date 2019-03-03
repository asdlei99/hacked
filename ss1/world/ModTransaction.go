package world

import (
	"github.com/inkyblackness/hacked/ss1/content/object"
	"github.com/inkyblackness/hacked/ss1/content/texture"
	"github.com/inkyblackness/hacked/ss1/resource"
)

type modAction func(modder Modder)

// ModTransaction is used to queue a list of modifications.
// It allows modifications of related resources in one atomic action.
type ModTransaction struct {
	actions     []modAction
	modifiedIDs resource.IDMarkerMap
}

// SetResourceBlock changes the block data of a resource.
//
// If the block data is not empty, then:
// If the resource does not exist, it will be created with default meta information.
// If the block does not exist, the resource is extended to allow its addition.
//
// If the block data is empty (or nil), then the block is cleared.
// If the resource is a compound list, then the underlying data will become visible again.
func (trans *ModTransaction) SetResourceBlock(lang resource.Language, id resource.ID, index int, data []byte) {
	trans.actions = append(trans.actions, func(modder Modder) {
		modder.SetResourceBlock(lang, id, index, data)
	})
	trans.modifiedIDs.Add(id)
}

// PatchResourceBlock modifies an existing block.
// This modification assumes the block already exists and can take the given patch data.
// The patch data is expected to be produced by rle.Compress().
func (trans *ModTransaction) PatchResourceBlock(lang resource.Language, id resource.ID, index int, expectedLength int, patch []byte) {
	trans.actions = append(trans.actions, func(modder Modder) {
		modder.PatchResourceBlock(lang, id, index, expectedLength, patch)
	})
	trans.modifiedIDs.Add(id)
}

// SetResourceBlocks sets the entire list of block data of a resource.
// This method is primarily meant for compound non-list resources (e.g. text pages).
func (trans *ModTransaction) SetResourceBlocks(lang resource.Language, id resource.ID, data [][]byte) {
	trans.actions = append(trans.actions, func(modder Modder) {
		modder.SetResourceBlocks(lang, id, data)
	})
	trans.modifiedIDs.Add(id)
}

// DelResource removes a resource from the mod in the given language.
//
// After the deletion, all the underlying data of the world will become visible again.
func (trans *ModTransaction) DelResource(lang resource.Language, id resource.ID) {
	trans.actions = append(trans.actions, func(modder Modder) {
		modder.DelResource(lang, id)
	})
	trans.modifiedIDs.Add(id)
}

// SetTextureProperties updates the properties of a specific texture.
func (trans *ModTransaction) SetTextureProperties(textureIndex int, properties texture.Properties) {
	trans.actions = append(trans.actions, func(modder Modder) {
		modder.SetTextureProperties(textureIndex, properties)
	})
}

// SetObjectProperties updates the properties of a specific object.
func (trans *ModTransaction) SetObjectProperties(triple object.Triple, properties object.Properties) {
	trans.actions = append(trans.actions, func(modder Modder) {
		modder.SetObjectProperties(triple, properties)
	})
}
