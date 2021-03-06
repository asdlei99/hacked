package wav_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/inkyblackness/hacked/ss1/content/audio/wav"
)

func TestSaveStoresBytes(t *testing.T) {
	sampleRate := float32(22050.0)
	samples := []byte{0x00, 0x40, 0x80, 0xC0, 0xFF}
	buf := bytes.NewBuffer(nil)

	err := wav.Save(buf, sampleRate, samples)

	require.Nil(t, err)
	assert.Equal(t, []byte{
		0x52, 0x49, 0x46, 0x46, // "RIFF"
		0x2B, 0x00, 0x00, 0x00, // len(RIFF)
		0x57, 0x41, 0x56, 0x45, // "WAVE"
		0x66, 0x6d, 0x74, 0x20, // "fmt "
		0x12, 0x00, 0x00, 0x00, // len(fmt)
		0x01, 0x00, // fmt:type
		0x01, 0x00, // fmt:channels
		0x22, 0x56, 0x00, 0x00, // fmt:samples/sec
		0x22, 0x56, 0x00, 0x00, // fmt:avgBytes/sec
		0x01, 0x00, // fmt:blockAlign
		0x08, 0x00, // fmt:bits/sample
		0x00, 0x00, // fmt:extensionSize
		0x64, 0x61, 0x74, 0x61, // "data"
		0x05, 0x00, 0x00, 0x00, // len(data)
		0x00, 0x40, 0x80, 0xC0, 0xFF}, buf.Bytes())
}
