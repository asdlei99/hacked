package model

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/inkyblackness/hacked/ss1/resource"
)

// MutableResource describes a resource open for modification.
type MutableResource struct {
	filename  string
	saveOrder int

	compound    bool
	contentType resource.ContentType
	compressed  bool

	blockCount int
	blocks     map[int][]byte
}

// LocalizedResources is a map of identified resources by language.
type LocalizedResources map[resource.Language]IdentifiedResources

// NewLocalizedResources returns a new instance, prepared for all language keys.
func NewLocalizedResources() LocalizedResources {
	res := make(LocalizedResources)
	for _, lang := range resource.Languages() {
		res[lang] = make(IdentifiedResources)
	}
	res[resource.LangAny] = make(IdentifiedResources)
	return res
}

// MutableResourcesFromProvider returns MutableResource instances based on a provider.
// This function retrieves all data from the provider.
func MutableResourcesFromProvider(filename string, provider resource.Provider) IdentifiedResources {
	ids := provider.IDs()
	mutables := make(IdentifiedResources)
	for index, id := range ids {
		res, _ := provider.Resource(id)
		mutable := &MutableResource{
			filename:  filename,
			saveOrder: index,

			compound:    res.Compound(),
			contentType: res.ContentType(),
			compressed:  res.Compressed(),
			blocks:      make(map[int][]byte),
		}
		blockCount := res.BlockCount()
		for blockIndex := 0; blockIndex < blockCount; blockIndex++ {
			mutable.SetBlock(blockIndex, readBlock(res, blockIndex))
		}
		mutables[id] = mutable
	}
	return mutables
}

func readBlock(provider resource.BlockProvider, index int) []byte {
	reader, err := provider.Block(index)
	if err != nil {
		return nil
	}
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil
	}
	return data
}

// Filename returns the file this resource should be stored in.
func (res MutableResource) Filename() string {
	return res.filename
}

// Compound returns true if the resource holds zero, one, or more blocks.
func (res MutableResource) Compound() bool {
	return res.compound
}

// ContentType describes how the data shall be interpreted.
func (res MutableResource) ContentType() resource.ContentType {
	return res.contentType
}

// Compressed returns true if the resource shall be serialized in compressed form.
func (res MutableResource) Compressed() bool {
	return res.compressed
}

// BlockCount returns the number of blocks in this resource.
func (res MutableResource) BlockCount() int {
	return res.blockCount
}

// Block returns a reader for the identified block.
func (res MutableResource) Block(index int) (io.Reader, error) {
	if !res.isBlockIndexValid(index) {
		return nil, fmt.Errorf("block index wrong: %v/%v", index, res.blockCount)
	}
	return bytes.NewReader(res.blocks[index]), nil
}

// SetBlocks sets the data of all the blocks, essentially resetting the resource.
func (res *MutableResource) SetBlocks(data [][]byte) {
	res.blockCount = 0
	res.blocks = make(map[int][]byte)
	for index, blockData := range data {
		res.SetBlock(index, blockData)
	}
}

// SetBlock sets the data of the identified block.
func (res *MutableResource) SetBlock(index int, data []byte) {
	if index < 0 {
		return
	}
	res.blocks[index] = data
	if index >= res.blockCount {
		res.blockCount = index + 1
	}
}

func (res MutableResource) isBlockIndexValid(index int) bool {
	return (index >= 0) && (index < res.blockCount)
}
