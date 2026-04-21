package cmd

var version = "dev"

func Version() string {
	return version
}

func versionString() string {
	return "curfew " + Version()
}
