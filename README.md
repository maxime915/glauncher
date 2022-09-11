# GLauncher

## Road map

- [ ] make config human readable
    - [ ] use a map[string]interface{} instead of []byte for second stage deserialization
    - [ ] use json.Indent for prettier printing
- [ ] add an entry in fzf to blacklist applications
- [X] optional parameters
    - [X] fzf could have an option to only highlight in nautilus (instead of xdg open)
    - [X] fzf could have an option to directly open the parent (although redundant with highlight option)
    - [X] fzf could have an option to start a new terminal in folder / parent of file
- [ ] improve error handling
    - [ ] use stacktraces https://pkg.go.dev/github.com/pkg/errors#WithStack
    - [ ] incorporate more info when creating errors (e.g. running commands)
- [ ] fix phantom symbols in fzf (see below)
- [ ] add a history file for fzf (is it even possible ?)
- [ ] Improve application provider, with inspiration from the Gnome desktop's extensions
- [X] f.go should accept arguments
    - [X] for the base directory
    - [X] to hide files
    - [X] to hide directories
- [X] add a way to add options in the config file directly
    (the JSON is hard to modify without risking errors)
- [X] combined CLI
- [X] Add build version to avoid version mismatch between the processes

### Phantom symbols in fzf

When presenting the results with fzf, a strange error can appear.
This behavior is not systematic, and seems to be almost random.
Certain entries may have altered text, sometimes only a suffix, that is displayed alongside.
However, the selection works properly (i.e. the read text has no alteration.).
It is possible that another thread/process writes to STDERR at the same time as fzf, thus mixing the output.
This bug doesn't seem to appear when no selection is performed.
