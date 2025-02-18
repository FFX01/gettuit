# **THIS IS A WIP, SUBJECT TO BREAKING CHANGES**
# GetTUIt

A Terminal User Interface for handling your tasks.

## Running locally
Clone this repository and run `go run .` in the root directory of the project(where `go.mod` is).

## Usage
After starting the program, press `F1` to see a list of keybinds. This list is relative
to the focused view and view mode.

## Structure Explanation
There are 2 major components to this project at this time; they are `main.go` and 
`internal/getuit`. `main.go` is the actual todo list/task management application.
`internal/gotuit` is an internal(for now) UI library designed t make it easier to build
a TUI without having to worry so much about coordinate offsets, conditionals for event 
handling logic, etc.

## Contributing
This project is open to contributions. There is no formal contribution procedure at 
this time. If you want to make changes, just create a fork and make a PR against this 
repo's main branch. Do be aware that the `internal/gotuit` module will eventually be 
broken out into an external package. Also be aware that this code is under heavy 
development and may change drastically on a daily, or even hourly basis.

## Development Journal
If you want to follow development of this project, you can check out these articles on
my personal blog here:
- https://justindev.io/tag/gettuit.html
- https://justindev.io/tag/gotui.html
