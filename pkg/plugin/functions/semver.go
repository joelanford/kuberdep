package functions

import "github.com/blang/semver/v4"

func init() {
	Register(inSemverRange)
}

func inSemverRange(versionRange string, version string) bool {
	r, err := semver.ParseRange(versionRange)
	if err != nil {
		panic(err)
	}
	v, err := semver.Parse(version)
	if err != nil {
		panic(err)
	}
	return r(v)
}
