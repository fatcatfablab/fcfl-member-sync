package version

import "fmt"

var (
	Version   string
	Commit    string
	Branch    string
	BuildTime string
	BuiltBy   string
)

func PrintVersion() {
	fmt.Printf("Version: %s\n"+
		"Commit: %s\n"+
		"Build Time: %s\n",
		Version,
		Commit,
		BuildTime,
	)
}
