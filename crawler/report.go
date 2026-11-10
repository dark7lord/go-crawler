package crawler

import (
	"encoding/json"
	"time"
)

func (crw *crawler) report() (data []byte, err error) {
	result := Report{
		RootURL:     crw.rootURL.String(),
		Depth:       crw.depth,
		Pages:       crw.pages,
		GeneratedAt: time.Now().UTC(),
	}

	if !crw.indentJSON {
		return json.Marshal(result)
	}

	return json.MarshalIndent(result, "", "    ")
}
