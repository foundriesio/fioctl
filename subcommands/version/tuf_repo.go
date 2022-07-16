package version

import _ "embed"

// This is generated with:
//
//	git show tuf-metadata:repository/root.json | json_pp > ./subcommands/version/root.json
//
//go:embed root.json
var InitialTufRoot []byte

const TufRepoUrl = "https://raw.githubusercontent.com/foundriesio/fioctl/tuf-metadata/repository/"
