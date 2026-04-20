package cmd

type exitError struct {
	code    int
	message string
}

func (e exitError) Error() string {
	return e.message
}

func (e exitError) ExitCode() int {
	if e.code == 0 {
		return 1
	}
	return e.code
}
