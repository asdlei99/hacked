package undoable

import (
	"time"

	"github.com/inkyblackness/hacked/ss1/content/audio"
	"github.com/inkyblackness/hacked/ss1/content/movie"
	"github.com/inkyblackness/hacked/ss1/edit"
	"github.com/inkyblackness/hacked/ss1/edit/media"
	"github.com/inkyblackness/hacked/ss1/edit/undoable/cmd"
	"github.com/inkyblackness/hacked/ss1/resource"
	"github.com/inkyblackness/hacked/ss1/world"
)

// MovieService provides read/write functionality with undo capability.
type MovieService struct {
	wrapped   edit.MovieService
	commander cmd.Commander
}

// NewMovieService returns a new instance of a service.
func NewMovieService(wrapped edit.MovieService, commander cmd.Commander) MovieService {
	return MovieService{
		wrapped:   wrapped,
		commander: commander,
	}
}

// SizeWarning returns true if the given movie is larger than the underlying archive would support.
func (service MovieService) SizeWarning(key resource.Key) bool {
	return service.wrapped.SizeWarning(key)
}

// Video returns the video component of identified movie.
func (service MovieService) Video(key resource.Key) []movie.Scene {
	return service.wrapped.Video(key)
}

// RequestMoveSceneEarlier queues to move the identified scene earlier.
func (service MovieService) RequestMoveSceneEarlier(key resource.Key, scene int, restoreFunc func()) {
	service.requestCommand(
		func(setter media.MovieBlockSetter) {
			service.wrapped.MoveSceneEarlier(setter, key, scene)
		},
		service.wrapped.RestoreFunc(key),
		restoreFunc)
}

// RequestMoveSceneLater queues to move the identified scene later.
func (service MovieService) RequestMoveSceneLater(key resource.Key, scene int, restoreFunc func()) {
	service.requestCommand(
		func(setter media.MovieBlockSetter) {
			service.wrapped.MoveSceneLater(setter, key, scene)
		},
		service.wrapped.RestoreFunc(key),
		restoreFunc)
}

// RequestAddScene queues to add the given scene at the end of the movie.
func (service MovieService) RequestAddScene(key resource.Key, scene movie.HighResScene, restoreFunc func()) {
	service.requestCommand(
		func(setter media.MovieBlockSetter) {
			service.wrapped.AddScene(setter, key, scene)
		},
		service.wrapped.RestoreFunc(key),
		restoreFunc)
}

// RequestRemoveScene queues to remove the identified scene.
func (service MovieService) RequestRemoveScene(key resource.Key, scene int, restoreFunc func()) {
	service.requestCommand(
		func(setter media.MovieBlockSetter) {
			service.wrapped.RemoveScene(setter, key, scene)
		},
		service.wrapped.RestoreFunc(key),
		restoreFunc)
}

// RequestSetSceneFramesDisplayTime requests to set the display time for each frame.
func (service MovieService) RequestSetSceneFramesDisplayTime(key resource.Key,
	scene int, displayTime time.Duration, restoreFunc func()) {
	service.requestCommand(
		func(setter media.MovieBlockSetter) {
			service.wrapped.SetSceneFramesDisplayTime(setter, key, scene, displayTime)
		},
		service.wrapped.RestoreFunc(key),
		restoreFunc)
}

// Audio returns the audio component of identified movie.
func (service MovieService) Audio(key resource.Key) audio.L8 {
	return service.wrapped.Audio(key)
}

// RequestSetAudio queues the change to update the audio track.
func (service MovieService) RequestSetAudio(key resource.Key, soundData audio.L8, restoreFunc func()) {
	service.requestCommand(
		func(setter media.MovieBlockSetter) {
			service.wrapped.SetAudio(setter, key, soundData)
		},
		service.wrapped.RestoreFunc(key),
		restoreFunc)
}

// Subtitles returns the subtitles associated with the given key.
func (service MovieService) Subtitles(key resource.Key, language resource.Language) movie.SubtitleList {
	return service.wrapped.Subtitles(key, language)
}

// RequestSetSubtitles queues the change to update subtitles.
func (service MovieService) RequestSetSubtitles(key resource.Key,
	language resource.Language, subtitles movie.SubtitleList, restoreFunc func()) {
	service.requestCommand(
		func(setter media.MovieBlockSetter) {
			service.wrapped.SetSubtitles(setter, key, language, subtitles)
		},
		service.wrapped.RestoreFunc(key),
		restoreFunc)
}

func (service MovieService) requestCommand(
	forward func(modder media.MovieBlockSetter),
	reverse func(modder media.MovieBlockSetter),
	restore func()) {
	c := command{
		forward: func(modder world.Modder) { forward(modder) },
		reverse: func(modder world.Modder) { reverse(modder) },
		restore: restore,
	}
	service.commander.Queue(c)
}
