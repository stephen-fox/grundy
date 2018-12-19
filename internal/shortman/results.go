package shortman

import (
	"github.com/stephen-fox/grundy/internal/steamw"
)

type ManageResult struct {
	Created CreatedOrUpdated
	Deleted Deleted
}

type CreatedOrUpdated struct {
	gameNamesToResults    map[string]steamw.NewShortcutResult
	configPathsToLoadErrs map[string]error
	notAddedToReasons     map[string]string
	missingIcons          map[string]string
}

func (o *CreatedOrUpdated) CreatedInfo() []string {
	var s []string

	for name, r := range o.gameNamesToResults {
		for _, id := range r.CreatedForIds {
			s = append(s, "Created shortcut to " + name +
				" for Steam ID " + id)
		}
	}

	return s
}

func (o *CreatedOrUpdated) UpdatedInfo() []string {
	var s []string

	for name, r := range o.gameNamesToResults {
		for _, id := range r.UpdatedForIds {
			s = append(s, "Updated shortcut to " + name +
				" for Steam ID " + id)
		}
	}

	return s
}

func (o *CreatedOrUpdated) NotAddedInfo() []string {
	var s []string

	for configFilePath, reason := range o.notAddedToReasons {
		s = append(s, "Config file " + configFilePath + " was not added - " + reason)
	}

	return s
}

func (o *CreatedOrUpdated) FailuresInfo() []string {
	var s []string

	for configFilePath, err := range o.configPathsToLoadErrs {
		if err != nil {
			s = append(s, "Failed to load " + configFilePath + " - " + err.Error())
		}
	}

	for name, r := range o.gameNamesToResults {
		for id, err := range r.IdsToFailures {
			s = append(s, "Failed to create or update shortcut to " +
				name + " for Steam ID " + id + " - " + err.Error())
		}
	}

	return s
}

func (o *CreatedOrUpdated) IsErr() bool {
	if len(o.configPathsToLoadErrs) > 0 {
		return true
	}

	for _, r := range o.gameNamesToResults {
		if len(r.IdsToFailures) > 0 {
			return true
		}
	}

	return false
}

func (o *CreatedOrUpdated) MissingIconsInfo() []string {
	var s []string

	for name := range o.missingIcons {
		s = append(s, "Game '" + name + "' is missing an icon")
	}

	return s
}

type Deleted struct {
	results                  []steamw.DeletedShortcutsForSteamIdsResult
	stillHasExecutableToPath map[string]string
}

func (o *Deleted) DeletedInfo() []string {
	var s []string

	for _, r := range o.results {
		for steamId, gameNames := range r.IdsToDeletedGames {
			for _, name := range gameNames {
				s = append(s, "Deleted shortcut for " + name + " for Steam ID " + steamId)
			}
		}
	}

	return s
}

func (o *Deleted) NotDeletedInfo() []string {
	var s []string

	for _, r := range o.results {
		for steamId, gameNames := range r.IdsToNotDeletedGames {
			for _, name := range gameNames {
				s = append(s, "Shortcut for " + name + " was not deleted for Steam ID " + steamId)
			}
		}
	}

	for name, exePath := range o.stillHasExecutableToPath {
		s = append(s, "Shortcut for " + name + " was not deleted because it still has an executable at '" +
			exePath + "'")
	}

	return s
}

func (o *Deleted) FailedToDeleteInfo() []string {
	var s []string

	for _, r := range o.results {
		for steamId, err := range r.IdsToFailures {
			s = append(s, "Failed to delete shortcut for Steam ID " + steamId + " - " + err.Error())
		}
	}

	return s
}

func (o *Deleted) IsErr() bool {
	for _, r := range o.results {
		if len(r.IdsToFailures) > 0 {
			return true
		}
	}

	return false
}
