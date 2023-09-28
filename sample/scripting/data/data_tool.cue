package data

import (
	"encoding/json"
	"tool/exec"
	"tool/file"
)

pkgReq: {
	url:    "https://raw.githubusercontent.com/enketo/enketo-core/master/package.json"
	method: "GET"
}

// while in this directory, run 
// $ cue cmd gendata 
// to generate data.cue

// with exec.Run, you can do anything 

command: gendata: {
	get: {
		req: pkgReq & {
			$id: "tool/http.Do"
		}
		resp:  req.response
		_data: json.Unmarshal(resp.body)
		data:  json.Marshal(_data.os)
	}
	cacheResult: file.Append & {
		filename: "data.json"
		contents: "\(get.data)"
	}
	write: exec.Run & {
		$dep: cacheResult.$done
		cmd:  "cue import -p data -l os: -f \(cacheResult.filename)"
	}
	cleanup: exec.Run & {
		$dep: write.$done
		cmd:  "rm \(cacheResult.filename)"
	}
}
