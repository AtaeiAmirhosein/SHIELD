package core

import (
	"net/http"

	"github.com/starkandwayne/shield/db"
)

func (core *Core) v2GetArchives(w http.ResponseWriter, req *http.Request) {

	status := []string{}
	if s := paramValue(req, "status", ""); s != "" {
		status = append(status, s)
	}

	limit := paramValue(req, "limit", "")
	if invalidlimit(limit) {
		bailWithError(w, ClientErrorf("invalid limit supplied"))
		return
	}

	archives, err := core.DB.GetAllArchives(
		&db.ArchiveFilter{
			ForTarget:  paramValue(req, "target", ""),
			ForStore:   paramValue(req, "store", ""),
			Before:     paramDate(req, "before"),
			After:      paramDate(req, "after"),
			WithStatus: status,
			Limit:      limit,
		},
	)

	if err != nil {
		bail(w, err)
		return
	}

	JSON(w, archives)
}
