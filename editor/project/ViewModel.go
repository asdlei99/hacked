package project

type viewModel struct {
	restoreFocus          bool
	windowOpen            bool
	selectedManifestEntry int

	autosaveTimeoutSec int
}

func freshViewModel() viewModel {
	return viewModel{
		windowOpen:            true,
		selectedManifestEntry: -1,
		autosaveTimeoutSec:    5,
	}
}
