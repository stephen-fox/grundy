package steamw

import (
	"github.com/stephen-fox/steamutil/locations"
)

type DataInfo struct {
	DataLocations locations.DataVerifier
	IdsToDirPaths map[string]string
}

func NewSteamDataInfo() (DataInfo, error) {
	v, err := locations.NewDataVerifier()
	if err != nil {
		return DataInfo{}, err
	}

	idsToDirs, err := v.UserIdsToDataDirPaths()
	if err != nil {
		return DataInfo{}, err
	}

	return DataInfo{
		DataLocations: v,
		IdsToDirPaths: idsToDirs,
	}, nil
}