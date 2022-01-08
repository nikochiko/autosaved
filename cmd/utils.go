package cmd

import (
	"fmt"

	"github.com/fatih/color"
)

type coloredOutput struct {
	Errorf    func(string, ...interface{})
	Printf    func(string, ...interface{})
	Successf  func(string, ...interface{})
	Warnf     func(string, ...interface{})
	Serrorf   func(string, ...interface{}) string
	Sprintf   func(string, ...interface{}) string
	Ssuccessf func(string, ...interface{}) string
	Swarnf    func(string, ...interface{}) string
}

var errDisplay = color.New(color.FgRed)
var successDisplay = color.New(color.FgGreen)
var warnDisplay = color.New(color.FgYellow)

var asdFmt = coloredOutput{
	Errorf: errDisplay.PrintfFunc(),
	Printf: func(format string, a ...interface{}) {
		fmt.Printf(format, a...)
	},
	Successf:  successDisplay.PrintfFunc(),
	Warnf:     warnDisplay.PrintfFunc(),
	Serrorf:   errDisplay.SprintfFunc(),
	Sprintf:   fmt.Sprintf,
	Ssuccessf: successDisplay.SprintfFunc(),
	Swarnf:    warnDisplay.SprintfFunc(),
}
