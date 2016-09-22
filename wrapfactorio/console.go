package main

import "fmt"
import "os/exec"
import "github.com/chzyer/readline"

type Factorio struct {
	RLConfig *readline.Config
	console  readline.Instance
	process  *exec.Cmd
}

const program = `bin/x64/factorio`

// Start blocks until the server exits.
func (f *Factorio) Start(args []string) error {
	var err error
	f.console, err = readline.NewEx(f.RLConfig)
	if err != nil {
		return err
	}
}

func main() {
	fmt.Println("vim-go")
}
